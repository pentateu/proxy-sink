package main

import (
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/moleculer-go/store/sqlite"

	"github.com/moleculer-go/gateway"
	"github.com/moleculer-go/gateway/websocket"
	"github.com/moleculer-go/moleculer"
	"github.com/moleculer-go/moleculer/broker"
	"github.com/moleculer-go/moleculer/cli"
	"github.com/moleculer-go/store"
	"github.com/spf13/cobra"
)

type M map[string]interface{}

var gatewaySvc = &gateway.HttpService{
	Settings: map[string]interface{}{},
	Mixins: []gateway.GatewayMixin{&websocket.WebSocketMixin{
		Mixins: []websocket.SocketMixin{
			&websocket.EventsMixin{},
		},
	}},
}

var sink = moleculer.ServiceSchema{
	Name: "sink",
	Mixins: []moleculer.Mixin{store.Mixin(&sqlite.Adapter{
		Table: "sinkStore",
		Columns: []sqlite.Column{
			{
				Name: "correlationID",
				Type: "string",
			},
			{
				Name: "headers",
				Type: "string",
			},
			{
				Name: "path",
				Type: "string",
			},
			{
				Name: "payload",
				Type: "string",
			},
		},
	})},
}

func getCorrelationID(headerField string, r *http.Request) string {
	correlationID := r.Header[headerField]
	if len(correlationID) > 0 && correlationID[0] != "" {
		return correlationID[0]
	}
	return "not-found"
}

func pathKey(path string) string {
	name := strings.Replace(path, "/", "_", -1)
	name = name[1:] + ".mock"
	return name
}

func respondWithMock(w http.ResponseWriter, mockFolder, pathKey string) {
	path := mockFolder + "/" + pathKey
	contents, err := ioutil.ReadFile(path)
	if err != nil {
		contents = []byte("Error reading file from : " + path + ". Error: " + err.Error())
	}
	w.Write(contents)
}

// sinkAndMockResponse store the incoming request and reponse with a mock response
func sinkAndMockResponse(c moleculer.BrokerContext, w http.ResponseWriter, r *http.Request, correlationID, mockFolder string) {
	payload, err := ioutil.ReadAll(r.Body)
	if err != nil {
		payload = []byte("Error reading body. Error: " + err.Error())
	}
	path := r.URL.Path
	headers := r.Header

	record := <-c.Call("sink.create", M{
		"path":          path,
		"payload":       payload,
		"correlationID": correlationID,
		"headers":       headers,
	})
	if record.IsError() {
		w.Write([]byte("Error saving request. Error: " + record.Error().Error()))
		return
	}
	respondWithMock(w, mockFolder, pathKey(path))
}

//sinkAndProxy transparent proxy path
func sinkAndProxy(c moleculer.BrokerContext, w http.ResponseWriter, r *http.Request, correlationID string) {
	//not required for now
}

var proxySink = moleculer.ServiceSchema{
	Name: "proxy-sink",
	Started: func(c moleculer.BrokerContext, svc moleculer.ServiceSchema) {
		mode := svc.Settings["mode"].(string)
		port := svc.Settings["port"].(int)
		correlationIdHeader := svc.Settings["correlation-header"].(string)
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			correlationID := getCorrelationID(correlationIdHeader, r)
			if mode == "sink" {
				mocks := svc.Settings["mocks"].(string)
				sinkAndMockResponse(c, w, r, correlationID, mocks)
			} else {
				sinkAndProxy(c, w, r, correlationID)
			}
		})
		go http.ListenAndServe(":"+strconv.Itoa(port), nil)
		c.Logger().Info("Proxy Sink started - listening on port: "+strconv.Itoa(port)+" mode: ", mode, " correlationId Header: ", correlationIdHeader)
	},
}

func main() {
	cli.Start(
		&moleculer.Config{LogLevel: "debug"},
		func(broker *broker.ServiceBroker, cmd *cobra.Command) {
			broker.Publish(gatewaySvc, sink, proxySink)
			broker.Start()
		})
}
