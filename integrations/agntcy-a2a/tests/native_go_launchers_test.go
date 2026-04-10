// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tests

// This file isolates the native Go fixture build and startup path from the shared launcher
// helpers so the cross-SDK launcher layer stays focused on generic orchestration and the
// test-only native Go startup path is easy to find.

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func buildGoFixture(root string, outputPath string) error {
	buildCtx, cancel := context.WithTimeout(context.Background(), buildTimeout)
	defer cancel()

	goBuild := exec.CommandContext(buildCtx, "go", "build", "-o", outputPath, "./fixtures/go-jsonrpc-server")
	goBuild.Dir = root
	if output, err := goBuild.CombinedOutput(); err != nil {
		return fmt.Errorf("build go fixture: %w\n%s", err, string(output))
	}

	return nil
}

func buildGoFixtureBinaryOnly() (fixtureBinaries, error) {
	root := componentRoot()
	tempDir, err := os.MkdirTemp("", "agntcy-a2a-go-")
	if err != nil {
		return fixtureBinaries{}, fmt.Errorf("create temp dir: %w", err)
	}

	binaries := fixtureBinaries{
		tempDir:  tempDir,
		goServer: filepath.Join(tempDir, executableName("go-jsonrpc-server")),
	}

	if err := buildGoFixture(root, binaries.goServer); err != nil {
		_ = os.RemoveAll(tempDir)
		return fixtureBinaries{}, err
	}

	return binaries, nil
}

func startGoFixture(binaries fixtureBinaries, port int, protocol transportProtocol) (*fixtureProcess, string, error) {
	return startNativeFixture(fmt.Sprintf("go-%s-server", protocol), binaries.goServer, port, protocol)
}
