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
		goAssets             fixtureBinaries
		dotNetAssets         dotNetFixtureBinaries
		goJSONRPCFixture     *fixtureProcess
		dotNetJSONRPCFixture *fixtureProcess
		goRESTFixture        *fixtureProcess
		dotNetRESTFixture    *fixtureProcess
		goJSONRPCFixtureURL  string
		dotNetJSONRPCURL     string
		goRESTFixtureURL     string
		dotNetRESTURL        string
	)

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
		{label: "go", displayName: "Go", harness: goSDKHarness{}},
		{label: "dotnet", displayName: ".NET", harness: dotNetPushUnsupported},
	}
	servers := []interopServerMatrixSpec{
		{
			label:               "go",
			displayName:         "Go",
			serverPrefix:        "go",
			expectPushSupported: true,
			urls: map[transportProtocol]func() string{
				transportJSONRPC: func() string { return goJSONRPCFixtureURL },
				transportREST:    func() string { return goRESTFixtureURL },
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
		interopPairKey("dotnet", "go"): dotNetPushSupported,
	}

	ginkgo.BeforeAll(func() {
		var err error

		goAssets, err = buildGoFixtureBinaryOnly()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		dotNetAssets, err = buildDotNetFixtureBinaryOnly()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		goJSONRPCFixture, goJSONRPCFixtureURL, err = startGoFixture(goAssets, findFreePort(), transportJSONRPC)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		dotNetJSONRPCFixture, dotNetJSONRPCURL, err = startDotNetFixture(dotNetAssets, findFreePort(), transportJSONRPC)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		goRESTFixture, goRESTFixtureURL, err = startGoFixture(goAssets, findFreePort(), transportREST)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		dotNetRESTFixture, dotNetRESTURL, err = startDotNetFixture(dotNetAssets, findFreePort(), transportREST)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	})

	ginkgo.AfterAll(func() {
		stopFixtureIfRunning(dotNetRESTFixture)
		stopFixtureIfRunning(goRESTFixture)
		stopFixtureIfRunning(dotNetJSONRPCFixture)
		stopFixtureIfRunning(goJSONRPCFixture)
		removeTempDir(dotNetAssets.tempDir)
		removeTempDir(goAssets.tempDir)
	})

	registerInteropTransportMatrixWithOverrides(
		[]transportProtocol{transportJSONRPC, transportREST},
		clients,
		servers,
		nil,
		clientHarnessOverrides,
	)
})
