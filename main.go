package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/moleculer-go/store/sqlite"

	"github.com/moleculer-go/gateway"
	"github.com/moleculer-go/gateway/websocket"
	"github.com/moleculer-go/moleculer"
	"github.com/moleculer-go/moleculer/broker"
	"github.com/moleculer-go/moleculer/cli"
	"github.com/moleculer-go/moleculer/payload"
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

type MockContent struct {
	StatusCode int
	Content    string
	TargetUrl  string
}

func getCorrelationID(c moleculer.BrokerContext, headerField string, r *http.Request) string {

	c.Logger().Debug("getCorrelationID() - headerField ", headerField)

	correlationID := r.Header[headerField]
	if len(correlationID) > 0 && correlationID[0] != "" {
		c.Logger().Debug("correlation id found: ", correlationID[0])
		return correlationID[0]
	}

	for name, value := range r.Header {
		c.Logger().Debug("header ", name, " value: ", value)
		if strings.Contains(strings.ToLower(name), "correlation") {
			c.Logger().Debug("correlation id found: ", value[0])
			return value[0]
		}
	}
	return "not-found"
}

func pathKey(path string) string {
	path = strings.Replace(path, "/", "_", -1)
	path = strings.Replace(path, ".", "_", -1)
	if path[:1] == "_" {
		return path[1:]
	}
	return path
}

// calcPaths calculates the possible paths for each mock file
// example: folder_file_1234 -> ${id}_file_1234, folder_${id}_1234, folder_file_${id}
func calcPaths(pathKey string) []string {
	paths := []string{pathKey}
	parts := strings.Split(pathKey, "_")
	for i, _ := range parts {
		parts[i] = "${id}"
		paths = append(paths, strings.Join(parts, "_"))
		parts = strings.Split(pathKey, "_")
	}
	return paths
}

// findFile for a given folder and path it will find a file that matchs the path considering id placeholders.
func findFile(folder, path string) string {
	paths := calcPaths(path)
	for _, name := range paths {
		file := folder + "/" + name + ".mock"
		if _, err := os.Stat(file); err == nil {
			return file
		}
	}
	return "no file found for path: " + path
}

func getMockConfig(c moleculer.BrokerContext, w http.ResponseWriter, mockFolder, pathKey string) *MockContent {
	path := findFile(mockFolder, pathKey)
	mock := MockContent{StatusCode: 200, Content: "empty!"}
	fileContents, err := ioutil.ReadFile(path)
	if err == nil {
		fmt.Println("Mock file being used: " + path)
		err = json.Unmarshal(fileContents, &mock)
		if err != nil {
			mock.Content = "Error reading JSON mock file: " + path + ". Details: " + err.Error()
		} else {
			c.Logger().Debug("Mock config loaded: mock.StatusCode: ", mock.StatusCode, " mock.Content: ", mock.Content)
		}
	} else {
		mock.Content = "Warning! Proxy Sink could not find the mock configuration file from : " + path + ". Details: " + err.Error()
	}
	// w.WriteHeader(mock.StatusCode)
	// w.Write([]byte(mock.Content))
	return &mock
}

func extractPayload(c moleculer.BrokerContext, r *http.Request) []byte {
	ct := r.Header.Get("Content-Type")
	if ct != "" {
		mediaType, params, err := mime.ParseMediaType(ct)
		if err != nil {
			return []byte("Error parsing media type. Error: " + err.Error())
		}
		if strings.HasPrefix(mediaType, "multipart/") {
			mr := multipart.NewReader(r.Body, params["boundary"])
			for {
				p, err := mr.NextPart()
				if err == io.EOF {
					break
				}
				if err != nil {
					c.Logger().Error("extractPayload() - Error getting multi-part - error: ", err)
				}
				payload, err := ioutil.ReadAll(p)
				if err != nil {
					payload = []byte("Error reading body. Error: " + err.Error())
				}
				return payload
			}
		}
	}
	payload, err := ioutil.ReadAll(r.Body)
	if err != nil {
		payload = []byte("Error reading body. Error: " + err.Error())
	}
	return payload
}

// handleRequest store the incoming request, load mock config and return mock content or invoke real system.
func handleRequest(c moleculer.BrokerContext, w http.ResponseWriter, r *http.Request, correlationID, mockFolder string) {
	c.Logger().Debug("handleRequest() - correlationID ", correlationID)

	payload := extractPayload(c, r)
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

	mock := getMockConfig(c, w, mockFolder, pathKey(path))

	if mock.TargetUrl != "" {
		status, content := invokeTarget(c, r, mock)
		w.WriteHeader(status)
		w.Write(content)
		//TODO write results here
	} else {
		w.WriteHeader(mock.StatusCode)
		w.Write([]byte(mock.Content))
	}
}

//sinkAndProxy transparent proxy path
func invokeTarget(c moleculer.BrokerContext, r *http.Request, mock *MockContent) (statusCode int, content []byte) {
	endpoint, err := url.Parse(mock.TargetUrl)
	if err != nil {
		c.Logger().Error("Could not parse target Endpoint: ", mock.TargetUrl)
	}

	r.RequestURI = ""
	r.URL.Host = endpoint.Host
	r.URL.Scheme = endpoint.Scheme
	r.Header.Set("X-Forwarded-Host", r.Header.Get("Host"))
	r.Host = endpoint.Host

	response, err := http.DefaultClient.Do(r)
	if err != nil {
		msg := fmt.Sprint("Error trying to call target endpoint! - Error: ", err)
		c.Logger().Error(content)
		statusCode = response.StatusCode
		content = []byte(msg)
	}

	targetContents, err := ioutil.ReadAll(response.Body)
	if err != nil {
		// handle error here
		msg := fmt.Sprint("Error trying to read body from the target endpoint! - Error: ", err)
		c.Logger().Error(content)
		statusCode = 500
		content = []byte(msg)
	} else {
		content = targetContents
		statusCode = response.StatusCode
	}
	return statusCode, content
}

var proxySink = moleculer.ServiceSchema{
	Name: "proxy-sink",
	Started: func(c moleculer.BrokerContext, svc moleculer.ServiceSchema) {
		settings := payload.New(svc.Settings)
		mode := settings.Get("mode").String()
		port := settings.Get("port").Int()
		correlationIDHeader := settings.Get("correlation-header").String()
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			correlationID := getCorrelationID(c, correlationIDHeader, r)
			mocks := svc.Settings["mocks"].(string)
			handleRequest(c, w, r, correlationID, mocks)
		})
		go http.ListenAndServe(":"+strconv.Itoa(port), nil)
		c.Logger().Info("Proxy Sink started - listening on port: "+strconv.Itoa(port)+" mode: ", mode, " correlationId Header: ", correlationIDHeader)
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
