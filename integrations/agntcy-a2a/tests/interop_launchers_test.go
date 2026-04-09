// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tests

// This file builds fixture binaries, launches per-transport test servers, and runs the Rust
// and .NET probe executables used by the shared Ginkgo behaviors.
// Extend this file when a new SDK fixture, probe binary, or transport startup path is needed.
// Keep behavior checks out of this layer so the wrappers and shared behavior registry stay
// declarative.

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

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

func buildRustDotNetFixtureBinaries() (rustDotNetFixtureBinaries, error) {
	root := componentRoot()
	dotnetCommand, err := resolveDotNetCommand()
	if err != nil {
		return rustDotNetFixtureBinaries{}, err
	}

	tempDir, err := os.MkdirTemp("", "agntcy-a2a-rust-dotnet-")
	if err != nil {
		return rustDotNetFixtureBinaries{}, fmt.Errorf("create temp dir: %w", err)
	}

	binaries := rustDotNetFixtureBinaries{
		tempDir:        tempDir,
		rustServer:     filepath.Join(tempDir, "cargo-target", "debug", executableName("interop-rust-server")),
		rustProbe:      filepath.Join(tempDir, "cargo-target", "debug", executableName("interop-rust-probe")),
		dotnetCommand:  dotnetCommand,
		dotnetServerDL: filepath.Join(tempDir, "dotnet-server", "InteropServer.dll"),
		dotnetProbeDL:  filepath.Join(tempDir, "dotnet-probe", "InteropProbe.dll"),
	}

	buildCtx, cancel := context.WithTimeout(context.Background(), buildTimeout)
	defer cancel()

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
		return rustDotNetFixtureBinaries{}, fmt.Errorf("build rust fixtures: %w\n%s", err, string(output))
	}

	dotnetServerBuild := exec.CommandContext(
		buildCtx,
		binaries.dotnetCommand,
		"publish",
		"./fixtures/dotnet/InteropServer/InteropServer.csproj",
		"-c",
		"Release",
		"-o",
		filepath.Join(tempDir, "dotnet-server"),
		"-p:UseAppHost=false",
	)
	dotnetServerBuild.Dir = root
	if output, err := dotnetServerBuild.CombinedOutput(); err != nil {
		_ = os.RemoveAll(tempDir)
		return rustDotNetFixtureBinaries{}, fmt.Errorf("build dotnet server fixture: %w\n%s", err, string(output))
	}

	dotnetProbeBuild := exec.CommandContext(
		buildCtx,
		binaries.dotnetCommand,
		"publish",
		"./fixtures/dotnet/InteropProbe/InteropProbe.csproj",
		"-c",
		"Release",
		"-o",
		filepath.Join(tempDir, "dotnet-probe"),
		"-p:UseAppHost=false",
	)
	dotnetProbeBuild.Dir = root
	if output, err := dotnetProbeBuild.CombinedOutput(); err != nil {
		_ = os.RemoveAll(tempDir)
		return rustDotNetFixtureBinaries{}, fmt.Errorf("build dotnet probe: %w\n%s", err, string(output))
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

func startNativeFixture(name string, command string, port int, protocol transportProtocol) (*fixtureProcess, string, error) {
	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	args, grpcAddress := protocolFixtureArgs(port, protocol)
	process, err := startFixtureProcess(
		name,
		componentRoot(),
		baseURL+"/.well-known/agent-card.json",
		command,
		args...,
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

func startGoFixture(binaries fixtureBinaries, port int, protocol transportProtocol) (*fixtureProcess, string, error) {
	return startNativeFixture(fmt.Sprintf("go-%s-server", protocol), binaries.goServer, port, protocol)
}

func startRustFixture(binaries fixtureBinaries, port int, protocol transportProtocol) (*fixtureProcess, string, error) {
	return startNativeFixture(fmt.Sprintf("rust-%s-server", protocol), binaries.rustServer, port, protocol)
}

func startDotNetFixture(binaries rustDotNetFixtureBinaries, port int, protocol transportProtocol) (*fixtureProcess, string, error) {
	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	args, _ := protocolFixtureArgs(port, protocol)
	process, err := startFixtureProcess(
		fmt.Sprintf("dotnet-%s-server", protocol),
		componentRoot(),
		baseURL+"/.well-known/agent-card.json",
		binaries.dotnetCommand,
		append([]string{binaries.dotnetServerDL}, args...)...,
	)
	if err != nil {
		return nil, "", err
	}

	return process, baseURL, nil
}

func appendProbeOptions(args []string, options rustProbeOptions) []string {
	if options.scenario != "" {
		args = append(args, "--scenario", string(options.scenario))
	}
	if options.expectSubscribeUnsupported {
		args = append(args, "--expect-subscribe-unsupported")
	}
	if options.expectPushSupported {
		args = append(args, "--expect-push-supported")
	}
	if options.expectPushUnsupported {
		args = append(args, "--expect-push-unsupported")
		if options.expectedPushErrorCode != 0 {
			args = append(args, "--expected-push-error-code", fmt.Sprintf("%d", options.expectedPushErrorCode))
		}
	}
	if options.relaxedErrorChecks {
		args = append(args, "--relaxed-error-checks")
	}

	return args
}

func runRustProbe(
	ctx context.Context,
	binaries fixtureBinaries,
	baseURL string,
	serverPrefix string,
	options rustProbeOptions,
) (string, error) {
	args := appendProbeOptions([]string{
		"--card-url",
		baseURL,
		"--server-prefix",
		serverPrefix,
	}, options)

	cmd := exec.CommandContext(ctx, binaries.rustProbe, args...)
	cmd.Dir = componentRoot()

	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("rust probe failed: %w\n%s", err, string(output))
	}

	return string(output), nil
}

func runDotNetProbe(
	ctx context.Context,
	binaries rustDotNetFixtureBinaries,
	baseURL string,
	serverPrefix string,
	options dotNetProbeOptions,
) (string, error) {
	args := append([]string{binaries.dotnetProbeDL}, appendProbeOptions([]string{
		"--card-url",
		baseURL,
		"--server-prefix",
		serverPrefix,
	}, options)...)

	cmd := exec.CommandContext(ctx, binaries.dotnetCommand, args...)
	cmd.Dir = componentRoot()

	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("dotnet probe failed: %w\n%s", err, string(output))
	}

	return string(output), nil
}
