package main

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

// TODO: consider switching to one config instead of separate FileConfig and Config
// FileConfig accepts data from yml config file
type FileConfig struct {
	Port               int
	ServerReadTimeout  uint
	ServerWriteTimeout uint
	ProxyTimeout       uint

	Upstreams map[string]struct {
		Servers []string

		Condition struct {
			Type  string
			Key   string
			Value string
		}
	}
}

type Cond struct {
	Type  ConditionType
	Key   string
	Value string
}

type Upstr struct {
	Name      string
	Servers   []url.URL
	Condition Cond
}

type Config struct {
	Port               int
	ServerReadTimeout  uint
	ServerWriteTimeout uint
	ProxyTimeout       uint
	Upstreams          []Upstr
}

func (fc FileConfig) validate() (*Config, error) {
	conf := Config{}

	// TODO: check for 0/to big port
	if fc.Port <= 0 || fc.Port > 65536 {
		return nil, errors.Errorf("Invalid Port value %v", fc.Port)
	}
	conf.Port = fc.Port
	// TODO: find a better way to copy this values
	conf.ServerReadTimeout = fc.ServerReadTimeout
	conf.ServerWriteTimeout = fc.ServerWriteTimeout
	conf.ProxyTimeout = fc.ProxyTimeout

	// TODO: validate host name
	// TODO: what rules should we apply?
	// No path?
	// Scheme is optional?
	for uname, ups := range fc.Upstreams {
		var sURLs []url.URL

		if len(ups.Servers) == 0 {
			return nil, errors.Errorf("Upstream %s should have at least one server", uname)
		}

		for _, s := range ups.Servers {
			// TODO: find a better way to do this
			if !strings.HasPrefix(s, "http") {
				s = "http://" + s
			}
			u, err := url.Parse(s)
			if err != nil || u.Hostname() == "" || u.Port() == "" {
				return nil, errors.Errorf("%s is not a valid host for upstream %s : %s", s, uname, err)
			}
			sURLs = append(sURLs, *u)
		}

		// TODO: check if type is in the known list
		cond := ups.Condition
		if cond.Type == "" {
			return nil, errors.Errorf("Upstream %s condition is missing the type field", uname)
		}

		if cond.Value == "" {
			return nil, errors.Errorf("Upstream %s condition is missing the value field", uname)
		}

		ct, err := GetConditionType(cond.Type)
		if err != nil {
			// TODO: wrap with upstream info
			return nil, errors.Errorf("Invalid condition type for upstream %s: %s", uname, err)
		}
		parsedCond := Cond{Type: ct, Key: cond.Key, Value: cond.Value}

		if ct == HeaderCond && cond.Key == "" {
			return nil, errors.Errorf("Upstream %s condition is missing the key field", uname)
		}

		if ct == RegexpCond {
			_, err := regexp.Compile(cond.Value)
			if err != nil {
				return nil, errors.Errorf("Upstream %s condition value is not a valid regexp", uname)
			}
		}

		upstr := Upstr{Name: uname, Servers: sURLs, Condition: parsedCond}
		conf.Upstreams = append(conf.Upstreams, upstr)
	}
	return &conf, nil
}

func ReadConfig(path string) (*Config, error) {
	// TODO: Check the correct way to read files
	source, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, errors.Wrapf(err, "can not read read config %s", path)
	}

	var fc FileConfig
	err = yaml.Unmarshal(source, &fc)
	if err != nil {
		return nil, errors.Wrap(err, "can not read yaml config")
	}

	fmt.Println(fc)

	config, err := fc.validate()
	if err != nil {
		return nil, errors.Wrap(err, "config is not valid")
	}
	return config, nil
}
