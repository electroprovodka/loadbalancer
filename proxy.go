package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

type Server struct {
	scheme string
	host   string
	port   string
}

func (s Server) URL() string {
	return s.scheme + "://" + s.host + ":" + s.port
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

func NewProxy(config *Config) (*Proxy, error) {
	upstreams := make([]Upstream, 0)
	for _, cu := range config.Upstreams {
		var servers []Server
		for _, sURL := range cu.Servers {
			servers = append(servers, Server{scheme: sURL.Scheme, host: sURL.Hostname(), port: sURL.Port()})
		}
		c := cu.Condition
		cond := GetCondition(c.Type, c.Key, c.Value)
		if cond == nil {
			return nil, fmt.Errorf("Can not parse condition for %s upstream", cu.Name)
		}

		upstreams = append(upstreams, Upstream{name: cu.Name, servers: servers, cond: cond})
	}
	return &Proxy{upstreams}, nil
}
