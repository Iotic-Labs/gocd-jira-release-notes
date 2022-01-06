package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Iotic-Labs/gocd-jira-release-notes/mocks"
)

func TestPublishToConfluence(t *testing.T) {
	cfg := NewDefaultConfig()

	date := time.Now()
	notes := &Notes{
		Groups: map[string][]string{
			"Change": {
				"one",
				"two",
				"three",
			},
		},
	}

	// NOTE: if you do not use a mocked response
	// then an actual blog page will be created in Confluence
	// and if the page already exists, the test will fail
	// … remember to delete the test page
	if useMockedResponse {
		mockGet := func(req *http.Request) (*http.Response, error) {
			// use a mocked client to prevent posting to Confluence
			if strings.HasSuffix("/wiki/rest/api/contentbody/convert/editor2", req.URL.Path) {
				fmt.Printf("Test request for: %s%s\n", req.Host, req.URL.Path)
				json := readSampleConfluenceConvert(t)
				r := ioutil.NopCloser(bytes.NewReader([]byte(json)))
				return &http.Response{
					StatusCode: 200,
					Body:       r,
				}, nil
			}
			if strings.HasSuffix("/wiki/rest/api/content/", req.URL.Path) {
				fmt.Printf("Test request for: %s%s\n", req.Host, req.URL.Path)
				json := readSampleConfluencePost(t)
				r := ioutil.NopCloser(bytes.NewReader([]byte(json)))
				return &http.Response{
					StatusCode: 200,
					Body:       r,
				}, nil
			}
			// …otherwise use a real http client
			client := &http.Client{}
			return client.Do(req)
		}
		cfg.Client = &mocks.MockClient{
			DoFunc: mockGet,
		}
	}

	_, err := publishReleaseNotesToConfluence(cfg, date, "Test", "test", "v0.0.1", notes)
	if err != nil {
		t.Error(err)
	}
}

func readSampleConfluenceConvert(t *testing.T) []byte {
	filename := "./sample-data/confluence-conversion.json"
	json, err := os.ReadFile(filename)
	if err != nil {
		t.Errorf("could not read file: %s", err)
	}
	fmt.Printf("Using mocked request/response: %s\n", filename)
	return json
}

func readSampleConfluencePost(t *testing.T) []byte {
	filename := "./sample-data/confluence-post.json"
	json, err := os.ReadFile(filename)
	if err != nil {
		t.Errorf("could not read file: %s", err)
	}
	fmt.Printf("Using mocked request/response: %s\n", filename)
	return json
}
