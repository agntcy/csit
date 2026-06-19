// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/agntcy/csit/benchmarks/slim-vs-a2a/internal/plan"
)

func main() {
	dir := flag.String("dir", filepath.Join("plans", "domains"), "directory containing plan yaml files")
	flag.Parse()

	plans, err := plan.LoadDir(*dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load plans: %v\n", err)
		os.Exit(1)
	}
	for _, p := range plans {
		fmt.Printf("%s: %d agents, %d tasks, %d contextUpdates\n",
			p.Metadata.Name, len(p.Agents), len(p.Tasks), len(p.ContextUpdates))
	}
}
