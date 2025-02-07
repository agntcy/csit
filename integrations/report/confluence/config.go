// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package confluence

import (
	"fmt"

	"github.com/spf13/viper"
)

const (
	defaultEnvPrefix = "CONFLUENCE"
)

type Config struct {
	Location string `mapstructure:"location"`
	Username string `mapstructure:"username"`
	PAT      string `mapstructure:"pat"`
	SpaceID  string `mapstructure:"space_id"`
	ParentID string `mapstructure:"parent_id"`
}

func NewConfig() (*Config, error) {
	v := viper.New()

	v.SetEnvPrefix(defaultEnvPrefix)
	v.AutomaticEnv()

	_ = v.BindEnv("location")
	_ = v.BindEnv("username")
	_ = v.BindEnv("pat")
	_ = v.BindEnv("space_id")
	_ = v.BindEnv("parent_id")

	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse configuration: %w", err)
	}

	return cfg, nil
}
