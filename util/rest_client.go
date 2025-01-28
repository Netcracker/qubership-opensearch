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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Credentials struct {
	Username string
	Password string
}

func NewCredentials(username, password string) Credentials {
	return Credentials{
		Username: username,
		Password: password,
	}
}

type RestClient struct {
	url         string
	httpClient  http.Client
	credentials Credentials
}

func NewRestClient(url string, httpClient http.Client, credentials Credentials) *RestClient {
	return &RestClient{
		url:         url,
		httpClient:  httpClient,
		credentials: credentials,
	}
}

func (rc RestClient) SendRequest(method string, path string, body io.Reader) (int, []byte, error) {
	return rc.SendBasicRequest(method, path, body, true)
}

func (rc RestClient) SendBasicRequest(method string, path string, body io.Reader, useHeaders bool) (statusCode int, responseBody []byte, err error) {
	requestUrl := fmt.Sprintf("%s/%s", rc.url, path)
	request, err := http.NewRequest(method, requestUrl, body)
	if err != nil {
		return
	}
	if useHeaders {
		request.Header.Add("Accept", "application/json")
		request.Header.Add("Content-Type", "application/json")
	}
	if rc.credentials.Username != "" && rc.credentials.Password != "" {
		request.SetBasicAuth(rc.credentials.Username, rc.credentials.Password)
	}
	response, err := rc.httpClient.Do(request)
	if err != nil {
		return
	}
	defer response.Body.Close()
	statusCode = response.StatusCode
	responseBody, err = io.ReadAll(response.Body)
	return
}

func (rc RestClient) SendRequestWithStatusCodeCheck(method string, path string, body io.Reader) ([]byte, error) {
	statusCode, responseBody, err := rc.SendRequest(method, path, body)
	if statusCode >= 400 {
		return responseBody, fmt.Errorf("%s request to %s/%s returned [%d] status code: %s", method, rc.url,
			path, statusCode, responseBody)
	}
	return responseBody, err
}

func (rc RestClient) GetArrayData(path, key string, filter func(string) bool) ([]string, error) {
	arrayData := make([]string, 0, 64)
	_, body, err := rc.SendRequest(http.MethodGet, path, nil)
	if err != nil {
		return arrayData, err
	}
	var bodySlice []map[string]string
	if err = json.Unmarshal(body, &bodySlice); err != nil {
		return arrayData, err
	}

	for _, data := range bodySlice {
		dataItem := data[key]
		if filter(dataItem) {
			arrayData = append(arrayData, dataItem)
		}
	}
	return arrayData, nil
}
