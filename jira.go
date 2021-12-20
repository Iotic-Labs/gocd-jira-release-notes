package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"

	"log"

	"gopkg.in/go-playground/validator.v9"
)

type JiraIssue struct {
	Expand string `json:"expand"`
	ID     string `json:"id"`
	Self   string `json:"self"`
	Key    string `json:"key"`
	Names  struct {
		Statuscategorychangedate string `json:"statuscategorychangedate"`
		Fixversions              string `json:"fixVersions"`
		// NOTE: this should be configurable;
		// the custom field will be different for each Jira account
		Customfield10110              string `json:"customfield_10110"`
		Resolution                    string `json:"resolution"`
		Lastviewed                    string `json:"lastViewed"`
		Epic                          string `json:"epic"`
		Labels                        string `json:"labels"`
		Aggregatetimeoriginalestimate string `json:"aggregatetimeoriginalestimate"`
		Timeestimate                  string `json:"timeestimate"`
		Versions                      string `json:"versions"`
		Assignee                      string `json:"assignee"`
		Status                        string `json:"status"`
		Components                    string `json:"components"`
		Aggregatetimeestimate         string `json:"aggregatetimeestimate"`
		Aggregateprogress             string `json:"aggregateprogress"`
		Progress                      string `json:"progress"`
		Issuetype                     string `json:"issuetype"`
		Timespent                     string `json:"timespent"`
		Sprint                        string `json:"sprint"`
		Aggregatetimespent            string `json:"aggregatetimespent"`
		Resolutiondate                string `json:"resolutiondate"`
		Workratio                     string `json:"workratio"`
		Issuerestriction              string `json:"issuerestriction"`
		Created                       string `json:"created"`
		Updated                       string `json:"updated"`
		Timeoriginalestimate          string `json:"timeoriginalestimate"`
		Security                      string `json:"security"`
		Attachment                    string `json:"attachment"`
		Flagged                       string `json:"flagged"`
		Environment                   string `json:"environment"`
		Duedate                       string `json:"duedate"`
	} `json:"names"`
	Fields struct {
		Statuscategorychangedate string        `json:"statuscategorychangedate"`
		Fixversions              []interface{} `json:"fixVersions"`
		// NOTE: this should be configurable;
		// the custom field will be different for each Jira account
		Customfield10110 string `json:"customfield_10110"`
		Resolution       struct {
			Self        string `json:"self"`
			ID          string `json:"id"`
			Description string `json:"description"`
			Name        string `json:"name"`
		} `json:"resolution"`
		Lastviewed                    string        `json:"lastViewed"`
		Epic                          interface{}   `json:"epic"`
		Labels                        []string      `json:"labels"`
		Aggregatetimeoriginalestimate interface{}   `json:"aggregatetimeoriginalestimate"`
		Timeestimate                  interface{}   `json:"timeestimate"`
		Versions                      []interface{} `json:"versions"`
		Assignee                      interface{}   `json:"assignee"`
		Status                        struct {
			Self           string `json:"self"`
			Description    string `json:"description"`
			Iconurl        string `json:"iconUrl"`
			Name           string `json:"name"`
			ID             string `json:"id"`
			Statuscategory struct {
				Self      string `json:"self"`
				ID        int    `json:"id"`
				Key       string `json:"key"`
				Colorname string `json:"colorName"`
				Name      string `json:"name"`
			} `json:"statusCategory"`
		} `json:"status"`
		Components            []interface{} `json:"components"`
		Aggregatetimeestimate interface{}   `json:"aggregatetimeestimate"`
		Aggregateprogress     struct {
			Progress int `json:"progress"`
			Total    int `json:"total"`
		} `json:"aggregateprogress"`
		Progress struct {
			Progress int `json:"progress"`
			Total    int `json:"total"`
		} `json:"progress"`
		Issuetype struct {
			Self        string `json:"self"`
			ID          string `json:"id"`
			Description string `json:"description"`
			Iconurl     string `json:"iconUrl"`
			Name        string `json:"name"`
			Subtask     bool   `json:"subtask"`
			Avatarid    int    `json:"avatarId"`
		} `json:"issuetype"`
		Timespent          interface{} `json:"timespent"`
		Sprint             interface{} `json:"sprint"`
		Aggregatetimespent interface{} `json:"aggregatetimespent"`
		Resolutiondate     string      `json:"resolutiondate"`
		Workratio          int         `json:"workratio"`
		Issuerestriction   struct {
			Issuerestrictions struct {
			} `json:"issuerestrictions"`
			Shoulddisplay bool `json:"shouldDisplay"`
		} `json:"issuerestriction"`
		Created              string        `json:"created"`
		Updated              string        `json:"updated"`
		Timeoriginalestimate interface{}   `json:"timeoriginalestimate"`
		Security             interface{}   `json:"security"`
		Attachment           []interface{} `json:"attachment"`
		Flagged              bool          `json:"flagged"`
		Environment          interface{}   `json:"environment"`
		Duedate              interface{}   `json:"duedate"`
	} `json:"fields"`
}

func validateJiraIssue(json JiraIssue) error {
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

func findJiraIssueKeys(commitMessage string) []string {
	r := regexp.MustCompile(`(?m)^(?P<Project>\w+)-(?P<Number>\d+)`)
	matches := r.FindAllString(commitMessage, -1)
	if len(matches) == 0 {
		return nil
	}
	return matches
}

func getJiraIssue(cfg *Config, key string) (*JiraIssue, error) {

	// see https://developer.atlassian.com/cloud/confluence/basic-auth-for-rest-apis/
	apiURL := fmt.Sprintf("%s/rest/agile/latest/issue/%s", cfg.JiraUrl, key)

	// NOTE: we could limit the fields by excluding them
	// …we could also expand the names, to check which custom field corresponds to the Release Notes
	// ?fields=-comment,-description,-issuelinks,-project,-watches,-worklog,-watches,-votes,-reporter,-subtasks,-creator,-priority,-closedSprints&expand=names&properties=-self

	log.Printf("Calling %s", apiURL)

	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
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
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}
	// NOTE: for debugging/testing
	//os.WriteFile(fmt.Sprintf("./examples/jira-%s.json", key), body, 0644)

	obj, err := parseJiraIssue(body)
	return &obj, err
}

func parseJiraIssue(jsonData []byte) (JiraIssue, error) {
	var data JiraIssue

	if !isJSON(jsonData) {
		return data, errors.New("cannot create object - invalid json")
	}

	err := json.Unmarshal(jsonData, &data)
	if err != nil {
		return data, err
	}

	err = validateJiraIssue(data)

	return data, err
}

func getUniqueJiraIssues(cfg *Config, jiraKeys []string) ([]JiraIssue, error) {
	uniqueJiraKeys := unique(jiraKeys)
	jiraIssues := []JiraIssue{}
	for _, jiraIssueKey := range uniqueJiraKeys {
		jiraIssue, err := getJiraIssue(cfg, jiraIssueKey)
		if err != nil {
			return nil, err
		}
		jiraIssues = append(jiraIssues, *jiraIssue)
	}
	return jiraIssues, nil
}

func extractReleaseNotes(jiraIssues []JiraIssue) *Notes {
	// combinedNotes := make(map[string]string)
	notes := &Notes{
		Groups: make(map[string][]string),
	}
	for _, issue := range jiraIssues {
		fmt.Printf("%s - %s\n", issue.Key, issue.Fields.Issuetype.Name)
		// e.g.
		// "h4. Breaking Change\n\n* rename Iotic Web API methods and objects"

		// TODO: we could validate that the Release Notes field
		// still corresponds to Customfield10110
		jiraNotes := issue.Fields.Customfield10110
		if jiraNotes == "" {
			fmt.Println("- no release notes found")
			continue
		}

		newGroups := extractGroups(jiraNotes)
		addGroups(notes.Groups, newGroups)
	}
	return notes
}

func addGroups(originalGroups map[string][]string, newGroups []Group) {
	for _, n := range newGroups {
		if val, ok := originalGroups[n.Name]; ok {
			// add to an existing group
			originalGroups[n.Name] = append(val, n.BulletPoints...)
			continue
		}
		// create a new group
		originalGroups[n.Name] = n.BulletPoints
	}
}

func findHeader(line string) string {
	// NOTE: tested in https://regex101.com/
	// using "h4. Breaking Change"
	r := regexp.MustCompile(`^(?:h\d{1}\.\s)(.*)`)
	match := r.FindAllStringSubmatch(line, -1)
	if len(match) == 0 {
		return ""
	}
	return match[0][1]
}

func extractGroups(notes string) []Group {
	groups := []Group{}
	lines := strings.Split(notes, "\n")
	for _, line := range lines {
		if len(strings.TrimSpace(line)) == 0 {
			continue
		}
		header := findHeader(line)
		if header != "" {
			groups = append(groups, Group{Name: header})
			continue
		}
		if len(groups) == 0 {
			defaultGroup := "Changes"
			groups = append(groups, Group{Name: defaultGroup})
		}
		lastGroup := &groups[len(groups)-1]
		lastGroup.BulletPoints = append(lastGroup.BulletPoints, line)
	}
	return groups
}
