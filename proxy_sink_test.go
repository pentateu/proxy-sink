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

	var bkr *broker.ServiceBroker
	BeforeSuite(func() {
		bkr = broker.New(&moleculer.Config{
			LogLevel: logLevel,
		})
		proxySink.Settings = M{"port": "8387", "mode": "sink", "mocks": "./mocks", "correlationIdHeader": "Correlation-Id"}
		bkr.Publish(gatewaySvc, sink, proxySink)
		bkr.Start()
		time.Sleep(time.Millisecond)
	})

	AfterSuite(func() {
		bkr.Stop()
	})

	It("should sink request contets to database and respond with mock response", func(done Done) {
		rq, err := http.NewRequest("GET", "http://localhost:8387/v2/service/endpoint/param?otherParam=ParamValue&otherParam2=ParamValue2", nil)
		Expect(err).Should(BeNil())
		rq.Header["Correlation-Id"] = []string{"345678oo"}
		rq.Header["Other-Header"] = []string{"some value"}
		rs, err := http.DefaultClient.Do(rq)
		Expect(err).Should(BeNil())
		Expect(rs.Status).Should(Equal("201 Created"))

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

	It("should respond with mock which has ${id} placeholder in the name", func(done Done) {
		correlationId := "1234-dynamix"
		rq, err := http.NewRequest("GET", "http://localhost:8387/v2/dynamic/endpoint_12121212", nil)
		Expect(err).Should(BeNil())
		rq.Header["Correlation-Id"] = []string{correlationId}
		rq.Header["Other-Header"] = []string{"some value"}
		rs, err := http.DefaultClient.Do(rq)
		Expect(err).Should(BeNil())
		Expect(rs.Status).Should(Equal("203 Non-Authoritative Information"))
		bts, err := ioutil.ReadAll(rs.Body)
		str := string(bts)
		fmt.Println("str ", str)
		Expect(str).Should(Equal("I have a ${id} placeholder in my name :) which means I can math files like: v2_service_endpoint_1234 or v2_service_endpoint_someOtherId and etc"))

		//check if the results are in the database
		rs, err = http.Get("http://localhost:3100/sink/find?search=" + correlationId + "&searchFields=correlationID")
		Expect(err).Should(BeNil())
		Expect(rs.Status).Should(Equal("200 OK"))
		bts, err = ioutil.ReadAll(rs.Body)
		str = string(bts)
		fmt.Println("str: ", str)
		Expect(str).Should(Equal(`[{"correlationID":"` + correlationId + `","headers":"map[Accept-Encoding:[gzip] Correlation-Id:[` + correlationId + `] Other-Header:[some value] User-Agent:[Go-http-client/1.1]]","id":"2","path":"/v2/dynamic/endpoint_12121212","payload":null}]`))

		close(done)
	}, 2)

	It("calc paths should return 3 possible paths", func() {
		paths := calcPaths("folder_file_1234")
		Expect(paths).Should(HaveLen(4))
		Expect(paths).Should(ContainElement("folder_file_1234"))
		Expect(paths).Should(ContainElement("${id}_file_1234"))
		Expect(paths).Should(ContainElement("folder_${id}_1234"))
		Expect(paths).Should(ContainElement("folder_file_${id}"))
	})

	It("findFile file with dynamic id", func() {
		//folder, _ := filepath.Abs(filepath.Dir(os.Args[0]))
		//fmt.Println("folder: ", folder)
		folder := "./mocks"
		path := "v2_dynamic_endpoint_1234"
		file := findFile(folder, path)
		Expect(file).Should(HaveSuffix("v2_dynamic_endpoint_${id}.mock"))

		path = "v2_dynamic_1234_bla"
		file = findFile(folder, path)
		Expect(file).Should(HaveSuffix("v2_dynamic_${id}_bla.mock"))

		path = "v2_1234555555_endpoint_bla"
		file = findFile(folder, path)
		Expect(file).Should(HaveSuffix("v2_${id}_endpoint_bla.mock"))
	})

	XIt("should invoke a target Url and return its contents", func(done Done) {
		correlationId := "1234-target"
		rq, err := http.NewRequest("GET", "http://localhost:8387/v2/endpoint/with/target", nil)
		Expect(err).Should(BeNil())
		rq.Header["Correlation-Id"] = []string{correlationId}
		rq.Header["Other-Header"] = []string{"some value"}
		rs, err := http.DefaultClient.Do(rq)
		Expect(err).Should(BeNil())
		Expect(rs.Status).Should(Equal("203 Non-Authoritative Information"))
		bts, err := ioutil.ReadAll(rs.Body)
		str := string(bts)
		fmt.Println("str ", str)
		Expect(str).Should(Equal("I have a ${id} placeholder in my name :) which means I can math files like: v2_service_endpoint_1234 or v2_service_endpoint_someOtherId and etc"))

		//check if the results are in the database
		rs, err = http.Get("http://localhost:3100/sink/find?search=" + correlationId + "&searchFields=correlationID")
		Expect(err).Should(BeNil())
		Expect(rs.Status).Should(Equal("200 OK"))
		bts, err = ioutil.ReadAll(rs.Body)
		str = string(bts)
		fmt.Println("str: ", str)
		Expect(str).Should(Equal(`[{"correlationID":"` + correlationId + `","headers":"map[Accept-Encoding:[gzip] Correlation-Id:[` + correlationId + `] Other-Header:[some value] User-Agent:[Go-http-client/1.1]]","id":"3","path":"/v2/dynamic/endpoint_12121212","payload":null}]`))

		close(done)
	}, 2)

})
