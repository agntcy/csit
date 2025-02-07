// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"flag"
	"log"

	"github.com/agntcy/csit/integrations/report/confluence"
)

func main() {
	// Define command-line arguments
	filePath := flag.String("filePath", "", "Path to the JSON report file")
	runID := flag.String("runID", "", "Run ID for the report")

	// Parse command-line arguments
	flag.Parse()

	// Configuration for Confluence
	cfg, err := confluence.NewConfig()
	if err != nil {
		log.Fatalf("Failed to create Confluence config: %v", err)
	}

	// Initialize Confluence client
	client := confluence.New(cfg, *filePath, *runID)

	// Run PublishReport
	ctx := context.Background()
	err = client.PublishReport(ctx)
	if err != nil {
		log.Fatalf("Failed to publish report: %v", err)
	}

	log.Println("Report published successfully")
}
