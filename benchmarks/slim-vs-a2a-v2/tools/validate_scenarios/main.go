// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/agntcy/csit/benchmarks/slim-vs-a2a-v2/internal/scenario"
)

func main() {
	dir := flag.String("dir", "./plans/sweeps", "directory containing scenario yaml files")
	flag.Parse()

	entries, err := os.ReadDir(*dir)
	if err != nil {
		log.Fatalf("read dir: %v", err)
	}
	var failed int
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		path := filepath.Join(*dir, e.Name())
		if _, err := scenario.LoadFile(path); err != nil {
			fmt.Printf("INVALID %s: %v\n", path, err)
			failed++
		} else {
			fmt.Printf("OK %s\n", path)
		}
	}
	if failed > 0 {
		os.Exit(1)
	}
}
