package main

import (
	"encoding/json"

	"gopkg.in/go-playground/validator.v9"
)

var validate *validator.Validate

func isJSON(jsonData []byte) bool {
	var j map[string]interface{}
	if err := json.Unmarshal(jsonData, &j); err != nil {
		return false
	}
	return true
}
