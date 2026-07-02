// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBuildEventViews(t *testing.T) {
	events := []specEvent{
		{
			SpecEventType: "Node",
			Message:       "Slim topology test",
			NodeType:      "BeforeAll",
			TimelineLocation: timelineLocation{
				Order: 4,
				Time:  time.Date(2026, 6, 10, 8, 7, 24, 0, time.UTC),
			},
			CodeLocation: failureLocation{
				FileName:   "/home/runner/work/csit/csit/integrations/agntcy-slim/topology/tests/topology_test.go",
				LineNumber: 147,
			},
		},
		{
			SpecEventType: "By",
			Message:       "Waiting for alice to show received: hello message",
			TimelineLocation: timelineLocation{
				Order: 8,
				Time:  time.Date(2026, 6, 10, 8, 8, 27, 0, time.UTC),
			},
			CodeLocation: failureLocation{
				FileName:   "/home/runner/work/csit/csit/integrations/agntcy-slim/topology/tests/topology_test.go",
				LineNumber: 278,
			},
		},
		{
			SpecEventType: "Node (End)",
			Message:       "Slim topology test",
			NodeType:      "DeferCleanup (Each)",
			Duration:      5496141,
			TimelineLocation: timelineLocation{
				Order: 29,
				Time:  time.Date(2026, 6, 10, 8, 8, 27, 0, time.UTC),
			},
		},
	}

	views := buildEventViews(events)
	if len(views) != 3 {
		t.Fatalf("expected 3 events, got %d", len(views))
	}
	if views[0].EventClass != "event-node" {
		t.Fatalf("unexpected class for node event: %s", views[0].EventClass)
	}
	if views[1].EventClass != "event-by" {
		t.Fatalf("unexpected class for by event: %s", views[1].EventClass)
	}
	if views[2].EventClass != "event-end" {
		t.Fatalf("unexpected class for end event: %s", views[2].EventClass)
	}
	if views[0].Source != "tests/topology_test.go:147" {
		t.Fatalf("unexpected short source: %s", views[0].Source)
	}
	if views[1].Message != "Waiting for alice to show received: hello message" {
		t.Fatalf("unexpected message: %s", views[1].Message)
	}
}

func TestRenderDashboardIncludesTimeline(t *testing.T) {
	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "report-slim-topology.json")
	fixture := `[{
		"SuiteDescription": "Tests Suite",
		"SuiteSucceeded": true,
		"PreRunStats": {"TotalSpecs": 1, "SpecsThatWillRun": 1},
		"StartTime": "2026-06-10T08:07:24.872782124Z",
		"EndTime": "2026-06-10T08:08:27.354121628Z",
		"RunTime": 62481339504,
		"SuiteConfig": {"LabelFilter": "", "RandomSeed": 1},
		"SpecReports": [{
			"ContainerHierarchyTexts": ["Agntcy slim topology test", "Slim topology test"],
			"LeafNodeText": "Create SLIM client Pods",
			"State": "passed",
			"RunTime": 62480269466,
			"SpecEvents": [{
				"SpecEventType": "By",
				"Message": "Waiting for bob to show Sent message hello - 10/10 message",
				"TimelineLocation": {"Order": 9, "Time": "2026-06-10T08:08:27.327461626Z"},
				"CodeLocation": {
					"FileName": "/home/runner/work/csit/csit/integrations/agntcy-slim/topology/tests/topology_test.go",
					"LineNumber": 278
				}
			}]
		}]
	}]`
	if err := os.WriteFile(jsonPath, []byte(fixture), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	view, err := buildDashboard("Slim Topology Test Report", dir)
	if err != nil {
		t.Fatalf("build dashboard: %v", err)
	}
	if len(view.Reports) != 1 || len(view.Reports[0].Specs) != 1 {
		t.Fatalf("unexpected report structure: %+v", view.Reports)
	}
	if len(view.Reports[0].Specs[0].Events) != 1 {
		t.Fatalf("expected spec events in view")
	}

	html, err := renderDashboard(view)
	if err != nil {
		t.Fatalf("render dashboard: %v", err)
	}
	body := string(html)
	for _, want := range []string{
		"Timeline (1 events)",
		"Waiting for bob to show Sent message hello - 10/10 message",
		"tests/topology_test.go:278",
		"event-by",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("rendered html missing %q", want)
		}
	}
}
