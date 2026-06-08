// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type plan struct {
	Agents []struct {
		ID string `yaml:"id"`
	} `yaml:"agents"`
	Tasks []struct {
		ID                   string   `yaml:"id"`
		Agent                string   `yaml:"agent"`
		DependsOn            []string `yaml:"dependsOn"`
		CompletionTimeSec    float64  `yaml:"completionTimeSec"`
		MaxCompletionTimeSec float64  `yaml:"maxCompletionTimeSec"`
	} `yaml:"tasks"`
	ContextUpdates []struct {
		AfterTask    string   `yaml:"afterTask"`
		TargetAgents []string `yaml:"targetAgents"`
	} `yaml:"contextUpdates"`
}

func main() {
	dir := filepath.Join("..", "plans", "domains")
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read dir: %v\n", err)
		os.Exit(1)
	}

	var errs []string
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}

		var p plan
		if err := yaml.Unmarshal(data, &p); err != nil {
			errs = append(errs, fmt.Sprintf("%s: parse: %v", e.Name(), err))
			continue
		}

		agents := map[string]bool{}
		for _, a := range p.Agents {
			agents[a.ID] = true
		}
		tasks := map[string]struct{}{}
		for _, t := range p.Tasks {
			tasks[t.ID] = struct{}{}
		}

		for _, t := range p.Tasks {
			if !agents[t.Agent] {
				errs = append(errs, fmt.Sprintf("%s: task %s unknown agent %s", e.Name(), t.ID, t.Agent))
			}
			for _, d := range t.DependsOn {
				if _, ok := tasks[d]; !ok {
					errs = append(errs, fmt.Sprintf("%s: task %s unknown dep %s", e.Name(), t.ID, d))
				}
			}
			if t.MaxCompletionTimeSec < t.CompletionTimeSec {
				errs = append(errs, fmt.Sprintf("%s: task %s max < completion", e.Name(), t.ID))
			}
		}
		for _, cu := range p.ContextUpdates {
			if _, ok := tasks[cu.AfterTask]; !ok {
				errs = append(errs, fmt.Sprintf("%s: contextUpdate unknown afterTask %s", e.Name(), cu.AfterTask))
			}
			for _, a := range cu.TargetAgents {
				if !agents[a] {
					errs = append(errs, fmt.Sprintf("%s: contextUpdate unknown agent %s", e.Name(), a))
				}
			}
		}

		fmt.Printf("%s: %d agents, %d tasks, %d contextUpdates\n", e.Name(), len(p.Agents), len(p.Tasks), len(p.ContextUpdates))
	}

	if len(errs) > 0 {
		for _, e := range errs {
			fmt.Println("ERROR:", e)
		}
		os.Exit(1)
	}
}
