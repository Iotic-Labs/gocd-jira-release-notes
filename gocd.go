package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"gopkg.in/go-playground/validator.v9"
)

// NOTE: used https://mholt.github.io/json-to-go/
// not-used: https://pkg.go.dev/github.com/drewsonne/go-gocd/gocd#PipelineInstance

type GocdPipelineHistory struct {
	Name                string      `json:"name"`
	Counter             int         `json:"counter"`
	Label               string      `json:"label"`
	NaturalOrder        float64     `json:"natural_order"`
	CanRun              bool        `json:"can_run"`
	PreparingToSchedule bool        `json:"preparing_to_schedule"`
	Comment             interface{} `json:"comment"`
	ScheduledDate       int64       `json:"scheduled_date"`
	BuildCause          struct {
		TriggerMessage    string `json:"trigger_message"`
		TriggerForced     bool   `json:"trigger_forced"`
		Approver          string `json:"approver"`
		MaterialRevisions []struct {
			Changed  bool `json:"changed"`
			Material struct {
				Name        string `json:"name"`
				Fingerprint string `json:"fingerprint"`
				Type        string `json:"type"`
				Description string `json:"description"`
			} `json:"material"`
			Modifications []struct {
				Revision     string      `json:"revision"`
				ModifiedTime int64       `json:"modified_time"`
				UserName     string      `json:"user_name"`
				Comment      string      `json:"comment"`
				EmailAddress interface{} `json:"email_address"`
			} `json:"modifications"`
		} `json:"material_revisions"`
	} `json:"build_cause"`
	Stages []struct {
		Result            string      `json:"result"`
		Status            string      `json:"status"`
		RerunOfCounter    interface{} `json:"rerun_of_counter"`
		Name              string      `json:"name"`
		Counter           string      `json:"counter"`
		Scheduled         bool        `json:"scheduled"`
		ApprovalType      string      `json:"approval_type"`
		ApprovedBy        string      `json:"approved_by"`
		OperatePermission bool        `json:"operate_permission"`
		CanRun            bool        `json:"can_run"`
		Jobs              []struct {
			Name          string `json:"name"`
			ScheduledDate int64  `json:"scheduled_date"`
			State         string `json:"state"`
			Result        string `json:"result"`
		} `json:"jobs"`
	} `json:"stages"`
}

type GocdPipelineComparison struct {
	Links struct {
		Self struct {
			Href string `json:"href"`
		} `json:"self"`
		Doc struct {
			Href string `json:"href"`
		} `json:"doc"`
	} `json:"_links"`
	PipelineName string `json:"pipeline_name"`
	FromCounter  int    `json:"from_counter"`
	ToCounter    int    `json:"to_counter"`
	IsBisect     bool   `json:"is_bisect"`
	Changes      []struct {
		Material struct {
			Type       string `json:"type"`
			Attributes struct {
				Destination     string `json:"destination"`
				Filter          string `json:"filter"`
				InvertFilter    bool   `json:"invert_filter"`
				Name            string `json:"name"`
				AutoUpdate      bool   `json:"auto_update"`
				DisplayType     string `json:"display_type"`
				Description     string `json:"description"`
				URL             string `json:"url"`
				Branch          string `json:"branch"`
				SubmoduleFolder string `json:"submodule_folder"`
				ShallowClone    bool   `json:"shallow_clone"`
				// additional fields for material type=dependency
				Pipeline string `json:"pipeline"`
				Stage    string `json:"stage"`
			} `json:"attributes"`
		} `json:"material"`
		Revision []struct {
			RevisionSha   string    `json:"revision_sha"`
			ModifiedBy    string    `json:"modified_by"`
			ModifiedAt    time.Time `json:"modified_at"`
			CommitMessage string    `json:"commit_message"`
			// additional fields for material type=dependency
			Revision        string    `json:"revision"`
			PipelineCounter string    `json:"pipeline_counter"`
			CompletedAt     time.Time `json:"completed_at"`
		} `json:"revision"`
	} `json:"changes"`
}

func validateHistory(json GocdPipelineHistory) error {
	validate = validator.New()
	err := validate.Struct(json)
	if err != nil {

		if _, ok := err.(*validator.InvalidValidationError); ok {
			logger.Error(err)
			return err
		}

		failures := []string{}
		for _, err := range err.(validator.ValidationErrors) {
			failures = append(failures, fmt.Sprintf("ns:%s tag:%s param:%s type:%s \n", err.Namespace(), err.Tag(), err.Param(), err.Type()))
		}
		return fmt.Errorf("data validation failed %v", failures)
	}
	return nil
}

func validateComparison(json GocdPipelineComparison) error {
	validate = validator.New()
	err := validate.Struct(json)
	if err != nil {

		if _, ok := err.(*validator.InvalidValidationError); ok {
			logger.Error(err)
			return err
		}

		failures := []string{}
		for _, err := range err.(validator.ValidationErrors) {
			failures = append(failures, fmt.Sprintf("ns:%s tag:%s param:%s type:%s \n", err.Namespace(), err.Tag(), err.Param(), err.Type()))
		}
		return fmt.Errorf("data validation failed %v", failures)
	}
	return nil
}

func loadPipelineHistoryFromResponse(body io.ReadCloser) (*GocdPipelineHistory, error) {
	if body == nil {
		return nil, fmt.Errorf("empty response body")
	}
	defer body.Close()
	input, _ := ioutil.ReadAll(body)

	// NOTE: for debugging
	//os.WriteFile("./examples/gocd-pipeline-history-2.json", input, 0644)

	data, err := parseGocdPipelineHistory(input)
	return &data, err
}

func loadPipelineComparisonFromResponse(body io.ReadCloser) (*GocdPipelineComparison, error) {
	if body == nil {
		return nil, fmt.Errorf("empty response body")
	}
	defer body.Close()
	input, _ := ioutil.ReadAll(body)
	// NOTE: for debugging
	// os.WriteFile("./examples/gocd-pipeline-compare.json", input, 0644)

	data, err := parseGocdPipelineComparison(input)
	return &data, err
}

func getGocdPipelineHistory(cfg *Config, pipeline string, counter int) (*GocdPipelineHistory, error) {

	apiURL := fmt.Sprintf("%s/go/api/pipelines/%s/%d", cfg.GocdUrl, pipeline, counter)

	log.Printf("Calling %s", apiURL)

	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", cfg.GocdApiKey))
	req.Header.Add("Accept", "application/vnd.go.cd.v1+json")

	resp, err := cfg.Client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("401 Unauthorized")
	}
	return loadPipelineHistoryFromResponse(resp.Body)
}

func getGocdPipelineComparison(cfg *Config, pipeline string, counter int) (*GocdPipelineComparison, error) {

	prevCounter := counter - 1
	apiURL := fmt.Sprintf("%s/go/api/pipelines/%s/compare/%d/%d", cfg.GocdUrl, pipeline, prevCounter, counter)
	// webUrl := fmt.Sprintf("%s/pipelines/value_stream_map/%s/%s", baseUrl, pipeline, counter)

	log.Printf("Calling %s", apiURL)

	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", cfg.GocdApiKey))
	req.Header.Add("Accept", "application/vnd.go.cd.v2+json")

	resp, err := cfg.Client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("401 Unauthorized")
	}
	return loadPipelineComparisonFromResponse(resp.Body)
}

func parseGocdPipelineHistory(jsonData []byte) (GocdPipelineHistory, error) {
	var result GocdPipelineHistory

	if !isJSON(jsonData) {
		return result, errors.New("cannot create object - invalid json")
	}

	err := json.Unmarshal(jsonData, &result)
	if err != nil {
		return result, err
	}

	err = validateHistory(result)

	return result, err
}

func parseGocdPipelineComparison(jsonData []byte) (GocdPipelineComparison, error) {
	var result GocdPipelineComparison

	if !isJSON(jsonData) {
		return result, errors.New("cannot create object - invalid json")
	}

	err := json.Unmarshal(jsonData, &result)
	if err != nil {
		return result, err
	}

	err = validateComparison(result)

	return result, err
}

func getStoriesFromCommits(pipelineHistory *GocdPipelineComparison) []string {
	allJiraKeys := []string{}
	for _, changes := range pipelineHistory.Changes {
		for _, revision := range changes.Revision {
			if revision.CommitMessage != "" {
				// if strings.HasPrefix(revision.ModifiedBy, "dependabot") {
				// 	data.DependabotChanges = append(data.DependabotChanges, revision.CommitMessage)
				// 	continue
				// }
				if strings.HasPrefix(revision.CommitMessage, "") {
					jiraKeys := findJiraIssueKeys(revision.CommitMessage)
					allJiraKeys = append(allJiraKeys, jiraKeys...)
				}
			}
		}

	}
	return allJiraKeys
}

func convertGocdTimestampToGo(unixTimeStamp int64) time.Time {
	// e.g. "scheduled_date" : 1615391237492, is a nanosecond time
	// so we need to convert it to seconds
	secondsSinceEpoch := unixTimeStamp / 1000
	return time.Unix(secondsSinceEpoch, 0)
}
