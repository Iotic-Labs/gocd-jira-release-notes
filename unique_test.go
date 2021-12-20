package main

import (
	"reflect"
	"testing"
)

func TestUnique(t *testing.T) {
	type test struct {
		input []string
		want  []string
	}
	tests := []test{
		{input: []string{}, want: []string{}},
		{input: []string{"a", "a"}, want: []string{"a"}},
		{input: []string{"a", "b", "a"}, want: []string{"a", "b"}},
		{input: []string{"c", "a", "b", "a"}, want: []string{"c", "a", "b"}},
	}

	for _, tc := range tests {
		got := unique(tc.input)
		if !reflect.DeepEqual(tc.want, got) {
			t.Fatalf("expected: %v, got: %v", tc.want, got)
		}
	}
}
