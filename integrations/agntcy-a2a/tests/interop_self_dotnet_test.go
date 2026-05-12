// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tests

import (
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = ginkgo.Describe(".NET", ginkgo.Ordered, ginkgo.ContinueOnFailure, ginkgo.Label("suite-self-dotnet"), func() {
	var dotNetAssets dotNetFixtureBinaries

	runtime := newInteropSuiteRuntime()
	protocols := []transportProtocol{transportJSONRPC, transportREST}
	fixtures := []interopSuiteFixtureSpec{
		{label: "dotnet", protocol: transportJSONRPC, start: func() (*fixtureProcess, string, error) {
			return startDotNetFixture(dotNetAssets, findFreePort(), transportJSONRPC)
		}},
		{label: "dotnet", protocol: transportREST, start: func() (*fixtureProcess, string, error) {
			return startDotNetFixture(dotNetAssets, findFreePort(), transportREST)
		}},
	}

	clients := []interopClientMatrixSpec{
		{
			label:       "dotnet",
			displayName: ".NET",
			harness: newDotNetProbeHarness(
				func() dotNetFixtureBinaries { return dotNetAssets },
				dotNetProbeOptions{
					expectPushUnsupported: true,
					expectedPushErrorCode: dotNetPushUnsupportedCode,
				},
			),
		},
	}
	servers := []interopServerMatrixSpec{
		newInteropServerSpec(runtime, "dotnet", ".NET", "dotnet", false, protocols...),
	}

	ginkgo.BeforeAll(func() {
		var err error
		dotNetAssets, err = buildDotNetFixtureBinaryOnly()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		err = startInteropSuiteFixtures(runtime, fixtures)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	})

	ginkgo.AfterAll(func() {
		runtime.stopFixtures(fixtures)
		removeTempDir(dotNetAssets.tempDir)
	})

	registerInteropSelfTestMatrix(protocols, clients, servers)
})
