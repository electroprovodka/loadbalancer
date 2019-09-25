package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
)

const (
	PORT       = "8081"
	SCHEME     = "http"
	SERVER     = "127.0.0.1:3000"
	SERVER_URL = SCHEME + "://" + SERVER
)

func handler(w http.ResponseWriter, r *http.Request) {
	// TODO: What is context here?
	// TODO: Should we copy request or we can just change the existing one
	fwd := r.Clone(r.Context())
	// TODO: check this is the way how url should be constructed
	url, err := url.Parse(SERVER_URL + r.RequestURI)
	if err != nil {
		fmt.Println("URL error", err)
		return
	}
	fmt.Println("Original url", r.RequestURI, "New url", url)
	fwd.URL = url
	fwd.Host = SERVER_URL
	fwd.RequestURI = ""
	// TODO: Set all required headers
	fwd.Header.Set("X-Forwarded-For", r.RemoteAddr)
	// TODO: Remove/update all required headers

	// TODO: timeouts?
	client := http.Client{}

	resp, err := client.Do(fwd)
	if err != nil {
		fmt.Println("Request error", err)
		return
	}
	fmt.Println("Response", resp.StatusCode)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Response body error", err)
		return
	}

	w.WriteHeader(resp.StatusCode)

	for k, v := range resp.Header {
		// TODO: headers filtering
		// TODO: values processing
		w.Header().Add(k, strings.Join(v, ";"))
	}

	// TODO: write directly?
	io.Copy(w, bytes.NewBuffer(body))
}

func main() {
	// TODO: multiple targets
	// TODO: file config
	// TODO: hot reload
	// TODO: redirect rules
	// TODO: graceful shutdown
	// TODO: signals processing
	// TODO: logging
	// TODO: metrics?
	// TODO: healthchecks?
	// TODO: targets autodiscovery?

	// TODO: change HandleFunc?
	http.HandleFunc("/", handler)
	// TODO: use cusom handler
	err := http.ListenAndServe(":"+PORT, nil)
	if err != nil {
		log.Fatal(err.Error())
	}
}
