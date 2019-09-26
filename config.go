package main

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"regexp"
	"strings"

	"gopkg.in/yaml.v2"
)

// FileConfig accepts data from yml config file
type FileConfig struct {
	Port int

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
	Value interface{}
}

type Upstr struct {
	Name      string
	Servers   []url.URL
	Condition Cond
}

type Config struct {
	Port      int
	Upstreams []Upstr
}

func (fc FileConfig) validate() (*Config, error) {
	conf := Config{}

	// TODO: check for 0/to big port
	// Allow int instead of strings
	if fc.Port <= 0 || fc.Port > 65536 {
		return nil, fmt.Errorf("Invalid Port value %s", fc.Port)
	}
	conf.Port = fc.Port

	// TODO: validate host name
	// TODO: what rules should we apply?
	// No path?
	// Scheme is optional?
	for uname, ups := range fc.Upstreams {
		var sURLs []url.URL

		if len(ups.Servers) == 0 {
			return nil, fmt.Errorf("Upstream %s should have at least one server", uname)
		}

		for _, s := range ups.Servers {
			// TODO: find a better way to do this
			if !strings.HasPrefix(s, "http") {
				s = "http://" + s
			}
			u, err := url.Parse(s)
			if err != nil || u.Hostname() == "" || u.Port() == "" {
				return nil, fmt.Errorf("%s is not a valid host for upstream %s : %s", s, uname, err)
			}
			sURLs = append(sURLs, *u)
		}

		// TODO: check if type is in the known list
		cond := ups.Condition
		if cond.Type == "" {
			return nil, fmt.Errorf("Upstream %s condition is missing the type field", uname)
		}

		if cond.Value == "" {
			return nil, fmt.Errorf("Upstream %s condition is missing the value field", uname)
		}

		ct, err := GetConditionType(cond.Type)
		if err != nil {
			// TODO: wrap with upstream info
			return nil, fmt.Errorf("Invalid condition type for upstream %s: %s", uname, err)
		}
		parsedCond := Cond{Type: ct, Key: cond.Key, Value: cond.Value}

		if ct == HeaderCond && cond.Key == "" {
			return nil, fmt.Errorf("Upstream %s condition is missing the key field", uname)
		}

		if ct == RegexpCond {
			reg, err := regexp.Compile(cond.Value)
			if err != nil {
				return nil, fmt.Errorf("Upstream %s condition value is not a valid regexp", uname)
			}
			// Replacing the string value with parsed regexp
			parsedCond.Value = reg
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
		// TODO: wrap error
		return nil, err
	}

	var fc FileConfig
	err = yaml.Unmarshal(source, &fc)
	if err != nil {
		// TODO: wrap err
		return nil, err
	}

	fmt.Println(fc)

	config, err := fc.validate()
	if err != nil {
		return nil, err
	}
	return config, nil
}
