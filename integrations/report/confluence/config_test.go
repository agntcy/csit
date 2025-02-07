// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package confluence

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewConfig(t *testing.T) {
	tests := []struct {
		name       string
		envVars    map[string]string
		wantConfig *Config
	}{
		{
			name:    "Default values",
			envVars: map[string]string{},
			wantConfig: &Config{
				Location: "",
				Username: "",
				PAT:      "",
				SpaceID:  "",
				ParentID: "",
			},
		},
		{
			name: "Custom values",
			envVars: map[string]string{
				"CONFLUENCE_LOCATION":  "https://example.com",
				"CONFLUENCE_USERNAME":  "john_doe@cisco.com",
				"CONFLUENCE_PAT":       "asdf1234",
				"CONFLUENCE_SPACE_ID":  "5678",
				"CONFLUENCE_PARENT_ID": "9876",
			},
			wantConfig: &Config{
				Location: "https://example.com",
				Username: "john_doe@cisco.com",
				PAT:      "asdf1234",
				SpaceID:  "5678",
				ParentID: "9876",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.envVars {
				err := os.Setenv(k, v)
				if err != nil {
					return
				}
			}

			t.Cleanup(func() {
				os.Clearenv()
			})

			cfg, err := NewConfig()
			if err != nil {
				assert.EqualError(t, nil, err.Error(), "Unexpected error message")
			}

			assert.Equal(t, tt.wantConfig, cfg, "Unexpected config")
		})
	}
}
