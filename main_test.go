package main

import (
	"reflect"
	"testing"
)

func TestParseArgs(t *testing.T) {
	var tests = []struct {
		in      []string
		filters []string
		command []string
	}{
		{[]string{}, nil, nil},
		{[]string{"filter"}, []string{"filter"}, nil},
		{[]string{"filter", "another"}, []string{"filter", "another"}, nil},
		{[]string{"filter", "another", "--", "cmd"}, []string{"filter", "another"}, []string{"cmd"}},
		{[]string{"--", "cmd"}, nil, []string{"cmd"}},
	}
	for _, tt := range tests {
		filters, command := ParseArgs(tt.in)
		if !reflect.DeepEqual(filters, tt.filters) {
			t.Errorf("ParseArgs(%+v).Filters => %#v, want %#v", tt.in, filters, tt.filters)
		}
		if !reflect.DeepEqual(command, tt.command) {
			t.Errorf("ParseArgs(%+v).Command => %#v, want %#v", tt.in, command, tt.command)
		}
	}
}
