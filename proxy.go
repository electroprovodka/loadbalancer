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

	"github.com/pkg/errors"
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
	servers []*Server
	idx     int
	name    string
}

func (u *Upstream) getServer() (*Server, error) {
	if len(u.servers) == 0 {
		return nil, errors.New("Empty upstream servers list")
	}
	// TODO: find better way for round robin
	s := u.servers[u.idx%len(u.servers)]
	u.idx++
	return s, nil
}

type Proxy struct {
	us []*Upstream
}

func (p *Proxy) getUpstream(r *http.Request) (*Upstream, error) {
	for idx := range p.us {
		// Retrieve value directly without copy
		if p.us[idx].cond.Check(r) {
			return p.us[idx], nil
		}
	}
	return nil, errors.New("No upstream matches the provided request")
}

func (p *Proxy) prepareRequest(r *http.Request) (*http.Request, error) {
	// TODO: What is context here?
	// TODO: Should we copy request or we can just change the existing one
	fwd := r.Clone(r.Context())

	// TODO: consider better name
	u, err := p.getUpstream(r)
	if err != nil {
		return nil, errors.Wrap(err, "Error retrieving upstream")
	}
	server, err := u.getServer()
	if err != nil {
		return nil, errors.Wrapf(err, "Can not get server for upstream %s", u.name)
	}
	// TODO: check this is the way how url should be constructed
	url, err := url.Parse(server.URL() + r.RequestURI)
	if err != nil {
		return nil, errors.Wrapf(err, "Can not parse the url %s", server.URL()+r.RequestURI)
	}

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
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "Error reading the upstream response")
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
		return errors.Wrap(err, "Error writing the response")
	}

	return nil
}

func (p *Proxy) handle(w http.ResponseWriter, r *http.Request) {
	// TODO: return error to user

	fwd, err := p.prepareRequest(r)
	if err != nil {
		log.Println(err)
		http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		return
	}

	// TODO: timeouts?
	client := http.Client{}

	resp, err := client.Do(fwd)
	if err != nil {
		log.Println("Error during making upstream request", err)
		http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
		return
	}
	fmt.Println("Response", resp.StatusCode)
	err = p.writeResponse(w, resp)
	if err != nil {
		log.Println(err)
		http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		return
	}
}

func NewProxy(config *Config) (*Proxy, error) {
	upstreams := make([]*Upstream, 0)
	for _, cu := range config.Upstreams {
		var servers []*Server
		for _, sURL := range cu.Servers {
			servers = append(servers, &Server{scheme: sURL.Scheme, host: sURL.Hostname(), port: sURL.Port()})
		}
		c := cu.Condition
		cond := GetCondition(c.Type, c.Key, c.Value)
		if cond == nil {
			return nil, errors.Errorf("Can not parse condition for %s upstream", cu.Name)
		}

		upstreams = append(upstreams, &Upstream{name: cu.Name, servers: servers, cond: cond})
	}
	return &Proxy{upstreams}, nil
}
