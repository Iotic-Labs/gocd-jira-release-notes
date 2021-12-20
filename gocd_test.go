package main

import (
	"os"
	"reflect"
	"strings"
	"testing"
)

func ErrorContains(out error, want string) bool {
	if out == nil {
		return want == ""
	}
	if want == "" {
		return false
	}
	return strings.Contains(out.Error(), want)
}

func TestShouldReturnErrorIfInvalidJsonPassedIn(t *testing.T) {

	invalidJSON := []byte(`abc123`)
	_, err := parseGocdPipelineComparison(invalidJSON)
	if err == nil {
		t.Errorf("expected error from parseGocdPipelineComparison: %v", err)
	}
	if !ErrorContains(err, "cannot create object - invalid json") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestParseInvalidPipelineHistory(t *testing.T) {

	invalidJSON := []byte(`abc123`)
	_, err := parseGocdPipelineHistory(invalidJSON)
	if err == nil {
		t.Errorf("expected error from parseGocdPipelineHistory: %v", err)
	}
	if !ErrorContains(err, "cannot create object - invalid json") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestShouldGetPipelineComparisonFromValidJson(t *testing.T) {

	filename := "./examples/gocd-pipeline-compare-long.json"
	validJSON, err := os.ReadFile(filename)
	if err != nil {
		t.Errorf("could not read file: %s", err)
	}
	_, err = parseGocdPipelineComparison(validJSON)
	if err != nil {
		t.Errorf("unexpected error from parseGocdPipelineComparison: %v", err)
	}
}

func TestShouldGetPipelineFromValidJson(t *testing.T) {

	filename := "./examples/gocd-pipeline-history.json"
	validJSON, err := os.ReadFile(filename)
	if err != nil {
		t.Errorf("could not read file: %s", err)
	}
	_, err = parseGocdPipelineHistory(validJSON)
	if err != nil {
		t.Errorf("unexpected error from parseGocdPipelineHistory: %v", err)
	}
}

func TestConvertScheduledDateToGo(t *testing.T) {

	type test struct {
		input int64
		want  string
	}
	tests := []test{
		{input: 1615391237492, want: "2021-03-10"},
	}

	for _, tc := range tests {
		timestamp := convertGocdTimestampToGo(tc.input)
		got := timestamp.Format("2006-01-02")
		if !reflect.DeepEqual(tc.want, got) {
			t.Fatalf("expected: %v, got: %v", tc.want, got)
		}
	}
}

func TestConvertScheduledDateFromJSONToGo(t *testing.T) {

	// Arrange
	filename := "./examples/gocd-pipeline-history.json"
	validJSON, err := os.ReadFile(filename)
	if err != nil {
		t.Errorf("could not read file: %s", err)
	}
	h, _ := parseGocdPipelineHistory(validJSON)
	expectedDate := "2021-03-10"

	// Act
	timestamp := convertGocdTimestampToGo(h.ScheduledDate)

	// Assert
	actualDate := timestamp.Format("2006-01-02")
	if actualDate != expectedDate {
		t.Errorf("actual date %s doesn't match the expected date: %s", actualDate, expectedDate)
	}
}
