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
	var binaries rustDotNetFixtureBinaries

	runtime := newInteropSuiteRuntime()
	protocols := []transportProtocol{transportJSONRPC, transportREST}
	fixtures := []interopSuiteFixtureSpec{
		{
			label:    "dotnet",
			protocol: transportJSONRPC,
			start: func() (*fixtureProcess, string, error) {
				return startDotNetFixture(binaries.dotNetAssets(), findFreePort(), transportJSONRPC)
			},
		},
		{
			label:    "rust",
			protocol: transportJSONRPC,
			start: func() (*fixtureProcess, string, error) {
				return startRustFixture(binaries.rustAssets(), findFreePort(), transportJSONRPC)
			},
		},
		{
			label:    "dotnet",
			protocol: transportREST,
			start: func() (*fixtureProcess, string, error) {
				return startDotNetFixture(binaries.dotNetAssets(), findFreePort(), transportREST)
			},
		},
		{
			label:    "rust",
			protocol: transportREST,
			start: func() (*fixtureProcess, string, error) {
				return startRustFixture(binaries.rustAssets(), findFreePort(), transportREST)
			},
		},
	}

	rustAssets := func() fixtureBinaries { return binaries.rustAssets() }
	dotNetAssets := func() dotNetFixtureBinaries { return binaries.dotNetAssets() }
	dotNetPushUnsupported := newDotNetProbeHarness(
		dotNetAssets,
		dotNetProbeOptions{
			expectPushUnsupported: true,
			expectedPushErrorCode: dotNetPushUnsupportedCode,
		},
	)
	dotNetPushSupported := newDotNetProbeHarness(
		dotNetAssets,
		dotNetProbeOptions{expectPushSupported: true},
	)
	rustPushUnsupported := newRustProbeHarness(
		rustAssets,
		rustProbeOptions{
			expectPushUnsupported: true,
			expectedPushErrorCode: dotNetPushUnsupportedCode,
		},
	)
	rustPushSupported := newRustProbeHarness(
		rustAssets,
		rustProbeOptions{expectPushSupported: true},
	)
	clients := []interopClientMatrixSpec{
		{label: "dotnet", displayName: ".NET", harness: dotNetPushUnsupported},
		{label: "rust", displayName: "Rust", harness: rustPushUnsupported},
	}
	servers := []interopServerMatrixSpec{
		newInteropServerSpec(runtime, "dotnet", ".NET", "dotnet", false, protocols...),
		newInteropServerSpec(runtime, "rust", "Rust", "rust", true, protocols...),
	}
	clientHarnessOverrides := map[string]interopHarness{
		interopPairKey("dotnet", "rust"): dotNetPushSupported,
		interopPairKey("rust", "rust"):   rustPushSupported,
	}

	ginkgo.BeforeAll(func() {
		var err error

		binaries, err = buildRustDotNetFixtureBinaries()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		err = startInteropSuiteFixtures(runtime, fixtures)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	})

	ginkgo.AfterAll(func() {
		runtime.stopFixtures(fixtures)
		removeTempDir(binaries.tempDir)
	})

	registerInteropTransportMatrixWithOverrides(
		protocols,
		clients,
		servers,
		map[string]string{interopPairKey("rust", "rust"): "rust-rust-dotnet"},
		clientHarnessOverrides,
	)
})
