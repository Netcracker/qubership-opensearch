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

package client

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/Netcracker/dbaas-opensearch-adapter/common"
	"github.com/Netcracker/dbaas-opensearch-adapter/health"
)

var logger = common.GetLogger()

const (
	certificateFilePath        = "/tls/ca.crt"
	curatorCertificateFilePath = "/tls/curator/ca.crt"
)

type AdapterClient struct {
	proto      string
	host       string
	port       int
	username   string
	password   string
	httpClient *http.Client
}

type DbResource struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
}

type CreatedDatabase struct {
	Name      string       `json:"name"`
	Resources []DbResource `json:"resources"`
}

type Smoke struct {
	client    AdapterClient
	createdDb CreatedDatabase
}

func NewAdapterClient(proto string, host string, port int, username, password string) *AdapterClient {
	httpClient := ConfigureClient()
	return &AdapterClient{
		proto:      proto,
		host:       host,
		port:       port,
		username:   username,
		password:   password,
		httpClient: httpClient,
	}
}

func (client AdapterClient) Exec(command string) bool {
	args := strings.Fields(command)
	if len(args) == 0 {
		return false
	}
	switch args[0] {
	case "health":
		fmt.Println(client.ReceiveHealth())
		return true
	case "smoke":
		fmt.Println(client.DoSmoke())
		return true
	}
	return false
}

func (client AdapterClient) ReceiveHealth() string {
	address := fmt.Sprintf("%s://%s:%d/health", client.proto, client.host, client.port)
	logger.Info(fmt.Sprintf("Get health on '%s' address", address))
	response, err := http.Get(address)
	if err != nil {
		logger.Error("Failed to get health", slog.Any("error", err))
		return "PROBLEM"
	}
	defer response.Body.Close()

	var adapterHealth health.Health
	err = common.ProcessBody(response.Body, &adapterHealth)
	if err != nil {
		logger.Error("Failed to unmarshal response as JSON", slog.Any("error", err))
		return "PROBLEM"
	}
	return adapterHealth.Status
}

func (client AdapterClient) DoSmoke() string {
	logger.Debug(fmt.Sprintf("Start smoke on %s://%s:%d", client.proto, client.host, client.port))
	var err error
	smoke := Smoke{
		client: client,
	}
	smoke.createdDb, err = smoke.createDatabase()
	if err != nil {
		return "FAIL"
	}
	if err = smoke.dropDatabase(); err != nil {
		return "FAIL"
	}
	return "OK"
}

func (smoke Smoke) createDatabase() (CreatedDatabase, error) {
	logger.Info("Creating database...")
	client := smoke.client
	request, err := http.NewRequest(http.MethodPost,
		fmt.Sprintf("%s://%s:%d%s/databases", client.proto, client.host, client.port, common.BasePath),
		bytes.NewBuffer([]byte(`{
				"namePrefix": "smoketest",
				"metadata": {
					"classifier": {
						"microserviceName": "smoke-service"
					}
				}
			}
			`,
		)),
	)
	request.Header.Set("Content-Type", "application/json")
	if err != nil {
		logger.Error("Failed to create database", slog.Any("error", err))
		return CreatedDatabase{}, err
	}
	request.SetBasicAuth(client.username, client.password)
	response, err := client.httpClient.Do(request)
	if err != nil {
		logger.Error("Failed to create database", slog.Any("error", err))
		return CreatedDatabase{}, err
	}
	defer response.Body.Close()
	logger.Debug(fmt.Sprintf("Received response status: %s", response.Status))
	var created CreatedDatabase
	err = common.ProcessBody(response.Body, &created)
	if err != nil {
		return CreatedDatabase{}, err
	}
	return created, nil
}

func (smoke Smoke) dropDatabase() error {
	logger.Info("Deleting database...")
	client := smoke.client
	resources, err := json.Marshal(smoke.createdDb.Resources)
	if err != nil {
		logger.Error("Failed to drop database", slog.Any("error", err))
		return err
	}
	request, err := http.NewRequest("POST",
		fmt.Sprintf("%s://%s:%d%s/resources/bulk-drop", client.proto, client.host, client.port, common.BasePath),
		bytes.NewBuffer(resources),
	)
	request.Header.Set("Content-Type", "application/json")
	if err != nil {
		logger.Error("Failed to drop database", slog.Any("error", err))
		return err
	}
	request.SetBasicAuth(client.username, client.password)
	response, err := client.httpClient.Do(request)
	if err != nil {
		logger.Error("Failed to drop database", slog.Any("error", err))
		return err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		logger.Error("Failed to read response body", slog.Any("error", err))
		return err
	}
	logger.Debug(fmt.Sprintf("Response status is %s, body is %s", response.Status, string(body)))
	return nil
}

func ConfigureClient() *http.Client {
	return ConfigureHttpClient([]string{certificateFilePath})
}

func ConfigureCuratorClient() *http.Client {
	return ConfigureHttpClient([]string{certificateFilePath, curatorCertificateFilePath})
}

func ConfigureHttpClient(certPaths []string) *http.Client {
	httpClient := &http.Client{}
	caCertPool := x509.NewCertPool()
	successfullyAppendedCerts := 0
	for _, certPath := range certPaths {
		if _, err := os.Stat(certPath); !errors.Is(err, os.ErrNotExist) {
			caCert, readErr := os.ReadFile(certPath)
			if readErr != nil {
				logger.Error(fmt.Sprintf("Unable to read certificates by %s path", certPath), slog.Any("error", readErr))
				continue
			}
			if caCertPool.AppendCertsFromPEM(caCert) {
				successfullyAppendedCerts++
			}
		}
	}
	if successfullyAppendedCerts == 0 {
		logger.Info("Cannot load valid TLS certificates. Using client without TLS")
		return httpClient
	}
	httpClient.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs: caCertPool,
		},
	}
	return httpClient
}
