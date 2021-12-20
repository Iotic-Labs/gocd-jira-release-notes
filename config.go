package main

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Client             HTTPClient
	Port               string
	GocdUrl            string
	GocdApiKey         string
	JiraUrl            string
	JiraUser           string
	JiraApiKey         string
	ConfluenceSpaceKey string
}

func init() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("fatal error config file: %w", err))
	}
}

func NewDefaultConfig() *Config {
	gocdApiKey, err := getAPISecret("gocdapikey")
	if err != nil {
		log.Fatalf("failed to read secret %s: %v", "gocdapikey", err)
	}

	jiraApiKey, err := getAPISecret("jiraapikey")
	if err != nil {
		log.Fatalf("failed to read secret %s: %v", "jiraapikey", err)
	}

	// NOTE: we could import a proper cert,
	// e.g. for GoCD our own corp.iotic.ca.pem cert
	// but that would add extra 20 lines I don't want to maintain now
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	return &Config{
		Client:             &http.Client{Transport: transport},
		Port:               viper.GetString("port"),
		GocdUrl:            viper.GetString("gocdUrl"),
		GocdApiKey:         gocdApiKey, // NOTE: create your own GoCD personal access token
		JiraUrl:            viper.GetString("jiraUrl"),
		JiraUser:           viper.GetString("jiraUser"), // NOTE: change to your email address for development
		JiraApiKey:         jiraApiKey,                  // NOTE: change to your JIRA password during development
		ConfluenceSpaceKey: viper.GetString("confluenceSpaceKey"),
	}
}

func getAPISecret(secretName string) (string, error) {
	rtn := ""

	// first try the k8s/FaaS secrets
	secretBytes, err := ioutil.ReadFile("/var/openfaas/secrets/" + secretName)
	if err != nil {
		secretBytes, err = ioutil.ReadFile("/run/secrets/" + secretName)
	}

	if err == nil {
		rtn = strings.TrimSpace(string(secretBytes))
		return rtn, nil
	}

	val := viper.GetString(secretName)
	if val != "" {
		return val, nil
	}

	fmt.Printf("using default for '%s' - use for testing only\n", secretName)
	return "notset", nil
}
