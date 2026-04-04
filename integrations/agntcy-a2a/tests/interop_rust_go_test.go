// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tests

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
	"github.com/a2aproject/a2a-go/v2/a2aclient"
	"github.com/a2aproject/a2a-go/v2/a2aclient/agentcard"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

const (
	fixtureReadyTimeout = 20 * time.Second
	probeTimeout        = 2 * time.Minute
	buildTimeout        = 3 * time.Minute
	stopTimeout         = 5 * time.Second
	requestText         = "ping"
)

type fixtureBinaries struct {
	tempDir    string
	goServer   string
	rustServer string
	rustProbe  string
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

func executableName(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}

	return name
}

func buildFixtureBinaries() (fixtureBinaries, error) {
	root := componentRoot()
	tempDir, err := os.MkdirTemp("", "agntcy-a2a-binaries-")
	if err != nil {
		return fixtureBinaries{}, fmt.Errorf("create temp dir: %w", err)
	}

	binaries := fixtureBinaries{
		tempDir:    tempDir,
		goServer:   filepath.Join(tempDir, executableName("go-jsonrpc-server")),
		rustServer: filepath.Join(tempDir, "cargo-target", "debug", executableName("interop-rust-server")),
		rustProbe:  filepath.Join(tempDir, "cargo-target", "debug", executableName("interop-rust-probe")),
	}

	buildCtx, cancel := context.WithTimeout(context.Background(), buildTimeout)
	defer cancel()

	goBuild := exec.CommandContext(buildCtx, "go", "build", "-o", binaries.goServer, "./fixtures/go-jsonrpc-server")
	goBuild.Dir = root
	if output, err := goBuild.CombinedOutput(); err != nil {
		_ = os.RemoveAll(tempDir)
		return fixtureBinaries{}, fmt.Errorf("build go fixture: %w\n%s", err, string(output))
	}

	rustBuild := exec.CommandContext(
		buildCtx,
		"cargo",
		"build",
		"--manifest-path",
		filepath.Join(root, "fixtures", "rust", "Cargo.toml"),
		"--bins",
		"--target-dir",
		filepath.Join(tempDir, "cargo-target"),
	)
	rustBuild.Dir = root
	if output, err := rustBuild.CombinedOutput(); err != nil {
		_ = os.RemoveAll(tempDir)
		return fixtureBinaries{}, fmt.Errorf("build rust fixtures: %w\n%s", err, string(output))
	}

	return binaries, nil
}

func startFixtureProcess(name string, dir string, readyURL string, command string, args ...string) (*fixtureProcess, error) {
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = dir

	logs := &lockedBuffer{}
	cmd.Stdout = logs
	cmd.Stderr = logs

	if err := cmd.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("start %s: %w", name, err)
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	if err := waitForReady(readyURL, done, logs); err != nil {
		cancel()
		<-done
		return nil, fmt.Errorf("wait for %s readiness: %w", name, err)
	}

	return &fixtureProcess{name: name, cmd: cmd, cancel: cancel, done: done, logs: logs}, nil
}

func startGoFixture(binaries fixtureBinaries, port int) (*fixtureProcess, string, error) {
	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	process, err := startFixtureProcess(
		"go-jsonrpc-server",
		componentRoot(),
		baseURL+"/.well-known/agent-card.json",
		binaries.goServer,
		"--port",
		fmt.Sprintf("%d", port),
	)
	return process, baseURL, err
}

func startRustFixture(binaries fixtureBinaries, port int) (*fixtureProcess, string, error) {
	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	process, err := startFixtureProcess(
		"interop-rust-server",
		componentRoot(),
		baseURL+"/.well-known/agent-card.json",
		binaries.rustServer,
		"--port",
		fmt.Sprintf("%d", port),
	)
	return process, baseURL, err
}

func newGoClient(ctx context.Context, baseURL string) (*a2aclient.Client, error) {
	card, err := agentcard.DefaultResolver.Resolve(ctx, baseURL)
	if err != nil {
		return nil, err
	}

	return a2aclient.NewFromCard(ctx, card)
}

func firstMessageText(message *a2a.Message) (string, error) {
	for _, part := range message.Parts {
		if text := part.Text(); text != "" {
			return text, nil
		}
	}

	return "", errors.New("message did not include a text part")
}

func goClientUnaryText(ctx context.Context, client *a2aclient.Client) (string, error) {
	result, err := client.SendMessage(ctx, &a2a.SendMessageRequest{
		Message: a2a.NewMessage(a2a.MessageRoleUser, a2a.NewTextPart(requestText)),
	})
	if err != nil {
		return "", err
	}

	message, ok := result.(*a2a.Message)
	if !ok {
		return "", fmt.Errorf("unexpected unary response type %T", result)
	}

	return firstMessageText(message)
}

func goClientStreamingText(ctx context.Context, client *a2aclient.Client) (string, error) {
	request := &a2a.SendMessageRequest{
		Message: a2a.NewMessage(a2a.MessageRoleUser, a2a.NewTextPart(requestText)),
	}

	for event, err := range client.SendStreamingMessage(ctx, request) {
		if err != nil {
			return "", err
		}

		message, ok := event.(*a2a.Message)
		if !ok {
			continue
		}

		return firstMessageText(message)
	}

	return "", errors.New("stream completed without a message event")
}

func runRustProbe(ctx context.Context, binaries fixtureBinaries, baseURL string, expectedText string) (string, error) {
	cmd := exec.CommandContext(
		ctx,
		binaries.rustProbe,
		"--card-url",
		baseURL,
		"--expect-text",
		expectedText,
	)
	cmd.Dir = componentRoot()

	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("rust probe failed: %w\n%s", err, string(output))
	}

	return string(output), nil
}

var _ = ginkgo.Describe("A2A Rust and Go interoperability", ginkgo.Ordered, func() {
	var (
		binaries       fixtureBinaries
		goFixture      *fixtureProcess
		rustFixture    *fixtureProcess
		goFixtureURL   string
		rustFixtureURL string
	)

	ginkgo.BeforeAll(func() {
		var err error

		binaries, err = buildFixtureBinaries()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		goFixture, goFixtureURL, err = startGoFixture(binaries, findFreePort())
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		rustFixture, rustFixtureURL, err = startRustFixture(binaries, findFreePort())
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	})

	ginkgo.AfterAll(func() {
		if rustFixture != nil {
			gomega.Expect(rustFixture.stop()).To(gomega.Succeed(), rustFixture.logs.String())
		}
		if goFixture != nil {
			gomega.Expect(goFixture.stop()).To(gomega.Succeed(), goFixture.logs.String())
		}
		if binaries.tempDir != "" {
			gomega.Expect(os.RemoveAll(binaries.tempDir)).To(gomega.Succeed())
		}
	})

	ginkgo.It("lets the Go client call the Go fixture", func(ctx ginkgo.SpecContext) {
		requestCtx, cancel := context.WithTimeout(ctx, probeTimeout)
		defer cancel()

		client, err := newGoClient(requestCtx, goFixtureURL)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		unaryText, err := goClientUnaryText(requestCtx, client)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(unaryText).To(gomega.Equal("go server received: ping"))

		streamText, err := goClientStreamingText(requestCtx, client)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(streamText).To(gomega.Equal("go server received: ping"))
	})

	ginkgo.It("lets the Go client call the Rust fixture", func(ctx ginkgo.SpecContext) {
		requestCtx, cancel := context.WithTimeout(ctx, probeTimeout)
		defer cancel()

		client, err := newGoClient(requestCtx, rustFixtureURL)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		unaryText, err := goClientUnaryText(requestCtx, client)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(unaryText).To(gomega.Equal("rust server received: ping"))

		streamText, err := goClientStreamingText(requestCtx, client)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(streamText).To(gomega.Equal("rust server received: ping"))
	})

	ginkgo.It("lets the Rust client call the Go fixture", func(ctx ginkgo.SpecContext) {
		requestCtx, cancel := context.WithTimeout(ctx, probeTimeout)
		defer cancel()

		output, err := runRustProbe(requestCtx, binaries, goFixtureURL, "go server received: ping")
		gomega.Expect(err).NotTo(gomega.HaveOccurred(), output)
	})

	ginkgo.It("lets the Rust client call the Rust fixture", func(ctx ginkgo.SpecContext) {
		requestCtx, cancel := context.WithTimeout(ctx, probeTimeout)
		defer cancel()

		output, err := runRustProbe(requestCtx, binaries, rustFixtureURL, "rust server received: ping")
		gomega.Expect(err).NotTo(gomega.HaveOccurred(), output)
	})
})
