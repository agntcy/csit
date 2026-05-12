// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tests

// This file is the Python/.NET suite wrapper. It composes the existing Python v1.0 fixture and
// probe with the reusable .NET fixture/probe binaries and keeps pair-specific expectations local
// to this slice.

import (
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = ginkgo.Describe("Python+.NET", ginkgo.Ordered, ginkgo.ContinueOnFailure, ginkgo.Label("suite-python-dotnet"), func() {
	var (
		pythonAssets pythonFixtureAssets
		dotNetAssets dotNetFixtureBinaries
	)

	runtime := newInteropSuiteRuntime()
	protocols := []transportProtocol{transportJSONRPC, transportREST}
	fixtures := []interopSuiteFixtureSpec{
		{
			label:    "python",
			protocol: transportJSONRPC,
			start: func() (*fixtureProcess, string, error) {
				return startPythonFixture(pythonAssets, findFreePort(), transportJSONRPC)
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
			label:    "python",
			protocol: transportREST,
			start: func() (*fixtureProcess, string, error) {
				return startPythonFixture(pythonAssets, findFreePort(), transportREST)
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

	pythonPushUnsupported := newPythonProbeHarness(
		func() pythonFixtureAssets { return pythonAssets },
		rustProbeOptions{
			expectPushUnsupported: true,
			expectedPushErrorCode: dotNetPushUnsupportedCode,
		},
	)
	pythonPushSupported := newPythonProbeHarness(
		func() pythonFixtureAssets { return pythonAssets },
		rustProbeOptions{expectPushSupported: true},
	)
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
		{label: "python", displayName: "Python", harness: pythonPushUnsupported},
		{label: "dotnet", displayName: ".NET", harness: dotNetPushUnsupported},
	}
	servers := []interopServerMatrixSpec{
		newInteropServerSpec(runtime, "python", "Python", "python", true, protocols...),
		newInteropServerSpec(runtime, "dotnet", ".NET", "dotnet", false, protocols...),
	}
	clientHarnessOverrides := map[string]interopHarness{
		interopPairKey("python", "python"): pythonPushSupported,
		interopPairKey("dotnet", "python"): dotNetPushSupported,
	}

	ginkgo.BeforeAll(func() {
		var err error

		pythonAssets, err = buildPythonFixtureAssets()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		dotNetAssets, err = buildDotNetFixtureBinaryOnly()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		err = startInteropSuiteFixtures(runtime, fixtures)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	})

	ginkgo.AfterAll(func() {
		runtime.stopFixtures(fixtures)
		removeTempDir(dotNetAssets.tempDir)
	})

	registerInteropTransportMatrixWithOverrides(
		protocols,
		clients,
		servers,
		nil,
		clientHarnessOverrides,
	)
})
