package jsonutil

import (
	"encoding/json"
)

// ToJSON converts a given request object into a JSON string.
// It returns an empty string if there is an error during JSON marshaling.
func ToJSON(data interface{}) string {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return ""
	}

	return string(jsonBytes)
}

// IsJSONString checks if a given string is a valid JSON string.
// It returns true if the string is a valid JSON string, otherwise false.
func IsJSONString(jsonStr string) bool {
	var js map[string]interface{}
	return json.Unmarshal([]byte(jsonStr), &js) == nil
}
