package main

import (
	"fmt"
	"log"
	"net/http"
)

const (
	PORT = "8081"
)

func main() {
	// TODO: multiple targets
	// TODO: redirect rules
	// TODO: file config
	// TODO: hot reload
	// TODO: logging
	// TODO: tests
	// TODO: graceful shutdown
	// TODO: signals processing
	// TODO: https
	// TODO: metrics?
	// TODO: healthchecks?
	// TODO: targets autodiscovery?
	// TODO: API for controlling

	// var servers1 = []Server{
	// 	{"http", "127.0.0.1", "3000"},
	// }

	// servers2 := []Server{
	// 	{"http", "127.0.0.1", "4000"},
	// }

	// condition1 := HeaderValueCondition{header: "Custom", value: "Header"}
	// condition2 := PrefixCondition{prefix: "/jokes"}

	// var upstreams = []Upstream{
	// 	{servers: servers1, cond: &condition1},
	// 	{servers: servers2, cond: &condition2},
	// }

	// proxy := Proxy{us: upstreams}

	config, err := ReadConfig("config.yml")
	if err != nil {
		log.Fatal(err.Error())
		return
	}

	proxy := NewProxy(config)

	// TODO: change HandleFunc?
	http.HandleFunc("/", proxy.handle)
	// TODO: use cusom handler
	err = http.ListenAndServe(fmt.Sprintf(":%d", config.Port), nil)
	if err != nil {
		log.Fatal(err.Error())
	}
}
