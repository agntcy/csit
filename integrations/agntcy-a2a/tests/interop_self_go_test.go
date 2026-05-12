// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tests

import (
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = ginkgo.Describe("Go", ginkgo.Ordered, ginkgo.ContinueOnFailure, ginkgo.Label("suite-self-go"), func() {
	var binaries fixtureBinaries

	runtime := newInteropSuiteRuntime()
	protocols := []transportProtocol{transportJSONRPC, transportREST, transportGRPC}
	fixtures := []interopSuiteFixtureSpec{
		{label: "go", protocol: transportJSONRPC, start: func() (*fixtureProcess, string, error) {
			return startGoFixture(binaries, findFreePort(), transportJSONRPC)
		}},
		{label: "go", protocol: transportREST, start: func() (*fixtureProcess, string, error) {
			return startGoFixture(binaries, findFreePort(), transportREST)
		}},
		{label: "go", protocol: transportGRPC, start: func() (*fixtureProcess, string, error) {
			return startGoFixture(binaries, findFreePort(), transportGRPC)
		}},
	}

	clients := []interopClientMatrixSpec{
		{label: "go", displayName: "Go", harness: goSDKHarness{}},
	}
	servers := []interopServerMatrixSpec{
		newInteropServerSpec(runtime, "go", "Go", "go", true, protocols...),
	}

	ginkgo.BeforeAll(func() {
		var err error
		binaries, err = buildGoFixtureBinaryOnly()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		err = startInteropSuiteFixtures(runtime, fixtures)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	})

	ginkgo.AfterAll(func() {
		runtime.stopFixtures(fixtures)
		removeTempDir(binaries.tempDir)
	})

	registerInteropSelfTestMatrix(protocols, clients, servers)
})
