package util

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type RestClient struct {
	url         string
	httpClient  http.Client
	credentials []string
}

func NewRestClient(url string, httpClient http.Client, credentials []string) *RestClient {
	return &RestClient{
		url:         url,
		httpClient:  httpClient,
		credentials: credentials,
	}
}

func (rc RestClient) SendRequest(method string, path string, body io.Reader) (statusCode int, responseBody []byte, err error) {
	requestUrl := fmt.Sprintf("%s/%s", rc.url, path)
	request, err := http.NewRequest(method, requestUrl, body)
	if err != nil {
		return
	}
	request.Header.Add("Accept", "application/json")
	request.Header.Add("Content-Type", "application/json")
	if len(rc.credentials) == 2 {
		request.SetBasicAuth(rc.credentials[0], rc.credentials[1])
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
	if statusCode >= 500 {
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
