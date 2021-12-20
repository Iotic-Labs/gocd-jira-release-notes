package main

import (
	"os"
	"reflect"
	"testing"
)

func TestFindJiraIssueKey(t *testing.T) {
	type test struct {
		input string
		want  []string
	}
	tests := []test{
		{input: "12345", want: nil},
		{input: "abc JI-1234", want: nil},
		{input: "JI-1234", want: []string{"JI-1234"}},
		{input: "JI-1234 abc", want: []string{"JI-1234"}},
		{input: "JI-1234 abc\nnnn JI-5678 xyz\n", want: []string{"JI-1234"}},
		{input: "JI-1234 abc\nJI-5678 xyz\n", want: []string{"JI-1234", "JI-5678"}},
		{input: "JI-1234 abc\nJI-5678 xyz\nIN-0001", want: []string{"JI-1234", "JI-5678", "IN-0001"}},
	}

	for _, tc := range tests {
		got := findJiraIssueKeys(tc.input)
		if !reflect.DeepEqual(tc.want, got) {
			t.Fatalf("expected: %v, got: %v", tc.want, got)
		}
	}
}

func TestShouldReturnErrorIfInvalidJiraJsonPassedIn(t *testing.T) {

	invalidJSON := []byte(`abc123`)
	_, err := parseJiraIssue(invalidJSON)
	if !ErrorContains(err, "cannot create object - invalid json") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestShouldGetJiraIssueFromValidJson(t *testing.T) {

	filename := "./examples/jira-issue-sample.json"
	validJSON, err := os.ReadFile(filename)
	if err != nil {
		t.Errorf("could not read file: %s", err)
	}
	_, err = parseJiraIssue(validJSON)
	if err != nil {
		t.Errorf("unexpected error from ParseJiraIssue: %v", err)
	}
}

func TestFindHeader(t *testing.T) {
	type test struct {
		input string
		want  string
	}
	tests := []test{
		{input: "", want: ""},
		{input: "My release notes", want: ""},
		{input: "not a heading", want: ""},
		{input: "h1. Blah", want: "Blah"},
		{input: "h2. Improvements", want: "Improvements"},
		{input: "h3. Test Heading", want: "Test Heading"},
		{input: "h4. Breaking Changes", want: "Breaking Changes"},
	}

	for _, tc := range tests {
		got := findHeader(tc.input)
		if !reflect.DeepEqual(tc.want, got) {
			t.Fatalf("expected: %v, got: %v", tc.want, got)
		}
	}
}

func TestExtractHeadings(t *testing.T) {
	type test struct {
		input string
		want  []Group
	}
	tests := []test{
		{input: "", want: []Group{}},
		{input: "My release notes", want: []Group{
			{
				Name:         "Changes",
				BulletPoints: []string{"My release notes"},
			},
		}},
		{input: "h1. Breaking Changes\nMy release notes", want: []Group{
			{
				Name:         "Breaking Changes",
				BulletPoints: []string{"My release notes"},
			},
		}},
		{input: "h2. Nested List\n* Item 1\n** Nested item 1\n** Nested item 2", want: []Group{
			{
				Name:         "Nested List",
				BulletPoints: []string{"* Item 1", "** Nested item 1", "** Nested item 2"},
			},
		}},
		{input: "h3. Significant Changes\n* Point 1\n* Point 2\n\nh3. Breaking Changes\nMy release notes", want: []Group{
			{
				Name:         "Significant Changes",
				BulletPoints: []string{"* Point 1", "* Point 2"},
			},
			{
				Name:         "Breaking Changes",
				BulletPoints: []string{"My release notes"},
			},
		}},
	}

	for _, tc := range tests {
		got := extractGroups(tc.input)
		if !reflect.DeepEqual(tc.want, got) {
			t.Fatalf("expected: %v, got: %v", tc.want, got)
		}
	}
}
