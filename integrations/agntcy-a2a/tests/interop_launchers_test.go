// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tests

// This file launches fixture server processes for all four SDKs (Go, Rust, Python,
// .NET). All fixtures use their toolchain's run command (go run / cargo run /
// uv run / dotnet run) so no explicit build step is required.

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func resolveUvCommand() (string, error) {
	if configured := os.Getenv("UV"); configured != "" {
		return configured, nil
	}

	path, err := exec.LookPath("uv")
	if err != nil {
		return "", errors.New("uv not found; install uv (https://docs.astral.sh/uv/) or set UV to its path")
	}

	return path, nil
}

func resolveDotNetCommand() (string, error) {
	if configured := os.Getenv("DOTNET"); configured != "" {
		return configured, nil
	}

	if root := os.Getenv("DOTNET_ROOT"); root != "" {
		candidate := filepath.Join(root, executableName("dotnet"))
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}
	}

	if path, err := exec.LookPath("dotnet"); err == nil {
		return path, nil
	}

	for _, candidate := range []string{
		"/usr/local/share/dotnet/dotnet",
		"/opt/homebrew/share/dotnet/dotnet",
		filepath.Join(os.Getenv("HOME"), ".dotnet", "dotnet"),
	} {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}
	}

	return "", errors.New("dotnet executable not found; install the .NET 8 SDK or set DOTNET to the dotnet CLI path")
}

func startFixtureProcess(name string, dir string, readyURL string, command string, args ...string) (*fixtureProcess, error) {
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = dir
	setProcessGroup(cmd)

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

func protocolFixtureArgs(port int, protocol transportProtocol) ([]string, string) {
	args := []string{
		"--port",
		fmt.Sprintf("%d", port),
		"--protocol",
		string(protocol),
	}
	grpcAddress := ""
	if protocol == transportGRPC {
		grpcPort := findFreePort()
		grpcAddress = fmt.Sprintf("127.0.0.1:%d", grpcPort)
		args = append(args, "--grpc-port", fmt.Sprintf("%d", grpcPort))
	}

	return args, grpcAddress
}

func startNativeFixture(name string, port int, protocol transportProtocol, command string, commandArgs ...string) (*fixtureProcess, string, error) {
	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	args, grpcAddress := protocolFixtureArgs(port, protocol)
	process, err := startFixtureProcess(
		name,
		componentRoot(),
		baseURL+"/.well-known/agent-card.json",
		command,
		append(commandArgs, args...)...,
	)
	if err != nil {
		return nil, "", err
	}
	if grpcAddress != "" {
		if err := waitForTCPListener(grpcAddress, process.logs); err != nil {
			_ = process.stop()
			return nil, "", fmt.Errorf("wait for %s listener: %w", name, err)
		}
	}

	return process, baseURL, nil
}

func startGoFixture(protocol transportProtocol, pushEnabled bool) (*fixtureProcess, string, error) {
	args := []string{"run", "./fixtures/go-jsonrpc-server"}
	if !pushEnabled {
		args = append(args, "--disable-push")
	}
	return startNativeFixture(
		fmt.Sprintf("go-%s-server", protocol),
		findFreePort(),
		protocol,
		"go", args...,
	)
}

func startRustFixture(protocol transportProtocol, pushEnabled bool) (*fixtureProcess, string, error) {
	args := []string{"run", "--manifest-path", "fixtures/rust/Cargo.toml", "--bin", "interop-rust-server", "--"}
	if !pushEnabled {
		args = append(args, "--disable-push")
	}
	return startNativeFixture(
		fmt.Sprintf("rust-%s-server", protocol),
		findFreePort(),
		protocol,
		"cargo", args...,
	)
}

func startDotNetFixture(protocol transportProtocol, pushEnabled bool) (*fixtureProcess, string, error) {
	if pushEnabled {
		return nil, "", ErrUnsupportedConfig
	}
	dotnetCmd, err := resolveDotNetCommand()
	if err != nil {
		return nil, "", err
	}
	return startNativeFixture(
		fmt.Sprintf("dotnet-%s-server", protocol),
		findFreePort(),
		protocol,
		dotnetCmd, "run",
		"--project", "./fixtures/dotnet/InteropServer",
		"--",
	)
}

func startPythonFixture(protocol transportProtocol, pushEnabled bool) (*fixtureProcess, string, error) {
	uvCmd, err := resolveUvCommand()
	if err != nil {
		return nil, "", err
	}

	port := findFreePort()
	fixtureDir := filepath.Join(componentRoot(), "fixtures", "python")
	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	args, grpcAddress := protocolFixtureArgs(port, protocol)
	if !pushEnabled {
		args = append(args, "--disable-push")
	}

	process, err := startFixtureProcess(
		fmt.Sprintf("python-%s-server", protocol),
		fixtureDir,
		baseURL+"/.well-known/agent-card.json",
		uvCmd,
		append([]string{"run", "interop_python_server.py"}, args...)...,
	)
	if err != nil {
		return nil, "", err
	}
	if grpcAddress != "" {
		if err := waitForTCPListener(grpcAddress, process.logs); err != nil {
			_ = process.stop()
			return nil, "", fmt.Errorf("wait for python-%s-server listener: %w", protocol, err)
		}
	}

	return process, baseURL, nil
}
