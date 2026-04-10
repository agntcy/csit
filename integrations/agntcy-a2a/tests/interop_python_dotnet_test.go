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

var _ = ginkgo.Describe("A2A Python and .NET interoperability", ginkgo.Ordered, ginkgo.Label("suite-python-dotnet"), func() {
	var (
		pythonAssets         pythonFixtureAssets
		dotNetAssets         dotNetFixtureBinaries
		pythonJSONRPCFixture *fixtureProcess
		dotNetJSONRPCFixture *fixtureProcess
		pythonRESTFixture    *fixtureProcess
		dotNetRESTFixture    *fixtureProcess
		pythonJSONRPCURL     string
		dotNetJSONRPCURL     string
		pythonRESTURL        string
		dotNetRESTURL        string
	)

	pythonPushUnsupported := pythonProbeHarness{
		getAssets: func() pythonFixtureAssets { return pythonAssets },
		options: rustProbeOptions{
			expectPushUnsupported: true,
			expectedPushErrorCode: dotNetPushUnsupportedCode,
		},
	}
	pythonPushSupported := pythonProbeHarness{
		getAssets: func() pythonFixtureAssets { return pythonAssets },
		options: rustProbeOptions{
			expectPushSupported: true,
		},
	}
	dotNetPushUnsupported := dotNetProbeHarness{
		getBinaries: func() dotNetFixtureBinaries { return dotNetAssets },
		options: dotNetProbeOptions{
			expectPushUnsupported: true,
			expectedPushErrorCode: dotNetPushUnsupportedCode,
		},
	}
	dotNetPushSupported := dotNetProbeHarness{
		getBinaries: func() dotNetFixtureBinaries { return dotNetAssets },
		options: dotNetProbeOptions{
			expectPushSupported: true,
		},
	}
	clients := []interopClientMatrixSpec{
		{label: "python", displayName: "Python", harness: pythonPushUnsupported},
		{label: "dotnet", displayName: ".NET", harness: dotNetPushUnsupported},
	}
	servers := []interopServerMatrixSpec{
		{
			label:               "python",
			displayName:         "Python",
			serverPrefix:        "python",
			expectPushSupported: true,
			urls: map[transportProtocol]func() string{
				transportJSONRPC: func() string { return pythonJSONRPCURL },
				transportREST:    func() string { return pythonRESTURL },
			},
		},
		{
			label:               "dotnet",
			displayName:         ".NET",
			serverPrefix:        "dotnet",
			expectPushSupported: false,
			urls: map[transportProtocol]func() string{
				transportJSONRPC: func() string { return dotNetJSONRPCURL },
				transportREST:    func() string { return dotNetRESTURL },
			},
		},
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

		pythonJSONRPCFixture, pythonJSONRPCURL, err = startPythonFixture(pythonAssets, findFreePort(), transportJSONRPC)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		dotNetJSONRPCFixture, dotNetJSONRPCURL, err = startDotNetFixture(dotNetAssets, findFreePort(), transportJSONRPC)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		pythonRESTFixture, pythonRESTURL, err = startPythonFixture(pythonAssets, findFreePort(), transportREST)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		dotNetRESTFixture, dotNetRESTURL, err = startDotNetFixture(dotNetAssets, findFreePort(), transportREST)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	})

	ginkgo.AfterAll(func() {
		stopFixtureIfRunning(dotNetRESTFixture)
		stopFixtureIfRunning(pythonRESTFixture)
		stopFixtureIfRunning(dotNetJSONRPCFixture)
		stopFixtureIfRunning(pythonJSONRPCFixture)
		removeTempDir(dotNetAssets.tempDir)
		removeTempDir(pythonAssets.tempDir)
	})

	registerInteropTransportMatrixWithOverrides(
		[]transportProtocol{transportJSONRPC, transportREST},
		clients,
		servers,
		nil,
		clientHarnessOverrides,
	)
})
