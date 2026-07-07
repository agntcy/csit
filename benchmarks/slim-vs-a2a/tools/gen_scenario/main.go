// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/agntcy/csit/benchmarks/slim-vs-a2a/internal/scenario"
)

func main() {
	family := flag.String("family", "hypothesis-convergence", "scenario family name")
	agents := flag.Int("agents", 10, "number of agents")
	thinkMS := flag.Int64("think-ms", 10, "think time per agent in ms")
	seed := flag.Int64("seed", 42, "random seed")
	payloadBytes := flag.Int("payload-bytes", 0, "fixed-size padding added to every finding (bytes)")
	epochs := flag.Int("epochs", 0, "number of consensus epochs to run (default 10)")
	maxEpochMS := flag.Int64("max-epoch-ms", 0, "per-epoch consensus time budget in ms (default 120000)")
	output := flag.String("output", "", "output yaml path")
	flag.Parse()

	if *output == "" {
		log.Fatal("-output is required")
	}

	s, err := scenario.Generate(scenario.GenerateOptions{
		Family:         *family,
		Agents:         *agents,
		ThinkTimeMs:    *thinkMS,
		Seed:           *seed,
		PayloadBytes:   *payloadBytes,
		Epochs:         *epochs,
		MaxEpochTimeMs: *maxEpochMS,
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
