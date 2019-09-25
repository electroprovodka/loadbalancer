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
	PORT = "8081"
)

type Server struct {
	schema string
	host   string
	port   string
}

func (s Server) URL() string {
	return s.schema + "://" + s.host + ":" + s.port
}

// TODO: consider better name
type Upstream struct {
	servers []Server
	idx     int
	name    string
}

func (u *Upstream) GetServer() Server {
	// TODO: find better way for round robin
	s := u.servers[u.idx%len(u.servers)]
	u.idx++
	return s
}

var servers = []Server{
	{"http", "127.0.0.1", "3000"},
	{"http", "127.0.0.1", "4000"},
}

var upstream = Upstream{servers: servers}

func prepareRequest(r *http.Request) (*http.Request, error) {
	// TODO: What is context here?
	// TODO: Should we copy request or we can just change the existing one
	fwd := r.Clone(r.Context())

	// TODO: consider better name
	server := upstream.GetServer()
	// TODO: check this is the way how url should be constructed
	url, err := url.Parse(server.URL() + r.RequestURI)
	if err != nil {
		fmt.Println("URL error", err)
		// TODO: wrap error and process above
		return nil, err
	}
	fmt.Println("Original url", r.RequestURI, "New url", url)
	fwd.URL = url
	fwd.Host = server.URL()
	fwd.RequestURI = ""

	// TODO: Remove/update all required headers
	// TODO: Set all required headers
	fwd.Header.Set("X-Forwarded-For", r.RemoteAddr)
	return fwd, nil
}

func writeResponse(w http.ResponseWriter, resp *http.Response) error {
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Response body error", err)
		return err
	}

	w.WriteHeader(resp.StatusCode)

	for k, v := range resp.Header {
		// TODO: headers filtering
		// TODO: values processing
		w.Header().Add(k, strings.Join(v, ";"))
	}

	// TODO: write directly?
	_, err = io.Copy(w, bytes.NewBuffer(body))
	if err != nil {
		fmt.Println("Body write error", err)
		return err
	}

	return nil
}

func handler(w http.ResponseWriter, r *http.Request) {
	// TODO: return error to user

	fwd, err := prepareRequest(r)
	if err != nil {
		// TODO: add logging
		return
	}

	// TODO: timeouts?
	client := http.Client{}

	resp, err := client.Do(fwd)
	if err != nil {
		fmt.Println("Request error", err)
		return
	}
	fmt.Println("Response", resp.StatusCode)
	writeResponse(w, resp)
}

func main() {
	// TODO: multiple targets
	// TODO: redirect rules
	// TODO: file config
	// TODO: hot reload
	// TODO: graceful shutdown
	// TODO: signals processing
	// TODO: logging
	// TODO: metrics?
	// TODO: healthchecks?
	// TODO: targets autodiscovery?
	// TODO: tests

	// TODO: change HandleFunc?
	http.HandleFunc("/", handler)
	// TODO: use cusom handler
	err := http.ListenAndServe(":"+PORT, nil)
	if err != nil {
		log.Fatal(err.Error())
	}
}
