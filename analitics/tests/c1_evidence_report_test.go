package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/onsi/gomega"
)

func TestWriteAndLoadC1EvidenceDocument(t *testing.T) {
	gomega.RegisterFailHandler(func(message string, callerSkip ...int) {
		t.Helper()
		t.Fatal(message)
	})

	dir := t.TempDir()
	path := filepath.Join(dir, "c1-evidence.json")

	row := c1EvidenceCase{
		RowID:          "c1-request-reply",
		Mode:           "request-reply",
		UseCase:        "Agent A calls B and waits for a reply",
		Status:         "verified",
		SenderMessages: 20,
		Assertions:     []string{"sender completed 20 messages with 0 runtime errors"},
	}

	gomega.Expect(upsertC1EvidenceCase(path, row)).To(gomega.Succeed())

	loaded, err := loadC1EvidenceDocument(path)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(loaded.SchemaVersion).To(gomega.Equal("1"))
	gomega.Expect(loaded.Cases).To(gomega.HaveLen(1))
	gomega.Expect(loaded.Cases[0].RowID).To(gomega.Equal("c1-request-reply"))

	updated := row
	updated.SenderErrors = 1
	updated.Status = "failed"
	gomega.Expect(upsertC1EvidenceCase(path, updated)).To(gomega.Succeed())

	loaded, err = loadC1EvidenceDocument(path)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(loaded.Cases).To(gomega.HaveLen(1))
	gomega.Expect(loaded.Cases[0].SenderErrors).To(gomega.Equal(int64(1)))

	content, err := os.ReadFile(path)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(string(content)).To(gomega.ContainSubstring("\"row_id\": \"c1-request-reply\""))
}
