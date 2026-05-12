// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tests

import (
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = ginkgo.Describe("Rust", ginkgo.Ordered, ginkgo.ContinueOnFailure, ginkgo.Label("suite-self-rust"), func() {
	var binaries fixtureBinaries

	runtime := newInteropSuiteRuntime()
	protocols := []transportProtocol{transportJSONRPC, transportREST, transportGRPC}
	fixtures := []interopSuiteFixtureSpec{
		{label: "rust", protocol: transportJSONRPC, start: func() (*fixtureProcess, string, error) {
			return startRustFixture(binaries, findFreePort(), transportJSONRPC)
		}},
		{label: "rust", protocol: transportREST, start: func() (*fixtureProcess, string, error) {
			return startRustFixture(binaries, findFreePort(), transportREST)
		}},
		{label: "rust", protocol: transportGRPC, start: func() (*fixtureProcess, string, error) {
			return startRustFixture(binaries, findFreePort(), transportGRPC)
		}},
	}

	clients := []interopClientMatrixSpec{
		{
			label:       "rust",
			displayName: "Rust",
			harness: newRustProbeHarness(
				func() fixtureBinaries { return binaries },
				rustProbeOptions{expectPushSupported: true},
			),
		},
	}
	servers := []interopServerMatrixSpec{
		newInteropServerSpec(runtime, "rust", "Rust", "rust", true, protocols...),
	}

	ginkgo.BeforeAll(func() {
		var err error
		binaries, err = buildRustFixtureBinaryOnly()
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
