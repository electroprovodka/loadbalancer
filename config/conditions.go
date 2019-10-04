package config

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/pkg/errors"
)

// TODO: add regex
// TODO: add header
// TODO: add other checks?
// TODO: add default type that matches all requests
// TODO: add url rewrites
// NOTE: is there any sence for "host"? since this LB should be an entrypoint there should be only one host

type conditionType string

// TODO: find better names for constants
const (
	// TODO: find better names for constants
	PrefixCond    = conditionType("prefix")
	RegexpCond    = conditionType("regexp")
	HasHeaderCond = conditionType("hasheader")
	HeaderCond    = conditionType("header")
)

var validCondTypes = map[conditionType]bool{
	PrefixCond: true, RegexpCond: true, HasHeaderCond: true, HeaderCond: true,
}

func GetConditionType(t string) (conditionType, error) {
	ct := conditionType(strings.ToLower(t))
	// TODO: check condition types are equal based on underlying string
	_, ok := validCondTypes[ct]
	if !ok {
		return conditionType(""), errors.Errorf("Invalid condition type %s", t)
	}
	return ct, nil
}

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
	reg *regexp.Regexp
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
	return strings.EqualFold(r.Header.Get(c.header), c.value)
}

func GetCondition(t conditionType, key, value string) Condition {
	switch t {
	case PrefixCond:
		return &PrefixCondition{prefix: value}
	case RegexpCond:
		{
			reg, err := regexp.Compile(value)
			if err != nil {
				return nil
			}
			return &RegexpCondition{reg: reg}
		}
	case HasHeaderCond:
		return &HasHeaderCondition{header: value}
	case HeaderCond:
		return &HeaderValueCondition{header: key, value: value}
	}
	return nil
}
