package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/moleculer-go/moleculer"
	"github.com/moleculer-go/moleculer/broker"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ProxySink", func() {

	logLevel := "error"

	It("should sink request contets to database and respond with mock response", func(done Done) {
		bkr := broker.New(&moleculer.Config{
			LogLevel: logLevel,
		})
		proxySink.Settings = M{"port": "8387", "mode": "sink", "mocks": "./mocks", "correlationIdHeader": "Correlation-Id"}
		bkr.Publish(gatewaySvc, sink, proxySink)
		bkr.Start()
		time.Sleep(time.Millisecond)

		rq, err := http.NewRequest("GET", "http://localhost:8387/v2/service/endpoint/param?otherParam=ParamValue&otherParam2=ParamValue2", nil)
		Expect(err).Should(BeNil())
		rq.Header["Correlation-Id"] = []string{"345678oo"}
		rq.Header["Other-Header"] = []string{"some value"}
		rs, err := http.DefaultClient.Do(rq)
		Expect(err).Should(BeNil())
		Expect(rs.Status).Should(Equal("200 OK"))
		bts, err := ioutil.ReadAll(rs.Body)
		str := string(bts)
		fmt.Println("str ", str)
		Expect(str).Should(Equal(`mock content!`))

		//check if the results are in the database
		rs, err = http.Get("http://localhost:3100/sink/find?search=345678oo&searchFields=correlationID")
		Expect(err).Should(BeNil())
		Expect(rs.Status).Should(Equal("200 OK"))
		bts, err = ioutil.ReadAll(rs.Body)
		str = string(bts)
		Expect(str).Should(Equal(`[{"correlationID":"345678oo","headers":"map[Accept-Encoding:[gzip] Correlation-Id:[345678oo] Other-Header:[some value] User-Agent:[Go-http-client/1.1]]","id":"1","path":"/v2/service/endpoint/param","payload":null}]`))

		close(done)
	}, 2)

})
