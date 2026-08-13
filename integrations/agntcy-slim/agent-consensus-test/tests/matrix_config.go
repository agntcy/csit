// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tests

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/agntcy/csit/integrations/agntcy-slim/agent-consensus-test/internal/scenario"
)

type matrixConfig struct {
	RootDir        string
	BinDir         string
	ReportsDir     string
	RawDir         string
	MatrixDir      string
	SlimctlPath    string
	SlimEndpoint   string
	LatencyMs      int64
	Agents         int
	PayloadBytes   int
	ThinkMs        int64
	Epochs         int
	MaxEpochMs     int64
	Repeats        int
	ScenarioPath   string
	ScenarioBase   string
	CellTSV        string
	P2PRelayGRPC   int
	P2PRelayCard   int
}

func projectRoot() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	dir := wd
	for {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return wd
		}
		dir = parent
	}
}

func resolveProjectPath(root, path string) string {
	if path == "" {
		return root
	}
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(root, filepath.Clean(path))
}

func loadMatrixConfig() matrixConfig {
	root := projectRoot()
	binDir := resolveProjectPath(root, envString("BIN_DIR", "bin"))
	reportsDir := resolveProjectPath(root, envString("REPORTS_DIR", "reports"))
	rawDirEnv := os.Getenv("MATRIX_RAW_DIR")
	rawDir := filepath.Join(reportsDir, "raw")
	if rawDirEnv != "" {
		rawDir = resolveProjectPath(root, rawDirEnv)
	}
	matrixDir := resolveProjectPath(root, envString("MATRIX_DIR", filepath.Join("plans", "matrix")))

	latency := envInt64("MATRIX_LATENCY_MS", -1)
	agents := envInt("MATRIX_AGENTS", -1)
	payload := envInt("MATRIX_PAYLOAD_BYTES", -1)

	base := scenario.MatrixScenarioFilename(latency, agents, payload)
	scenarioPath := filepath.Join(matrixDir, base+".yaml")
	cellTSV := filepath.Join(rawDir, base+".tsv")

	return matrixConfig{
		RootDir:      root,
		BinDir:       binDir,
		ReportsDir:   reportsDir,
		RawDir:       rawDir,
		MatrixDir:    matrixDir,
		SlimctlPath:  resolveProjectPath(root, envString("COMPARE_SLIMCTL", filepath.Join("bin", "slimctl"))),
		SlimEndpoint: envString("SLIM_ENDPOINT", "http://127.0.0.1:46357"),
		LatencyMs:    latency,
		Agents:       agents,
		PayloadBytes: payload,
		ThinkMs:      envInt64("MATRIX_THINK_MS", 20),
		Epochs:       envInt("MATRIX_EPOCHS", 5),
		MaxEpochMs:   envInt64("MATRIX_MAX_EPOCH_MS", 120000),
		Repeats:      envInt("CONSENSUS_REPEATS", 1),
		ScenarioPath: scenarioPath,
		ScenarioBase: base,
		CellTSV:      cellTSV,
		P2PRelayGRPC: envInt("P2P_RELAY_GRPC_PORT", 9600),
		P2PRelayCard: envInt("P2P_RELAY_CARD_PORT", 9601),
	}
}

func (c matrixConfig) validateBinaries() error {
	for _, path := range []string{
		filepath.Join(c.BinDir, "p2p-agent"),
		filepath.Join(c.BinDir, "p2p-runner"),
		filepath.Join(c.BinDir, "slim-agent"),
		filepath.Join(c.BinDir, "slim-runner"),
	} {
		if _, err := os.Stat(path); err != nil {
			return fmt.Errorf("missing binary %s (run task build first): %w", path, err)
		}
	}
	return nil
}
