// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/onsi/gomega"
)

func TestWriteAndLoadC2EvidenceDocument(t *testing.T) {
	gomega.RegisterFailHandler(func(message string, callerSkip ...int) {
		t.Helper()
		t.Fatal(message)
	})

	dir := t.TempDir()
	path := filepath.Join(dir, "c2-evidence.json")

	row := c2EvidenceCase{
		RowID:     "c2-topology-routing",
		Scenario:  "isolated-routes",
		Mechanism: "declarative-routes",
		UseCase:   "Multi-agent flow over fixed, named routes",
		Status:    "verified",
		Assertions: []string{
			"alice delivered to bob",
		},
	}

	gomega.Expect(upsertC2EvidenceCase(path, row)).To(gomega.Succeed())

	loaded, err := loadC2EvidenceDocument(path)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(loaded.SchemaVersion).To(gomega.Equal("1"))
	gomega.Expect(loaded.Cases).To(gomega.HaveLen(1))
	gomega.Expect(loaded.Cases[0].Scenario).To(gomega.Equal("isolated-routes"))

	updated := row
	updated.Status = "failed"
	gomega.Expect(upsertC2EvidenceCase(path, updated)).To(gomega.Succeed())

	loaded, err = loadC2EvidenceDocument(path)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(loaded.Cases).To(gomega.HaveLen(1))
	gomega.Expect(loaded.Cases[0].Status).To(gomega.Equal("failed"))

	content, err := os.ReadFile(path)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(string(content)).To(gomega.ContainSubstring("\"row_id\": \"c2-topology-routing\""))
}
