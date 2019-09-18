package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

func main() {

	//Run docker

	//wait
	time.Sleep(time.Second)

	//make a call
	// rs, err := http.Get("http://localhost:8387/v2/service/enpoint/param?key=value")
	// if err != nil {
	// 	panic("Error calling proxy: " + err.Error())
	// }
	// if rs.Status != "200" {
	// 	panic("Expected http status 200")
	// }

	//check results
	rs, err := http.Get("http://localhost:3100/sink/find")
	if err != nil {
		panic("Error calling proxy: " + err.Error())
	}
	str, err := ioutil.ReadAll(rs.Body)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(str))

	//rs, err = http.Get("http://localhost:3100/sink/find?search=345678oo&searchFields=correlationID")

}
