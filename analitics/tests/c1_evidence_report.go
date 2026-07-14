// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tests

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/agntcy/csit/analitics/harness"
)

const c1EvidenceSchemaVersion = "1"

type c1EvidenceDocument struct {
	SchemaVersion string           `json:"schema_version"`
	GeneratedAt   string           `json:"generated_at"`
	Source        string           `json:"source"`
	Cases         []c1EvidenceCase `json:"cases"`
}

type c1EvidenceCase struct {
	RowID          string   `json:"row_id"`
	Mode           string   `json:"mode"`
	UseCase        string   `json:"use_case"`
	Status         string   `json:"status"`
	SenderMessages int64    `json:"sender_messages"`
	SenderErrors   int64    `json:"sender_errors"`
	SinkReceived   int64    `json:"sink_received,omitempty"`
	SinkReplies    int64    `json:"sink_replies,omitempty"`
	SinkErrors     int64    `json:"sink_errors,omitempty"`
	MeanLatencyMS  float64  `json:"mean_latency_ms,omitempty"`
	Assertions     []string `json:"assertions"`
}

func c1EvidenceReportPath() string {
	reportDir := harness.EnvString("C1_EVIDENCE_REPORT_DIR", "./reports")
	absDir, err := filepath.Abs(reportDir)
	if err != nil {
		panic(fmt.Sprintf("resolve C1 evidence report dir: %v", err))
	}
	return filepath.Join(absDir, "c1-evidence.json")
}

func loadC1EvidenceDocument(path string) (c1EvidenceDocument, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return c1EvidenceDocument{}, err
	}
	var doc c1EvidenceDocument
	if err := json.Unmarshal(content, &doc); err != nil {
		return c1EvidenceDocument{}, err
	}
	return doc, nil
}

func writeC1EvidenceDocument(path string, cases []c1EvidenceCase) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	doc := c1EvidenceDocument{
		SchemaVersion: c1EvidenceSchemaVersion,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		Source:        "analitics/tests/c1_evidence_test.go",
		Cases:         cases,
	}

	encoded, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	encoded = append(encoded, '\n')
	return os.WriteFile(path, encoded, 0o644)
}

func upsertC1EvidenceCase(path string, row c1EvidenceCase) error {
	doc, err := loadC1EvidenceDocument(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		doc = c1EvidenceDocument{
			SchemaVersion: c1EvidenceSchemaVersion,
			Source:        "analitics/tests/c1_evidence_test.go",
		}
	}

	updated := false
	for idx, existing := range doc.Cases {
		if existing.RowID == row.RowID {
			doc.Cases[idx] = row
			updated = true
			break
		}
	}
	if !updated {
		doc.Cases = append(doc.Cases, row)
	}

	doc.GeneratedAt = time.Now().UTC().Format(time.RFC3339)
	return writeC1EvidenceDocument(path, doc.Cases)
}

func logC1EvidenceSummary(row c1EvidenceCase) {
	_ = harness.LogProgress(
		"C1_EVIDENCE row_id=%s mode=%s status=%s sender_messages=%d sender_errors=%d sink_received=%d sink_replies=%d sink_errors=%d",
		row.RowID,
		row.Mode,
		row.Status,
		row.SenderMessages,
		row.SenderErrors,
		row.SinkReceived,
		row.SinkReplies,
		row.SinkErrors,
	)
}
