package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"gopkg.in/go-playground/validator.v9"
)

// cURL sample:
// url="https://<account>.atlassian.net"/wiki/rest/api/content/"
// curl -X POST -u "<email>:<apikey>" -H "Content-Type: application/json" $url --data @- << EOF
// {
//     "type":"blogpost",
//     "space":{"key":"RN"},
//     "status": "current",
//     "id":"20220101001",
//     "title":"2022-01-01 Our Project v1.5.0 - Release Notes",
//     "body":{
//         "storage":{
//             "value":"<h1>Breaking Changes</h1><ul><li><b>nothing</b> 123</li></ul>",
//             "representation":"storage"}
//         }
// }
// EOF

// ConfluencePost represents Confluence Blog Post object
type ConfluencePost struct {
	Type   string          `json:"type"`
	Space  ConfluenceSpace `json:"space"`
	Status string          `json:"status"`
	// ID is optional, it could be created to match e.g. YYYYMMDDHHMMSS
	// ID     string          `json:"id"`
	Title    string             `json:"title"`
	Body     ConfluenceBody     `json:"body"`
	Metadata ConfluenceMetadata `json:"metadata"`
}

type ConfluenceSpace struct {
	Key string `json:"key"`
}

type ConfluenceBody struct {
	Storage ConfluenceStorage `json:"storage"`
}

type ConfluenceStorage struct {
	Value          string `json:"value"`
	Representation string `json:"representation"`
}

type ConfluenceBlogPost struct {
	ID     string `json:"id"`
	Type   string `json:"type"`
	Status string `json:"status"`
	Title  string `json:"title"`
}
type ConfluenceMetadata struct {
	Labels []ConfluenceLabel `json:"labels"`
}

type ConfluenceLabel struct {
	Name string `json:"name"`
}

func NewConfluencePost(spaceKey string, title string, content string, label string) *ConfluencePost {
	return &ConfluencePost{
		Type:   "blogpost",
		Space:  ConfluenceSpace{Key: spaceKey},
		Status: "current",
		Title:  title,
		Body: ConfluenceBody{
			Storage: ConfluenceStorage{
				Representation: "editor2",
				Value:          content,
			},
		},
		Metadata: ConfluenceMetadata{
			Labels: []ConfluenceLabel{
				{Name: label},
			},
		},
	}
}

func publishReleaseNotesToConfluence(cfg *Config, timestamp time.Time, title string, pipeline string, version string, notes *Notes) (*ConfluenceBlogPost, error) {

	// see https://developer.atlassian.com/cloud/confluence/rest/api-group-content/#api-api-content-post
	apiURL := fmt.Sprintf("%s/wiki/rest/api/content/", cfg.JiraUrl)

	log.Printf("Calling %s", apiURL)

	// NOTE: this is a magical string for the YYYY-MM-DD format
	date := timestamp.Format("2006-01-02")

	postTitle := fmt.Sprintf("%s Release Notes %s - %s", title, version, date)

	content, err := createConfluenceContentHTML(cfg, notes)
	if err != nil {
		return nil, err
	}

	post := NewConfluencePost(cfg.ConfluenceSpaceKey, postTitle, content, pipeline)
	jsonStr, _ := json.Marshal(post)

	req, err := http.NewRequest(http.MethodPost, apiURL, bytes.NewBuffer(jsonStr))
	if err != nil {
		return nil, err
	}

	// create a token here: https://id.atlassian.com/manage-profile/security/api-tokens
	req.SetBasicAuth(cfg.JiraUser, cfg.JiraApiKey)
	req.Header.Add("Content-Type", "application/json")

	resp, err := cfg.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	response, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to post to Confluence %v %v", err, string(response))
	}
	// NOTE: for debugging
	// os.WriteFile("./sample-data/confluence-post.json", response, 0644)

	blogPost, err := parseConfluenceBlogPost(response)
	if err != nil {
		return nil, err
	}
	return blogPost, nil
}

func createConfluenceContentHTML(cfg *Config, notes *Notes) (string, error) {
	var buf bytes.Buffer
	for k, v := range notes.Groups {
		buf.WriteString(fmt.Sprintf("\nh1. %s\n", k))
		for _, l := range v {
			buf.Write([]byte(l + "\n"))
		}
	}
	result := buf.Bytes()

	// NOTE: I've tried this approach initially,
	// but it didn't work well with JIRA/Confluence URL format or JIRA/Confluence macros
	// html := markdown.ToHTML([]byte(result), nil, nil)
	// have I missed some trick?

	confluenceFormat, err := convertToConfluenceFormat(cfg, result)
	if err != nil {
		return "", err
	}

	return string(confluenceFormat), nil
}

func convertToConfluenceFormat(cfg *Config, text []byte) ([]byte, error) {

	// then convert the remaining wiki markup to Confluence storage format
	// see https://developer.atlassian.com/server/confluence/confluence-rest-api-examples/#convert-wiki-markup-to-storage-format
	apiURL := fmt.Sprintf("%s/wiki/rest/api/contentbody/convert/editor2", cfg.JiraUrl)

	log.Printf("Calling %s", apiURL)

	post := &ConfluenceStorage{
		Value:          string(text),
		Representation: "wiki",
	}
	jsonStr, _ := json.Marshal(post)

	// NOTE: for debugging/testing
	// os.WriteFile("./sample-data/confluence-convert-format.json", jsonStr, 0644)

	req, err := http.NewRequest(http.MethodPost, apiURL, bytes.NewBuffer(jsonStr))
	if err != nil {
		return nil, err
	}

	// create a token here: https://id.atlassian.com/manage-profile/security/api-tokens

	req.SetBasicAuth(cfg.JiraUser, cfg.JiraApiKey)
	req.Header.Add("Content-Type", "application/json")

	resp, err := cfg.Client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("401 Unauthorized")
	}
	if resp.StatusCode != http.StatusOK {
		log.Fatalln(fmt.Errorf("failed to convert Confluence format: %v", err))
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	// NOTE: for debugging/testing
	//os.WriteFile(fmt.Sprintf("./sample-data/jira-%s.json", key), body, 0644)

	obj, err := parseConfluenceStorage(body)
	if err != nil {
		return nil, err
	}

	return []byte(obj.Value), nil
}

func validateConfluenceStorage(json ConfluenceStorage) error {
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

func parseConfluenceStorage(jsonData []byte) (ConfluenceStorage, error) {
	var data ConfluenceStorage

	if !isJSON(jsonData) {
		return data, errors.New("cannot create object - invalid json")
	}

	err := json.Unmarshal(jsonData, &data)
	if err != nil {
		return data, err
	}

	err = validateConfluenceStorage(data)

	return data, err
}

func validateConfluenceBlogPost(json ConfluenceBlogPost) error {
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

func parseConfluenceBlogPost(jsonData []byte) (*ConfluenceBlogPost, error) {
	var data ConfluenceBlogPost

	if !isJSON(jsonData) {
		return &data, errors.New("cannot create object - invalid json")
	}

	err := json.Unmarshal(jsonData, &data)
	if err != nil {
		return &data, err
	}

	err = validateConfluenceBlogPost(data)

	return &data, err
}
