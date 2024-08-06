package common

import (
	"strings"
)

func Extractdata(body string) map[string]string {

	pairs := strings.Split(body, " ")

	// Initialize a map to store extracted values
	dataMap := make(map[string]string)

	// Loop through each key-value pair
	for _, pair := range pairs {
		// Split each pair by '=' to separate key and value
		parts := strings.Split(pair, "=")
		if len(parts) == 2 {
			key := parts[0]
			value := parts[1]
			dataMap[key] = value
		}
	}
	return dataMap
}
