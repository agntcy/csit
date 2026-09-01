// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tests

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const c2EvidenceSchemaVersion = "1"

type c2EvidenceDocument struct {
	SchemaVersion string           `json:"schema_version"`
	GeneratedAt   string           `json:"generated_at"`
	Source        string           `json:"source"`
	Cases         []c2EvidenceCase `json:"cases"`
}

type c2EvidenceCase struct {
	RowID      string   `json:"row_id"`
	Scenario   string   `json:"scenario"`
	Mechanism  string   `json:"mechanism"`
	UseCase    string   `json:"use_case"`
	Status     string   `json:"status"`
	Assertions []string `json:"assertions"`
}

func c2EvidenceReportPath() string {
	reportDir := os.Getenv("C2_EVIDENCE_REPORT_DIR")
	if reportDir == "" {
		reportDir = "../reports"
	}
	absDir, err := filepath.Abs(reportDir)
	if err != nil {
		panic(fmt.Sprintf("resolve C2 evidence report dir: %v", err))
	}
	return filepath.Join(absDir, "c2-evidence.json")
}

func loadC2EvidenceDocument(path string) (c2EvidenceDocument, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return c2EvidenceDocument{}, err
	}
	var doc c2EvidenceDocument
	if err := json.Unmarshal(content, &doc); err != nil {
		return c2EvidenceDocument{}, err
	}
	return doc, nil
}

func writeC2EvidenceDocument(path string, cases []c2EvidenceCase) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	doc := c2EvidenceDocument{
		SchemaVersion: c2EvidenceSchemaVersion,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		Source:        "integrations/agntcy-slim/topology/tests/topology_test.go",
		Cases:         cases,
	}

	encoded, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	encoded = append(encoded, '\n')
	return os.WriteFile(path, encoded, 0o644)
}

func upsertC2EvidenceCase(path string, row c2EvidenceCase) error {
	doc, err := loadC2EvidenceDocument(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		doc = c2EvidenceDocument{
			SchemaVersion: c2EvidenceSchemaVersion,
			Source:        "integrations/agntcy-slim/topology/tests/topology_test.go",
		}
	}

	updated := false
	for idx, existing := range doc.Cases {
		if existing.RowID == row.RowID && existing.Scenario == row.Scenario {
			doc.Cases[idx] = row
			updated = true
			break
		}
	}
	if !updated {
		doc.Cases = append(doc.Cases, row)
	}

	doc.GeneratedAt = time.Now().UTC().Format(time.RFC3339)
	return writeC2EvidenceDocument(path, doc.Cases)
}

func logC2EvidenceSummary(row c2EvidenceCase) {
	fmt.Printf(
		"C2_EVIDENCE row_id=%s scenario=%s status=%s assertions=%d\n",
		row.RowID,
		row.Scenario,
		row.Status,
		len(row.Assertions),
	)
}
