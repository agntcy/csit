// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tests

// This file is the Rust/Go suite wrapper. It owns only the fixture lifecycle and the client/server
// matrix for this suite, then delegates spec generation to the shared behavior layer.
// To add a new Rust/Go leg, update the clients or servers declared here and let the shared matrix
// registration create the specs. Do not add one-off It blocks unless the suite truly needs a
// behavior that cannot be expressed through the shared harness model.

import (
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = ginkgo.Describe("Rust+Go", ginkgo.Ordered, ginkgo.ContinueOnFailure, ginkgo.Label("suite-rust-go"), func() {
	var binaries fixtureBinaries

	runtime := newInteropSuiteRuntime()
	protocols := []transportProtocol{transportJSONRPC, transportREST, transportGRPC}
	fixtures := []interopSuiteFixtureSpec{
		{
			label:    "go",
			protocol: transportJSONRPC,
			start: func() (*fixtureProcess, string, error) {
				return startGoFixture(binaries, findFreePort(), transportJSONRPC)
			},
		},
		{
			label:    "rust",
			protocol: transportJSONRPC,
			start: func() (*fixtureProcess, string, error) {
				return startRustFixture(binaries, findFreePort(), transportJSONRPC)
			},
		},
		{
			label:    "go",
			protocol: transportREST,
			start: func() (*fixtureProcess, string, error) {
				return startGoFixture(binaries, findFreePort(), transportREST)
			},
		},
		{
			label:    "rust",
			protocol: transportREST,
			start: func() (*fixtureProcess, string, error) {
				return startRustFixture(binaries, findFreePort(), transportREST)
			},
		},
		{
			label:    "go",
			protocol: transportGRPC,
			start: func() (*fixtureProcess, string, error) {
				return startGoFixture(binaries, findFreePort(), transportGRPC)
			},
		},
		{
			label:    "rust",
			protocol: transportGRPC,
			start: func() (*fixtureProcess, string, error) {
				return startRustFixture(binaries, findFreePort(), transportGRPC)
			},
		},
	}

	clients := []interopClientMatrixSpec{
		{label: "go", displayName: "Go", harness: goSDKHarness{}},
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
		newInteropServerSpec(runtime, "go", "Go", "go", true, protocols...),
		newInteropServerSpec(runtime, "rust", "Rust", "rust", true, protocols...),
	}

	ginkgo.BeforeAll(func() {
		var err error

		binaries, err = buildFixtureBinaries()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		err = startInteropSuiteFixtures(runtime, fixtures)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	})

	ginkgo.AfterAll(func() {
		runtime.stopFixtures(fixtures)
		removeTempDir(binaries.tempDir)
	})

	registerInteropTransportMatrix(
		protocols,
		clients,
		servers,
		nil,
	)
})
