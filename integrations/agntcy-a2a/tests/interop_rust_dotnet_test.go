// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tests

// This file is the Rust/.NET suite wrapper. It declares the Rust and .NET fixtures used in this
// slice, the client/server matrix, and any pair-specific harness overrides such as push-config
// expectations.
// To add a new Rust/.NET leg, change the matrix data here. If a new behavior applies to every leg,
// add it in interop_behaviors_test.go instead of introducing suite-local assertions here.

import (
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

const dotNetPushUnsupportedCode = -32003

var _ = ginkgo.Describe("A2A Rust and .NET interoperability", ginkgo.Ordered, ginkgo.Label("suite-rust-dotnet"), func() {
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

	rustAssets := func() fixtureBinaries { return binaries.rustAssets() }
	dotNetPushUnsupported := dotNetProbeHarness{
		getBinaries: func() rustDotNetFixtureBinaries { return binaries },
		options: dotNetProbeOptions{
			expectPushUnsupported: true,
			expectedPushErrorCode: dotNetPushUnsupportedCode,
		},
	}
	dotNetPushSupported := dotNetProbeHarness{
		getBinaries: func() rustDotNetFixtureBinaries { return binaries },
		options: dotNetProbeOptions{
			expectPushSupported: true,
		},
	}
	rustPushUnsupported := rustProbeHarness{
		getBinaries: rustAssets,
		options: rustProbeOptions{
			expectPushUnsupported: true,
			expectedPushErrorCode: dotNetPushUnsupportedCode,
		},
	}
	rustPushSupported := rustProbeHarness{
		getBinaries: rustAssets,
		options: rustProbeOptions{
			expectPushSupported: true,
		},
	}
	clients := []interopClientMatrixSpec{
		{label: "dotnet", displayName: ".NET", harness: dotNetPushUnsupported},
		{label: "rust", displayName: "Rust", harness: rustPushUnsupported},
	}
	servers := []interopServerMatrixSpec{
		{
			label:               "dotnet",
			displayName:         ".NET",
			serverPrefix:        "dotnet",
			expectPushSupported: false,
			urls: map[transportProtocol]func() string{
				transportJSONRPC: func() string { return dotnetJSONRPCFixtureURL },
				transportREST:    func() string { return dotnetRESTFixtureURL },
			},
		},
		{
			label:               "rust",
			displayName:         "Rust",
			serverPrefix:        "rust",
			expectPushSupported: true,
			urls: map[transportProtocol]func() string{
				transportJSONRPC: func() string { return rustJSONRPCFixtureURL },
				transportREST:    func() string { return rustRESTFixtureURL },
			},
		},
	}
	clientHarnessOverrides := map[string]interopHarness{
		interopPairKey("dotnet", "rust"): dotNetPushSupported,
		interopPairKey("rust", "rust"):   rustPushSupported,
	}

	ginkgo.BeforeAll(func() {
		var err error

		binaries, err = buildRustDotNetFixtureBinaries()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		dotnetJSONRPCFixture, dotnetJSONRPCFixtureURL, err = startDotNetFixture(binaries, findFreePort(), transportJSONRPC)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		rustJSONRPCFixture, rustJSONRPCFixtureURL, err = startRustFixture(binaries.rustAssets(), findFreePort(), transportJSONRPC)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		dotnetRESTFixture, dotnetRESTFixtureURL, err = startDotNetFixture(binaries, findFreePort(), transportREST)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		rustRESTFixture, rustRESTFixtureURL, err = startRustFixture(binaries.rustAssets(), findFreePort(), transportREST)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	})

	ginkgo.AfterAll(func() {
		stopFixtureIfRunning(rustRESTFixture)
		stopFixtureIfRunning(dotnetRESTFixture)
		stopFixtureIfRunning(rustJSONRPCFixture)
		stopFixtureIfRunning(dotnetJSONRPCFixture)
		removeTempDir(binaries.tempDir)
	})

	registerInteropTransportMatrixWithOverrides(
		[]transportProtocol{transportJSONRPC, transportREST},
		clients,
		servers,
		map[string]string{interopPairKey("rust", "rust"): "rust-rust-dotnet"},
		clientHarnessOverrides,
	)
})
