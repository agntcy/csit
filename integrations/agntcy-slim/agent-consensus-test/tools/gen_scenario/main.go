// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/agntcy/csit/integrations/agntcy-slim/agent-consensus-test/internal/scenario"
)

func main() {
	family := flag.String("family", "hypothesis-convergence", "scenario family name")
	agents := flag.Int("agents", 10, "number of agents")
	thinkMS := flag.Int64("think-ms", 10, "think time per agent in ms")
	seed := flag.Int64("seed", 42, "random seed")
	payloadBytes := flag.Int("payload-bytes", 0, "fixed-size padding added to every finding (bytes)")
	epochs := flag.Int("epochs", 0, "number of consensus epochs to run (default 10)")
	maxEpochMS := flag.Int64("max-epoch-ms", 0, "per-epoch consensus time budget in ms (default 120000)")
	latencyMS := flag.Int64("latency-ms", 0, "one-way delay applied to a subset of agents in ms (distant-region simulation)")
	latencyCount := flag.Int("latency-count", 0, "how many trailing agents get latency-ms (default round(agents/3) when latency-ms>0)")
	matrixNaming := flag.Bool("matrix-naming", false, "use matrix-lat{L}-{N}ag[-{B}b] filename stem")
	output := flag.String("output", "", "output yaml path")
	flag.Parse()

	if *output == "" {
		log.Fatal("-output is required")
	}

	familyName := *family
	if *matrixNaming && familyName == "hypothesis-convergence" {
		familyName = "matrix"
	}

	s, err := scenario.Generate(scenario.GenerateOptions{
		Family:         familyName,
		Agents:         *agents,
		ThinkTimeMs:    *thinkMS,
		Seed:           *seed,
		PayloadBytes:   *payloadBytes,
		Epochs:         *epochs,
		MaxEpochTimeMs: *maxEpochMS,
		LatencyMs:      *latencyMS,
		LatencyCount:   *latencyCount,
		MatrixNaming:   *matrixNaming,
	})
	if err != nil {
		log.Fatalf("generate: %v", err)
	}

	if err := os.MkdirAll(filepath.Dir(*output), 0o755); err != nil {
		log.Fatalf("mkdir: %v", err)
	}
	if err := scenario.WriteFile(*output, s); err != nil {
		log.Fatalf("write: %v", err)
	}
	fmt.Printf("wrote %s\n", *output)
}
