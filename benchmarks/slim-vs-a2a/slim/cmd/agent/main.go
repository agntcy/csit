// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

	slim "github.com/agntcy/slim-bindings-go"
	"github.com/agntcy/csit/benchmarks/slim-vs-a2a/slim/internal/executor"
	"github.com/agntcy/csit/benchmarks/slim-vs-a2a/slim/internal/server"
	"github.com/agntcy/csit/benchmarks/slim-vs-a2a/slim/internal/slimrpc"
)

const defaultSharedSecret = "demo-shared-secret-min-32-chars!!"

func main() {
	slimName := flag.String("slim-name", "", "SLIM identity org/group/app")
	endpoint := flag.String("endpoint", "http://127.0.0.1:46357", "SLIM dataplane endpoint")
	flag.Parse()

	if *slimName == "" {
		log.Fatal("--slim-name is required")
	}

	slim.InitializeWithDefaults()
	service := slim.GetGlobalService()

	name, err := nameFromString(*slimName)
	if err != nil {
		log.Fatalf("parse name: %v", err)
	}

	app, err := service.CreateAppWithSecret(name, defaultSharedSecret)
	if err != nil {
		log.Fatalf("create app: %v", err)
	}
	defer app.Destroy()

	connID, err := service.Connect(slim.NewInsecureClientConfig(*endpoint))
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	if err := app.Subscribe(name, &connID); err != nil {
		log.Fatalf("subscribe: %v", err)
	}

	rpcServer := slim.ServerNewWithConnection(app, name, &connID)
	rpcServer.RegisterUnaryUnary(slimrpc.ServiceName, slimrpc.MethodHandle, &server.Handler{
		Engine: executor.New(*slimName),
	})

	fmt.Printf("SLIM_AGENT_READY name=%s\n", *slimName)
	if err := rpcServer.Serve(); err != nil {
		log.Printf("slimrpc server: %v", err)
	}
}

func nameFromString(value string) (*slim.Name, error) {
	parts := strings.Split(value, "/")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid name format: %s", value)
	}
	return slim.NewName(parts[0], parts[1], parts[2]), nil
}
