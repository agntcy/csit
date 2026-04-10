// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tests

// This file is the Go/.NET suite wrapper. It keeps the client/server matrix local to the Go/C#
// slice while reusing the shared behavior layer and the existing Go and .NET harnesses.

import (
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = ginkgo.Describe("A2A Go and .NET interoperability", ginkgo.Ordered, ginkgo.Label("suite-go-dotnet"), func() {
	var (
		goAssets     fixtureBinaries
		dotNetAssets dotNetFixtureBinaries
	)

	runtime := newInteropSuiteRuntime()
	protocols := []transportProtocol{transportJSONRPC, transportREST}
	fixtures := []interopSuiteFixtureSpec{
		{
			label:    "go",
			protocol: transportJSONRPC,
			start: func() (*fixtureProcess, string, error) {
				return startGoFixture(goAssets, findFreePort(), transportJSONRPC)
			},
		},
		{
			label:    "dotnet",
			protocol: transportJSONRPC,
			start: func() (*fixtureProcess, string, error) {
				return startDotNetFixture(dotNetAssets, findFreePort(), transportJSONRPC)
			},
		},
		{
			label:    "go",
			protocol: transportREST,
			start: func() (*fixtureProcess, string, error) {
				return startGoFixture(goAssets, findFreePort(), transportREST)
			},
		},
		{
			label:    "dotnet",
			protocol: transportREST,
			start: func() (*fixtureProcess, string, error) {
				return startDotNetFixture(dotNetAssets, findFreePort(), transportREST)
			},
		},
	}

	dotNetPushUnsupported := newDotNetProbeHarness(
		func() dotNetFixtureBinaries { return dotNetAssets },
		dotNetProbeOptions{
			expectPushUnsupported: true,
			expectedPushErrorCode: dotNetPushUnsupportedCode,
		},
	)
	dotNetPushSupported := newDotNetProbeHarness(
		func() dotNetFixtureBinaries { return dotNetAssets },
		dotNetProbeOptions{expectPushSupported: true},
	)
	clients := []interopClientMatrixSpec{
		{label: "go", displayName: "Go", harness: goSDKHarness{}},
		{label: "dotnet", displayName: ".NET", harness: dotNetPushUnsupported},
	}
	servers := []interopServerMatrixSpec{
		newInteropServerSpec(runtime, "go", "Go", "go", true, protocols...),
		newInteropServerSpec(runtime, "dotnet", ".NET", "dotnet", false, protocols...),
	}
	clientHarnessOverrides := map[string]interopHarness{
		interopPairKey("dotnet", "go"): dotNetPushSupported,
	}

	ginkgo.BeforeAll(func() {
		var err error

		goAssets, err = buildGoFixtureBinaryOnly()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		dotNetAssets, err = buildDotNetFixtureBinaryOnly()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		err = startInteropSuiteFixtures(runtime, fixtures)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	})

	ginkgo.AfterAll(func() {
		runtime.stopFixtures(fixtures)
		removeTempDir(dotNetAssets.tempDir)
		removeTempDir(goAssets.tempDir)
	})

	registerInteropTransportMatrixWithOverrides(
		protocols,
		clients,
		servers,
		nil,
		clientHarnessOverrides,
	)
})
