// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tests

// This file holds the reusable interop test primitives: shared constants, transport and
// scenario enums, fixture process types, and generic lifecycle helpers.
// Add code here only when multiple suites or harnesses need the same low-level plumbing.
// New behavior assertions belong in interop_behaviors_test.go, and suite-specific matrix
// wiring belongs in the suite wrapper files.

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/onsi/gomega"
)

const (
	fixtureReadyTimeout          = 20 * time.Second
	probeTimeout                 = 2 * time.Minute
	buildTimeout                 = 3 * time.Minute
	stopTimeout                  = 5 * time.Second
	requestText                  = "ping"
	pendingRequestText           = "pending"
	messageOnlyRequestText       = "message-only"
	taskFailureRequestText       = "task-failure"
	multiTurnStartRequestText    = "multi-turn start"
	multiTurnContinueRequestText = "multi-turn continue"
	streamingRequestText         = "streaming"
	longRunningRequestText       = "long-running"
	dataTypesRequestText         = "data-types"
	requestDataKind              = "structured"
	requestDataScope             = "interop"
	requestMetadataKey           = "csit"
	requestMetadataValue         = "multipart"
)

var (
	extendedCardSchemeID = a2a.SecuritySchemeName("bearer_token")
	expectedSkillIDs     = []string{"message-only", "task-lifecycle", "task-failure", "task-cancel", "multi-turn", "streaming", "long-running", "data-types"}
)

type transportProtocol string

const (
	transportJSONRPC transportProtocol = "jsonrpc"
	transportREST    transportProtocol = "rest"
	transportGRPC    transportProtocol = "grpc"
)

type probeScenario string

const (
	probeScenarioCore           probeScenario = "core"
	probeScenarioUnaryStreaming probeScenario = "unary-streaming"
	probeScenarioTaskLifecycle  probeScenario = "task-lifecycle"
	probeScenarioPushConfig     probeScenario = "push-config"
	probeScenarioParity         probeScenario = "parity"
)

type rustProbeOptions struct {
	scenario                   probeScenario
	expectSubscribeUnsupported bool
	expectPushSupported        bool
	expectPushUnsupported      bool
	relaxedErrorChecks         bool
	expectedPushErrorCode      int
}

type dotNetProbeOptions = rustProbeOptions

type fixtureBinaries struct {
	tempDir    string
	goServer   string
	rustServer string
	rustProbe  string
}

type rustDotNetFixtureBinaries struct {
	tempDir        string
	rustServer     string
	rustProbe      string
	dotnetCommand  string
	dotnetServerDL string
	dotnetProbeDL  string
}

func (binaries rustDotNetFixtureBinaries) rustAssets() fixtureBinaries {
	return fixtureBinaries{
		tempDir:    binaries.tempDir,
		rustServer: binaries.rustServer,
		rustProbe:  binaries.rustProbe,
	}
}

type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (buffer *lockedBuffer) Write(data []byte) (int, error) {
	buffer.mu.Lock()
	defer buffer.mu.Unlock()
	return buffer.buf.Write(data)
}

func (buffer *lockedBuffer) String() string {
	buffer.mu.Lock()
	defer buffer.mu.Unlock()
	return buffer.buf.String()
}

type fixtureProcess struct {
	name   string
	cmd    *exec.Cmd
	cancel context.CancelFunc
	done   chan error
	logs   *lockedBuffer
}

func (process *fixtureProcess) stop() error {
	process.cancel()

	select {
	case err := <-process.done:
		return normalizeStopError(err)
	case <-time.After(stopTimeout):
	}

	if process.cmd.Process != nil {
		_ = process.cmd.Process.Kill()
	}

	select {
	case err := <-process.done:
		return normalizeStopError(err)
	case <-time.After(stopTimeout):
		return fmt.Errorf("timed out stopping %s", process.name)
	}
}

func normalizeStopError(err error) error {
	if err == nil {
		return nil
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return nil
	}

	return err
}

func componentRoot() string {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		panic("failed to determine test file path")
	}

	return filepath.Dir(filepath.Dir(currentFile))
}

func findFreePort() int {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	defer listener.Close()

	return listener.Addr().(*net.TCPAddr).Port
}

func waitForReady(url string, done <-chan error, logs *lockedBuffer) error {
	client := &http.Client{Timeout: 500 * time.Millisecond}
	deadline := time.Now().Add(fixtureReadyTimeout)

	for time.Now().Before(deadline) {
		select {
		case err := <-done:
			if err == nil {
				err = errors.New("process exited before becoming ready")
			}
			return fmt.Errorf("fixture exited early while waiting for %s: %w\n%s", url, err, logs.String())
		default:
		}

		response, err := client.Get(url)
		if err == nil {
			response.Body.Close()
			if response.StatusCode == http.StatusOK {
				return nil
			}
		}

		time.Sleep(200 * time.Millisecond)
	}

	return fmt.Errorf("timed out waiting for fixture readiness at %s\n%s", url, logs.String())
}

func waitForTCPListener(address string, logs *lockedBuffer) error {
	deadline := time.Now().Add(fixtureReadyTimeout)

	for time.Now().Before(deadline) {
		connection, err := net.DialTimeout("tcp", address, 500*time.Millisecond)
		if err == nil {
			connection.Close()
			return nil
		}

		time.Sleep(200 * time.Millisecond)
	}

	return fmt.Errorf("timed out waiting for fixture listener at %s\n%s", address, logs.String())
}

func executableName(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}

	return name
}

func stopFixtureIfRunning(process *fixtureProcess) {
	if process == nil {
		return
	}

	gomega.Expect(process.stop()).To(gomega.Succeed(), process.logs.String())
}

func removeTempDir(path string) {
	if path == "" {
		return
	}

	gomega.Expect(os.RemoveAll(path)).To(gomega.Succeed())
}

func expectedServerText(serverPrefix string, text string) string {
	return fmt.Sprintf("%s server received: %s", serverPrefix, text)
}

func expectedCancelText(serverPrefix string) string {
	return fmt.Sprintf("%s server canceled task", serverPrefix)
}
