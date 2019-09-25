package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

const (
	PORT = "8081"
)

type RequestUpdater interface {
	UpdateRequest(s Server, r *http.Request) error
}

type Server struct {
	schema string
	host   string
	port   string
}

func (s Server) URL() string {
	return s.schema + "://" + s.host + ":" + s.port
}

// TODO: add regex
// TODO: add header
// TODO: add other checks?
// TODO: add default type that matches all requests
// TODO: add url rewrites

// NOTE: is there any sence for "host"? since this LB should be an entrypoint there should be only one host
type Condition interface {
	Check(r *http.Request) bool
}

type PrefixCondition struct {
	prefix string
}

func (c *PrefixCondition) Check(r *http.Request) bool {
	return strings.HasPrefix(r.RequestURI, c.prefix)
}

type RegexpCondition struct {
	reg regexp.Regexp
}

func (c *RegexpCondition) Check(r *http.Request) bool {
	return c.reg.MatchString(r.RequestURI)
}

type HasHeaderCondition struct {
	header string
}

func (c *HasHeaderCondition) Check(r *http.Request) bool {
	return len(r.Header.Get(c.header)) != 0
}

type HeaderValueCondition struct {
	header string
	value  string
}

func (c *HeaderValueCondition) Check(r *http.Request) bool {
	return r.Header.Get(c.header) == c.value
}

// TODO: consider better name
// TODO: add weights?
type Upstream struct {
	cond    Condition
	servers []Server
	idx     int
	name    string
}

func (u *Upstream) GetServer() (*Server, error) {
	if len(u.servers) == 0 {
		// TODO: better error
		// TODO: not nil response
		return nil, errors.New("Empty upstream servers list")
	}
	// TODO: find better way for round robin
	s := u.servers[u.idx%len(u.servers)]
	u.idx++
	return &s, nil
}

type Proxy struct {
	us []Upstream
}

func (p *Proxy) GetUpstream(r *http.Request) (*Upstream, error) {
	for _, u := range p.us {
		// TODO: more complex check type?
		if u.cond.Check(r) {
			fmt.Println("Found upstream", u.servers)
			return &u, nil
		}
	}
	// TODO: provide a better error
	// TODO: not nil response?
	return nil, errors.New("Can not find upstream")
}

func (p *Proxy) prepareRequest(r *http.Request) (*http.Request, error) {
	// TODO: What is context here?
	// TODO: Should we copy request or we can just change the existing one
	fwd := r.Clone(r.Context())

	// TODO: consider better name
	u, err := p.GetUpstream(r)
	if err != nil {
		fmt.Println("Upstream error", err)
		return nil, err
	}
	server, err := u.GetServer()
	if err != nil {
		fmt.Println("Upstrean server error", err)
		return nil, err
	}
	// TODO: check this is the way how url should be constructed
	url, err := url.Parse(server.URL() + r.RequestURI)
	if err != nil {
		fmt.Println("URL error", err)
		// TODO: wrap error and process above
		return nil, err
	}
	fmt.Println("Original url", r.RequestURI, "New url", url)
	// TODO: update other request fields if needed
	fwd.URL = url
	fwd.Host = server.URL()
	fwd.RequestURI = ""

	// TODO: Remove/update all required headers
	// TODO: Set all required headers
	fwd.Header.Set("X-Forwarded-For", r.RemoteAddr)
	return fwd, nil
}

func (p *Proxy) writeResponse(w http.ResponseWriter, resp *http.Response) error {
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

func (p *Proxy) handle(w http.ResponseWriter, r *http.Request) {
	// TODO: return error to user

	fwd, err := p.prepareRequest(r)
	if err != nil {
		// TODO: add logging
		fmt.Println("Request preparation error", err)
		http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		return
	}

	// TODO: timeouts?
	client := http.Client{}

	resp, err := client.Do(fwd)
	if err != nil {
		fmt.Println("Request error", err)
		http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
		return
	}
	fmt.Println("Response", resp.StatusCode)
	err = p.writeResponse(w, resp)
	if err != nil {
		fmt.Println("Response write error", err)
		http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		return
	}
}

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

	var servers1 = []Server{
		{"http", "127.0.0.1", "3000"},
	}

	servers2 := []Server{
		{"http", "127.0.0.1", "4000"},
	}

	condition1 := HeaderValueCondition{header: "Custom", value: "Header"}
	condition2 := PrefixCondition{prefix: "/jokes"}

	var upstreams = []Upstream{
		{servers: servers1, cond: &condition1},
		{servers: servers2, cond: &condition2},
	}

	proxy := Proxy{us: upstreams}

	// TODO: change HandleFunc?
	http.HandleFunc("/", proxy.handle)
	// TODO: use cusom handler
	err := http.ListenAndServe(":"+PORT, nil)
	if err != nil {
		log.Fatal(err.Error())
	}
}
