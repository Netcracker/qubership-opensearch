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

package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"github.com/Netcracker/dbaas-opensearch-adapter/client"
	"github.com/Netcracker/dbaas-opensearch-adapter/common"
	"github.com/Netcracker/dbaas-opensearch-adapter/server"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
)

var (
	//nolint:errcheck
	tlsEnabled, _   = strconv.ParseBool(common.GetEnv("TLS_ENABLED", "false"))
	adapterPort     = 8080
	adapterProtocol = common.Http
	adapterUsername = common.GetEnv("DBAAS_ADAPTER_USERNAME", "dbaas-aggregator")
	adapterPassword = common.GetEnv("DBAAS_ADAPTER_PASSWORD", "dbaas-aggregator")
	adapterAddress  = common.GetEnv("DBAAS_ADAPTER_ADDRESS", "")

	buildstamp  string
	githash     string
	mode        = flag.String("mode", "", "Specify \"shell\" to run in shell mode, shell mode would also be enabled if first argument is sh")
	interactive = flag.Bool("i", false, "Enables shell mode")
	command     = flag.String("c", "", "Command to run in shell mode")

	logger = common.GetLogger()
)

func main() {

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger.Info(fmt.Sprintf("Run build %s / %s with %+v ...", buildstamp, githash, os.Args))
	flag.Parse()
	if tlsEnabled {
		adapterPort = 8443
		adapterProtocol = common.Https
	}
	cl := client.NewAdapterClient(adapterProtocol, "", adapterPort, adapterUsername, adapterPassword)
	if *command != "" {
		if cl.Exec(*command) {
			return
		}
	}
	if *interactive || *mode == "shell" || (*mode == "" && (os.Args[0] == "sh" || strings.HasSuffix(os.Args[0], "/sh"))) {
		reader := bufio.NewReader(os.Stdin)
		var enteredCommand string
		for enteredCommand != "exit" {
			terminal(reader, cl)
		}
		return
	}

	server.Server(ctx, adapterAddress, adapterUsername, adapterPassword)
}

func terminal(reader *bufio.Reader, cl *client.AdapterClient) {
	defer func() { // error handler, when error occurred in command processing
		if err := recover(); err != nil {
			fmt.Printf("Error during command execution: %v\n", err)
		}
	}()
	fmt.Print("dbaas_opensearch> ")
	line, err := reader.ReadString('\n')
	if err != nil {
		log.Println(err.Error())
		return
	}
	fmt.Println(line)
	cl.Exec(line)
}
