package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

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
// TODO: add specific timeouts for each upstream?
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

// Proxy is struct for managing the redirect settings
type Proxy struct {
	us           []*Upstream
	proxyTimeout time.Duration
}

func (p *Proxy) getClient() (http.Client, error) {
	// TODO: setup more timeouts if needed https://blog.cloudflare.com/the-complete-guide-to-golang-net-http-timeouts/
	// TODO: build manual timeout with context?
	client := http.Client{
		Timeout: p.proxyTimeout,
	}
	return client, nil
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
	// TODO: context timeouts/values?
	ctx := context.Background()
	fwd := r.Clone(ctx)

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

func (p *Proxy) handle(w http.ResponseWriter, r *http.Request) (int, error) {
	fwd, err := p.prepareRequest(r)
	if err != nil {
		log.Println(err)
		http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		return http.StatusServiceUnavailable, err
	}

	client, err := p.getClient()
	if err != nil {
		return http.StatusServiceUnavailable, errors.Wrap(err, "Error during creating request client")
	}

	resp, err := client.Do(fwd)
	if err != nil {
		return http.StatusBadGateway, errors.Wrap(err, "Error during making upstream request")
	}
	fmt.Println("Response", resp.StatusCode)
	err = p.writeResponse(w, resp)
	if err != nil {
		return http.StatusServiceUnavailable, err
	}
	return resp.StatusCode, nil
}

func (p *Proxy) Handle(w http.ResponseWriter, r *http.Request) {
	status, err := p.handle(w, r)
	if err != nil {
		log.Println(err)
		http.Error(w, http.StatusText(status), status)
	}

}

// NewProxy creates new Proxy struct based on the provided Config
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
	timeout := time.Duration(config.ProxyTimeout) * time.Second
	return &Proxy{us: upstreams, proxyTimeout: timeout}, nil
}
