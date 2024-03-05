package util

import (
	"fmt"
	"os"
	"strconv"
)

func GetIntEnvironmentVariable(varName string, defaultValue int) (int, error) {
	variable := os.Getenv(varName)
	result, err := strconv.Atoi(variable)
	if err != nil {
		return defaultValue, fmt.Errorf("unable to parse variable %v with value: %v", varName, variable)
	}
	return result, nil
}

func FilterSlice(slice []string, f func(string) bool) []string {
	filtered := make([]string, 0)
	for _, v := range slice {
		if f(v) {
			filtered = append(filtered, v)
		}
	}
	return filtered
}

func ArrayContains(slice []int32, searchElement int32) bool {
	for _, element := range slice {
		if element == searchElement {
			return true
		}
	}
	return false
}
