package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/Iotic-Labs/gocd-jira-release-notes/mocks"
)

// WARNING: if true, the tests will call real GoCD, JIRA and Confluence API
// which is good for API and e2e testing, but not so much for GoCD pipeline
// so by default we'll use mocked/pre-canned responses
var useMockedResponse = true

func TestHandlerNotImplemented(t *testing.T) {
	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, _ := http.NewRequest(http.MethodPost, "/", nil)
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg := NewDefaultConfig()
		handleRequest(w, r, cfg)
	})

	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusNotImplemented {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotImplemented)
	}
}

func readSampleGocdPipeline(t *testing.T) []byte {
	filename := "./examples/gocd-pipeline-compare-long.json"
	json, err := os.ReadFile(filename)
	if err != nil {
		t.Errorf("could not read file: %s", err)
	}
	//fmt.Printf("Using mocked request/response: %s\n", filename)
	return json
}

func extractKeyFromJiraURL(url string) string {
	// e.g. https://<account>.atlassian.net/rest/agile/latest/issue/JI-1227
	parts := strings.Split(url, "/")
	return parts[len(parts)-1]
}

func readSampleJira(t *testing.T, url string) []byte {
	key := extractKeyFromJiraURL(url)
	filename := fmt.Sprintf("./examples/jira-%s.json", key)
	json, err := os.ReadFile(filename)
	if err != nil {
		t.Errorf("could not read file: %s", err)
	}
	//fmt.Printf("Using mocked request/response: %s\n", filename)
	return json
}

func TestHandlerNoContent(t *testing.T) {
	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	title := "ProjectA"
	pipeline := "our-pipeline"
	counter := 390
	query := fmt.Sprintf("?title=%s&pipeline=%s&counter=%d", title, pipeline, counter)
	req, _ := http.NewRequest(http.MethodGet, "/"+query, nil)
	rr := httptest.NewRecorder()

	cfg := NewDefaultConfig()

	if useMockedResponse {
		mockGet := func(req *http.Request) (*http.Response, error) {
			var json []byte
			if strings.HasSuffix(cfg.GocdUrl, req.Host) {
				json = readSampleGocdPipeline(t)
			} else if strings.HasSuffix(cfg.JiraUrl, req.Host) {
				json = []byte("{}")
			}
			r := ioutil.NopCloser(bytes.NewReader([]byte(json)))

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       r,
			}, nil
		}
		cfg.Client = &mocks.MockClient{
			DoFunc: mockGet,
		}
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleRequest(w, r, cfg)
	})

	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusNoContent {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}

func TestHandlerEndToEndSuccess(t *testing.T) {
	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	title := "The Best Web"
	pipeline := "iotic-webbing"
	counter := 614
	query := fmt.Sprintf("?title=%s&pipeline=%s&counter=%d", title, pipeline, counter)
	req, _ := http.NewRequest(http.MethodGet, "/"+query, nil)
	rr := httptest.NewRecorder()

	cfg := NewDefaultConfig()

	if useMockedResponse {
		mockGet := func(req *http.Request) (*http.Response, error) {
			//fmt.Printf("Test request for: %v\n", req.Host)
			var json []byte
			if strings.HasSuffix(cfg.GocdUrl, req.Host) {
				json = readSampleGocdPipeline(t)
			} else if strings.HasSuffix(cfg.JiraUrl, req.Host) {
				json = readSampleJira(t, req.URL.Path)
			}
			r := ioutil.NopCloser(bytes.NewReader([]byte(json)))

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       r,
			}, nil
		}
		cfg.Client = &mocks.MockClient{
			DoFunc: mockGet,
		}
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleRequest(w, r, cfg)
	})

	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}
