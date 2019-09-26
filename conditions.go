package main

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

// TODO: add regex
// TODO: add header
// TODO: add other checks?
// TODO: add default type that matches all requests
// TODO: add url rewrites

type ConditionType string

// TODO: find better names for constants
const (
	// TODO: find better names for constants
	PrefixCond    = ConditionType("prefix")
	RegexpCond    = ConditionType("regexp")
	HasHeaderCond = ConditionType("hasheader")
	HeaderCond    = ConditionType("header")
)

var validCondTypes = map[ConditionType]bool{
	PrefixCond: true, RegexpCond: true, HasHeaderCond: true, HeaderCond: true,
}

func GetConditionType(t string) (ConditionType, error) {
	ct := ConditionType(strings.ToLower(t))
	// TODO: check condition types are equal based on underlying string
	_, ok := validCondTypes[ct]
	if !ok {
		return ConditionType(""), fmt.Errorf("Invalid condition type %s", t)
	}
	return ct, nil
}

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
	return r.Header.Get(c.header) == c.value
}

func GetCondition(t ConditionType, key, value string) Condition {
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