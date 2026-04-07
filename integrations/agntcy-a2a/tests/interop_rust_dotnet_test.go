// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tests

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

const dotNetPushUnsupportedCode = -32003

type dotNetProbeOptions = rustProbeOptions

type rustDotNetFixtureBinaries struct {
	tempDir        string
	rustServer     string
	rustProbe      string
	dotnetCommand  string
	dotnetServerDL string
	dotnetProbeDL  string
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

func startDotNetFixture(
	binaries rustDotNetFixtureBinaries,
	port int,
	protocol transportProtocol,
) (*fixtureProcess, string, error) {
	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	process, err := startFixtureProcess(
		fmt.Sprintf("dotnet-%s-server", protocol),
		componentRoot(),
		baseURL+"/.well-known/agent-card.json",
		binaries.dotnetCommand,
		binaries.dotnetServerDL,
		"--port",
		fmt.Sprintf("%d", port),
		"--protocol",
		string(protocol),
	)
	if err != nil {
		return nil, "", err
	}

	return process, baseURL, nil
}

func runDotNetProbe(
	ctx context.Context,
	binaries rustDotNetFixtureBinaries,
	baseURL string,
	serverPrefix string,
	options dotNetProbeOptions,
) (string, error) {
	args := []string{
		binaries.dotnetProbeDL,
		"--card-url",
		baseURL,
		"--server-prefix",
		serverPrefix,
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

	cmd := exec.CommandContext(ctx, binaries.dotnetCommand, args...)
	cmd.Dir = componentRoot()

	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("dotnet probe failed: %w\n%s", err, string(output))
	}

	return string(output), nil
}

var _ = ginkgo.Describe("A2A Rust and .NET interoperability", ginkgo.Ordered, func() {
	var (
		binaries                rustDotNetFixtureBinaries
		dotnetJSONRPCFixture    *fixtureProcess
		rustJSONRPCFixture      *fixtureProcess
		dotnetRESTFixture       *fixtureProcess
		rustRESTFixture         *fixtureProcess
		dotnetJSONRPCFixtureURL string
		rustJSONRPCFixtureURL   string
		dotnetRESTFixtureURL    string
		rustRESTFixtureURL      string
	)

	ginkgo.BeforeAll(func() {
		var err error

		binaries, err = buildRustDotNetFixtureBinaries()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		dotnetJSONRPCFixture, dotnetJSONRPCFixtureURL, err = startDotNetFixture(binaries, findFreePort(), transportJSONRPC)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		rustJSONRPCFixture, rustJSONRPCFixtureURL, err = startRustFixture(fixtureBinaries{
			tempDir:    binaries.tempDir,
			rustServer: binaries.rustServer,
			rustProbe:  binaries.rustProbe,
		}, findFreePort(), transportJSONRPC)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		dotnetRESTFixture, dotnetRESTFixtureURL, err = startDotNetFixture(binaries, findFreePort(), transportREST)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		rustRESTFixture, rustRESTFixtureURL, err = startRustFixture(fixtureBinaries{
			tempDir:    binaries.tempDir,
			rustServer: binaries.rustServer,
			rustProbe:  binaries.rustProbe,
		}, findFreePort(), transportREST)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	})

	ginkgo.AfterAll(func() {
		if rustRESTFixture != nil {
			gomega.Expect(rustRESTFixture.stop()).To(gomega.Succeed(), rustRESTFixture.logs.String())
		}
		if dotnetRESTFixture != nil {
			gomega.Expect(dotnetRESTFixture.stop()).To(gomega.Succeed(), dotnetRESTFixture.logs.String())
		}
		if rustJSONRPCFixture != nil {
			gomega.Expect(rustJSONRPCFixture.stop()).To(gomega.Succeed(), rustJSONRPCFixture.logs.String())
		}
		if dotnetJSONRPCFixture != nil {
			gomega.Expect(dotnetJSONRPCFixture.stop()).To(gomega.Succeed(), dotnetJSONRPCFixture.logs.String())
		}
		if binaries.tempDir != "" {
			gomega.Expect(os.RemoveAll(binaries.tempDir)).To(gomega.Succeed())
		}
	})

	ginkgo.Context("JSON-RPC transport", func() {
		ginkgo.It("lets the .NET client call the .NET fixture", ginkgo.Label("suite-rust-dotnet", "jsonrpc", "dotnet-dotnet"), func(ctx ginkgo.SpecContext) {
			requestCtx, cancel := context.WithTimeout(ctx, probeTimeout)
			defer cancel()

			output, err := runDotNetProbe(requestCtx, binaries, dotnetJSONRPCFixtureURL, "dotnet", dotNetProbeOptions{
				expectPushUnsupported: true,
				expectedPushErrorCode: dotNetPushUnsupportedCode,
			})
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), output)
		})

		ginkgo.It("lets the .NET client call the Rust fixture", ginkgo.Label("suite-rust-dotnet", "jsonrpc", "dotnet-rust"), func(ctx ginkgo.SpecContext) {
			requestCtx, cancel := context.WithTimeout(ctx, probeTimeout)
			defer cancel()

			output, err := runDotNetProbe(requestCtx, binaries, rustJSONRPCFixtureURL, "rust", dotNetProbeOptions{
				expectPushSupported: true,
			})
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), output)
		})

		ginkgo.It("lets the Rust client call the .NET fixture", ginkgo.Label("suite-rust-dotnet", "jsonrpc", "rust-dotnet"), func(ctx ginkgo.SpecContext) {
			requestCtx, cancel := context.WithTimeout(ctx, probeTimeout)
			defer cancel()

			output, err := runRustProbe(requestCtx, fixtureBinaries{
				tempDir:    binaries.tempDir,
				rustServer: binaries.rustServer,
				rustProbe:  binaries.rustProbe,
			}, dotnetJSONRPCFixtureURL, "dotnet", rustProbeOptions{
				expectPushUnsupported: true,
				expectedPushErrorCode: dotNetPushUnsupportedCode,
			})
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), output)
		})

		ginkgo.It("lets the Rust client call the Rust fixture", ginkgo.Label("suite-rust-dotnet", "jsonrpc", "rust-rust-dotnet"), func(ctx ginkgo.SpecContext) {
			requestCtx, cancel := context.WithTimeout(ctx, probeTimeout)
			defer cancel()

			output, err := runRustProbe(requestCtx, fixtureBinaries{
				tempDir:    binaries.tempDir,
				rustServer: binaries.rustServer,
				rustProbe:  binaries.rustProbe,
			}, rustJSONRPCFixtureURL, "rust", rustProbeOptions{
				expectPushSupported: true,
			})
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), output)
		})
	})

	ginkgo.Context("HTTP+JSON transport", func() {
		ginkgo.It("lets the .NET client call the .NET fixture over REST", ginkgo.Label("suite-rust-dotnet", "rest", "dotnet-dotnet"), func(ctx ginkgo.SpecContext) {
			requestCtx, cancel := context.WithTimeout(ctx, probeTimeout)
			defer cancel()

			output, err := runDotNetProbe(requestCtx, binaries, dotnetRESTFixtureURL, "dotnet", dotNetProbeOptions{
				expectPushUnsupported: true,
				expectedPushErrorCode: dotNetPushUnsupportedCode,
			})
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), output)
		})

		ginkgo.It("lets the .NET client call the Rust fixture over REST", ginkgo.Label("suite-rust-dotnet", "rest", "dotnet-rust"), func(ctx ginkgo.SpecContext) {
			requestCtx, cancel := context.WithTimeout(ctx, probeTimeout)
			defer cancel()

			output, err := runDotNetProbe(requestCtx, binaries, rustRESTFixtureURL, "rust", dotNetProbeOptions{
				expectPushSupported: true,
			})
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), output)
		})

		ginkgo.It("lets the Rust client call the .NET fixture over REST", ginkgo.Label("suite-rust-dotnet", "rest", "rust-dotnet"), func(ctx ginkgo.SpecContext) {
			requestCtx, cancel := context.WithTimeout(ctx, probeTimeout)
			defer cancel()

			output, err := runRustProbe(requestCtx, fixtureBinaries{
				tempDir:    binaries.tempDir,
				rustServer: binaries.rustServer,
				rustProbe:  binaries.rustProbe,
			}, dotnetRESTFixtureURL, "dotnet", rustProbeOptions{
				expectPushUnsupported: true,
				expectedPushErrorCode: dotNetPushUnsupportedCode,
			})
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), output)
		})

		ginkgo.It("lets the Rust client call the Rust fixture over REST", ginkgo.Label("suite-rust-dotnet", "rest", "rust-rust-dotnet"), func(ctx ginkgo.SpecContext) {
			requestCtx, cancel := context.WithTimeout(ctx, probeTimeout)
			defer cancel()

			output, err := runRustProbe(requestCtx, fixtureBinaries{
				tempDir:    binaries.tempDir,
				rustServer: binaries.rustServer,
				rustProbe:  binaries.rustProbe,
			}, rustRESTFixtureURL, "rust", rustProbeOptions{
				expectPushSupported: true,
			})
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), output)
		})
	})
})
