// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/agntcy/csit/benchmarks/slim-vs-a2a/internal/plan"
)

func main() {
	family := flag.String("family", "sustainable-resource", "canonical family: preference-aggregation, supply-chain-cascade, sustainable-resource")
	agents := flag.Int("agents", 5, "number of agents")
	fanOut := flag.Int("fan-out", 0, "context fan-out (0 = all agents)")
	roundBudgetMS := flag.Int64("round-budget-ms", 20, "coordination round budget ms")
	executeSec := flag.Float64("execute-sec", 0.01, "task execute sleep seconds")
	rounds := flag.Int("rounds", 1, "rounds for sustainable-resource family")
	output := flag.String("output", "", "write plan yaml to path (default stdout)")
	flag.Parse()

	p, err := plan.GenerateCanonical(plan.GenerateOptions{
		Family:          plan.CanonicalFamily(*family),
		Agents:          *agents,
		FanOut:          *fanOut,
		RoundBudgetMS:   *roundBudgetMS,
		ExecuteSleepSec: *executeSec,
		Rounds:          *rounds,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "generate: %v\n", err)
		os.Exit(1)
	}

	if *output == "" {
		data, err := yaml.Marshal(p)
		if err != nil {
			fmt.Fprintf(os.Stderr, "marshal: %v\n", err)
			os.Exit(1)
		}
		os.Stdout.Write(data)
		return
	}
	if err := plan.WriteFile(*output, p); err != nil {
		fmt.Fprintf(os.Stderr, "write: %v\n", err)
		os.Exit(1)
	}
}
