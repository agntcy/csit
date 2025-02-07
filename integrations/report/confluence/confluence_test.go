// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package confluence

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPublishReport(t *testing.T) {
	t.Run("PublishReport successfully publishes a report", func(t *testing.T) {
		// Mock server to simulate Confluence API
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost && r.URL.Path == "/wiki/api/v2/pages" {
				// Validate page creation payload
				body, err := io.ReadAll(r.Body)
				require.NoError(t, err)

				var page Page
				err = json.Unmarshal(body, &page)
				require.NoError(t, err)

				assert.Equal(t, "template-space-id", page.SpaceID)
				assert.Equal(t, "current", page.Status)
				assert.Equal(t, "template-parent-id", page.ParentID)
				assert.Contains(t, page.Body.Value, "<h1>Test Report</h1>")

				w.WriteHeader(http.StatusOK)
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer server.Close()

		// Set up Confluence client
		cfg := &Config{
			Location: server.URL,
			Username: "test-user",
			PAT:      "test-pat",
			SpaceID:  "template-space-id",
			ParentID: "template-parent-id",
		}
		client := New(cfg, "testdata/report.json", "123456789")
		client.Client = server.Client()

		// Run PublishReport
		ctx := context.Background()
		err := client.PublishReport(ctx)
		assert.NoError(t, err)
	})

	t.Run("PublishReport handles upload error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost {
				w.WriteHeader(http.StatusInternalServerError)
			}
		}))
		defer server.Close()

		cfg := &Config{
			Location: server.URL,
			Username: "test-user",
			PAT:      "test-pat",
			SpaceID:  "template-space-id",
			ParentID: "template-parent-id",
		}
		client := New(cfg, "testdata/report.json", "123456789")
		client.Client = server.Client()

		ctx := context.Background()
		err := client.PublishReport(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create page")
	})
}
