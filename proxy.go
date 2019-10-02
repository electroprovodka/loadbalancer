package main

import (
	"bytes"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/pkg/errors"
)

type Server struct {
	scheme string
	host   string
	port   string
}

// TODO: add weights?
// TODO: add specific timeouts for each upstream?
type Upstream struct {
	cond    Condition
	servers []*Server
	idx     int
	name    string
}

// Proxy is struct for managing the redirect settings
type Proxy struct {
	us           []*Upstream
	proxyTimeout time.Duration
}

func (s Server) URL() string {
	return s.scheme + "://" + s.host + ":" + s.port
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

func (p *Proxy) getClient() (http.Client, error) {
	// TODO: Read/Write buffers sizes
	// TODO: setup more timeouts if needed https://blog.cloudflare.com/the-complete-guide-to-golang-net-http-timeouts/
	// TODO: build manual timeout with context?
	client := http.Client{
		// NOTE: this timeout includes the response body read
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

// Copy of the same list from  https://golang.org/src/net/http/httputil/reverseproxy.go
// Headers that are used for the particular hop in the end-to-end connection between client and backend
// We remove them b/c we intercept the connection and for client we are backend
var hopHeaders = []string{
	"Connection",
	"Proxy-Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Te",
	"Trailer",
	"Transfer-Encoding",
	"Upgrade",
}

// Copy of the same function from https://golang.org/src/net/http/httputil/reverseproxy.go
func removeConnectionHeaders(h http.Header) {
	for _, f := range h["Connection"] {
		for _, sf := range strings.Split(f, ",") {
			if sf = strings.TrimSpace(sf); sf != "" {
				h.Del(sf)
			}
		}
	}
}

// Copy of the same functionality from https://golang.org/src/net/http/httputil/reverseproxy.go:212
func removeHopByHopHeaders(h http.Header) {
	// Remove hop-by-hop headers to the backend.
	for _, hn := range hopHeaders {
		h.Del(hn)
	}
}

func (p *Proxy) prepareRequest(r *http.Request) (*http.Request, error) {
	// TODO: context timeouts/values?
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
	url, err := url.Parse(server.URL() + r.URL.RequestURI())
	if err != nil {
		return nil, errors.Wrapf(err, "Can not parse the url %s", server.URL()+r.URL.RequestURI())
	}

	// TODO: log

	fwd.URL = url

	// Replace the value of the Host in Request
	// Also no need to update the Host header b/c it's removed from the Request automatically
	// ????? fwd.Host = url.Host

	fwd.RequestURI = ""

	if _, ok := fwd.Header["User-Agent"]; !ok {
		// See https://golang.org/src/net/http/httputil/reverseproxy.go for details
		fwd.Header.Set("User-Agent", "")
	}

	// TODO: allow the connection upgrade
	if callerIP, _, err := net.SplitHostPort(fwd.RemoteAddr); err != nil {
		if prior, ok := fwd.Header["X-Forwarded-For"]; ok {
			callerIP = strings.Join(prior, ",") + "," + callerIP
		}
		// NOTE: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Forwarded
		fwd.Header.Set("X-Forwarded-For", callerIP)
		// TODO: Process the ipv6 correctly
		// TODO: decide which header to use(probably both)
		fwd.Header.Set("Forwarded", "for="+callerIP)
	}

	fwd.Header.Set("X-Forwarded-Proto", fwd.URL.Scheme)

	removeConnectionHeaders(fwd.Header)
	removeHopByHopHeaders(fwd.Header)

	return fwd, nil
}

func (p *Proxy) writeResponse(w http.ResponseWriter, resp *http.Response) error {
	defer resp.Body.Close()

	removeConnectionHeaders(resp.Header)
	removeHopByHopHeaders(resp.Header)

	// TODO: update location
	for k, vv := range resp.Header {
		// TODO: headers filtering
		// TODO: values processing
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}

	w.WriteHeader(resp.StatusCode)

	// NOTE: the err might be a timeout caused by the proxyTimeout for request
	// TODO: read and write in chunks
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "Error reading the upstream response")
	}

	// TODO: write directly?
	_, err = io.Copy(w, bytes.NewBuffer(body))
	if err != nil {
		return errors.Wrap(err, "Error writing the response")
	}

	return nil
}

func (p *Proxy) handle(w http.ResponseWriter, r *http.Request) (int, error) {
	// TODO: process the timeout errors
	client, err := p.getClient()
	if err != nil {
		return http.StatusServiceUnavailable, errors.Wrap(err, "Error during creating request client")
	}

	fwd, err := p.prepareRequest(r)
	if err != nil {
		return http.StatusServiceUnavailable, errors.Wrap(err, "Error during the proxy request preparation")
	}
	log.Println(fwd.URL)
	resp, err := client.Do(fwd)
	if err != nil {
		return http.StatusBadGateway, errors.Wrap(err, "Error during making upstream request")
	}

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
		if e, ok := errors.Cause(err).(net.Error); ok && e.Timeout() {
			status = http.StatusGatewayTimeout
		}
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
