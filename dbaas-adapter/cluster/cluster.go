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

package cluster

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/Netcracker/dbaas-opensearch-adapter/common"
	"github.com/opensearch-project/opensearch-go"
	"github.com/opensearch-project/opensearch-go/opensearchapi"
)

var logger = common.GetLogger()

type Opensearch struct {
	Host     string
	Port     int
	Protocol string
	Health   common.ComponentHealth
	Client   common.Client
}

const trustCertsFolder = "/trusted-certs"

func NewOpensearch(host string, port int, protocol string, username string, password string) *Opensearch {
	address := fmt.Sprintf("%s://%s:%d", protocol, host, port)
	logger.Info(fmt.Sprintf("Creating new OpenSearch on '%s' address", address))

	var transport *http.Transport
	if strings.EqualFold(protocol, common.Https) {
		certsDir, err := os.ReadDir(trustCertsFolder)
		if err != nil || len(certsDir) == 0 {
			logger.Info(fmt.Sprintf("Cannot load trusted TLS certificates from path '%s'. InsecureSkipVerify is used.", trustCertsFolder))
			transport = &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			}
		} else {
			certs := x509.NewCertPool()
			successfullyAppendedCerts := 0
			for _, cert := range certsDir {
				if common.IsNotDir(cert) {
					pemData, err := os.ReadFile(fmt.Sprintf("%s/%s", trustCertsFolder, cert.Name()))
					if err != nil {
						logger.Error(fmt.Sprintf("Failed to read certificate '%s': %+v", cert.Name(), err))
						panic(err)
					}
					if certs.AppendCertsFromPEM(pemData) {
						successfullyAppendedCerts++
					}
					logger.Info(fmt.Sprintf("Trusted certificate '%s' was added to client", cert.Name()))
				}
			}
			if successfullyAppendedCerts == 0 {
				logger.Warn(fmt.Sprintf("Cannot load valid trusted TLS certificates from path '%s'. InsecureSkipVerify mode is used. Do not use this mode in production.", trustCertsFolder))
				transport = &http.Transport{
					TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
				}
			} else {
				transport = &http.Transport{
					TLSClientConfig: &tls.Config{RootCAs: certs},
				}
			}
		}
	}

	config := opensearch.Config{
		Addresses: []string{address},
		Username:  username,
		Password:  password,
	}
	if transport != nil {
		config.Transport = transport
	}
	oc, err := opensearch.NewClient(config)
	if err != nil {
		logger.Error("Failed to connect to OpenSearch", slog.Any("error", err))
		panic(err)
	}

	service := &Opensearch{
		Host:     host,
		Port:     port,
		Protocol: protocol,
		Health:   common.ComponentHealth{Status: common.Up},
		Client:   oc,
	}

	service.Health.Status = service.GetHealth(context.Background())

	return service
}

func (o Opensearch) GetHealth(ctx context.Context) string {
	healthRequest := opensearchapi.CatHealthRequest{
		Format: "json",
	}
	var componentHealth []common.ComponentHealth
	err := common.DoRequest(healthRequest, o.Client, &componentHealth, ctx)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to get cluster health", slog.Any("error", err))
		return "PROBLEM"
	}
	switch componentHealth[0].Status {
	case "green":
		return "UP"
	case "red":
		return "DOWN"
	case "yellow":
		return "WARNING"
	default:
		return "PROBLEM"
	}
}
