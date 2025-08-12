// Copyright 2024-2025 NetCracker Technology Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
