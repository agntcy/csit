// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package confluence

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/agntcy/csit/integrations/report/template"
)

type Confluence struct {
	Client *http.Client

	location string
	username string
	PAT      string
	spaceID  string
	parentID string
	filePath string
	runID    string
}

type Template struct {
	Body struct {
		Storage struct {
			Value string `json:"value"`
		} `json:"storage"`
	} `json:"body"`
}

type Body struct {
	Representation string `json:"representation"`
	Value          string `json:"value"`
}

type Page struct {
	SpaceID  string `json:"spaceId"`
	Status   string `json:"status"`
	Title    string `json:"title"`
	ParentID string `json:"parentId"`
	Body     Body   `json:"body"`
}

func New(cfg *Config, filePath, runID string) *Confluence {
	return &Confluence{
		Client:   &http.Client{},
		location: cfg.Location,
		username: cfg.Username,
		PAT:      cfg.PAT,
		spaceID:  cfg.SpaceID,
		parentID: cfg.ParentID,
		filePath: filePath,
		runID:    runID,
	}
}

func (c *Confluence) PublishReport(ctx context.Context) error {
	// Read test report
	report, err := readReport(c.filePath)
	if err != nil {
		return fmt.Errorf("failed to read report: %w", err)
	}

	// If parent page ID is provided, use the one from the report
	if _, ok := report["test_confluence_parent_page_id"]; ok {
		c.parentID = report["test_confluence_parent_page_id"]
	}

	// Convert report to page data
	html := template.ConvertTestReportToHTML(report, c.runID)

	// Upload page data to Confluence
	if err := c.uploadTestReport(ctx, html); err != nil {
		return fmt.Errorf("failed to upload test report: %w", err)
	}

	return nil
}

func readReport(path string) (map[string]string, error) {
	// Read the JSON file
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Read file content
	bytes, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Unmarshal JSON content to a map
	var data map[string]interface{}
	if err := json.Unmarshal(bytes, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	// Convert map to map[string]string
	output := make(map[string]string)
	for key, value := range data {
		output[key] = fmt.Sprintf("%v", value)
	}

	return output, nil
}

func (c *Confluence) uploadTestReport(ctx context.Context, pageData string) error {
	url := fmt.Sprintf("%s/wiki/api/v2/pages", c.location)
	data := Page{
		SpaceID:  c.spaceID,
		Status:   "current",
		Title:    time.Now().Format("2006-01-02 15:04:05"),
		ParentID: c.parentID,
		Body: Body{
			Representation: "storage",
			Value:          pageData,
		},
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(c.username, c.PAT)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create page: %s\nResponse: %s", resp.Status, string(body))
	}

	return nil
}
