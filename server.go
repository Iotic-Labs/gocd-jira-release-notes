package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/rs/xid"
	log "github.com/sirupsen/logrus"
)

var logger *log.Entry
var requestID string

type QueryParams struct {
	Title    string
	Pipeline string
	Counter  int
}

type Notes struct {
	DependabotChanges []string
	Groups            map[string][]string
}

type Group struct {
	Name         string
	BulletPoints []string
}

func init() {
	requestID = fmt.Sprintf("%v", xid.New())
}

func Serve() {
	cfg := NewDefaultConfig()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handleRequest(w, r, cfg)
	})
	log.Infof("starting server on %s", cfg.Port)
	log.Fatal(http.ListenAndServe(cfg.Port, nil))
}

func writeResponseError(w http.ResponseWriter, err error) {
	logger.Errorf("Got error %s", err.Error())
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte(fmt.Sprintf("%v", string(err.Error()))))
}

func getQueryParamsFromRequest(query url.Values) (*QueryParams, error) {

	title := query.Get("title")
	if title == "" {
		return nil, fmt.Errorf("set title in query string")
	}
	logger.Infof("Title: %s\n", title)

	pipeline := query.Get("pipeline")
	if pipeline == "" {
		return nil, fmt.Errorf("set pipeline in query string")
	}
	logger.Infof("Pipeline: %s\n", pipeline)

	counterParam := query.Get("counter")
	counter, err := strconv.Atoi(counterParam)
	if err != nil {
		return nil, fmt.Errorf("could not process counter")
	}
	if counter == 0 {
		return nil, fmt.Errorf("set counter in query string")
	}
	logger.Infof("Counter: %d\n", counter)
	params := &QueryParams{
		Title:    title,
		Pipeline: pipeline,
		Counter:  counter,
	}
	return params, nil
}

func handleRequest(w http.ResponseWriter, r *http.Request, cfg *Config) {
	logger = log.WithFields(log.Fields{"requestID": requestID})

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusNotImplemented)
		return
	}

	logger.Infoln("Creating release notes")

	queryParams, err := getQueryParamsFromRequest(r.URL.Query())
	if err != nil {
		writeResponseError(w, err)
		return
	}

	notes, err := createReleaseNotes(cfg, queryParams)
	if err != nil {
		writeResponseError(w, err)
		return
	}
	if notes == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	jsonNotes, _ := json.Marshal(notes)
	w.Write(jsonNotes)
	w.WriteHeader(http.StatusOK)
}

func createReleaseNotes(cfg *Config, queryParams *QueryParams) (*Notes, error) {

	pipelineHistory, err := getGocdPipelineHistory(cfg, queryParams.Pipeline, queryParams.Counter)
	if err != nil {
		return nil, err
	}

	pipelineComparison, err := getGocdPipelineComparison(cfg, queryParams.Pipeline, queryParams.Counter)
	if err != nil {
		return nil, err
	}

	allJiraKeys := getStoriesFromCommits(pipelineComparison)
	jiraIssues, err := getUniqueJiraIssues(cfg, allJiraKeys)
	if err != nil || len(jiraIssues) == 0 {
		// no JIRA issues found, so no release notes
		return nil, err
	}

	releaseNotes := extractReleaseNotes(jiraIssues)
	if releaseNotes == nil || len(releaseNotes.Groups) == 0 {
		// JIRA issues found, but none have release notes
		return nil, nil
	}

	version := pipelineHistory.Label
	timestamp := convertGocdTimestampToGo(pipelineHistory.ScheduledDate)
	_, err = publishReleaseNotesToConfluence(cfg, timestamp, queryParams.Title, queryParams.Pipeline, version, releaseNotes)
	if err != nil {
		return releaseNotes, err
	}

	return releaseNotes, nil
}
