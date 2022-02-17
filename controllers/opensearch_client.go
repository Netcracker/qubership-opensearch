package controllers

import (
	"fmt"
	"io"
	"io/ioutil"
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

func (rc RestClient) sendRequest(method string, path string, body io.Reader) (statusCode int, responseBody []byte, err error) {
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
	statusCode = response.StatusCode
	responseBody, err = ioutil.ReadAll(response.Body)
	return
}
