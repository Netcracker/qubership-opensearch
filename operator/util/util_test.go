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

//go:build unit

package util

import (
	"reflect"
	"testing"
)

func TestDifference(t *testing.T) {
	tests := []struct {
		name     string
		first    []string
		second   []string
		expected []string
	}{
		{"all present", []string{"a", "b"}, []string{"b", "a", "c"}, nil},
		{"some absent", []string{"a", "b", "c"}, []string{"b"}, []string{"a", "c"}},
		{"second is empty", []string{"a"}, nil, []string{"a"}},
		{"first is empty", nil, []string{"a"}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := Difference(tt.first, tt.second)
			if len(actual) != len(tt.expected) || (len(actual) > 0 && !reflect.DeepEqual(actual, tt.expected)) {
				t.Errorf("expected %v, got %v", tt.expected, actual)
			}
		})
	}
}
