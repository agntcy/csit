// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/agntcy/csit/benchmarks/slim-vs-a2a-v2/internal/scenario"
)

func main() {
	family := flag.String("family", "hypothesis-convergence", "scenario family name")
	agents := flag.Int("agents", 10, "number of agents")
	thinkMS := flag.Int64("think-ms", 10, "think time per agent in ms")
	seed := flag.Int64("seed", 42, "random seed")
	output := flag.String("output", "", "output yaml path")
	flag.Parse()

	if *output == "" {
		log.Fatal("-output is required")
	}

	s, err := scenario.Generate(scenario.GenerateOptions{
		Family:      *family,
		Agents:      *agents,
		ThinkTimeMs: *thinkMS,
		Seed:        *seed,
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
