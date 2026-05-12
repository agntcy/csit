// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tests

// This file is the Rust/Python suite wrapper. It keeps the matrix declaration,
// fixture lifecycle, and transport coverage local to this slice while reusing
// the shared Go-authored behavior assertions.

import (
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = ginkgo.Describe("Rust+Python", ginkgo.Ordered, ginkgo.ContinueOnFailure, ginkgo.Label("suite-rust-python"), func() {
	var (
		rustAssets   fixtureBinaries
		pythonAssets pythonFixtureAssets
	)

	runtime := newInteropSuiteRuntime()
	protocols := []transportProtocol{transportJSONRPC, transportREST, transportGRPC}
	fixtures := []interopSuiteFixtureSpec{
		{
			label:    "rust",
			protocol: transportJSONRPC,
			start: func() (*fixtureProcess, string, error) {
				return startRustFixture(rustAssets, findFreePort(), transportJSONRPC)
			},
		},
		{
			label:    "python",
			protocol: transportJSONRPC,
			start: func() (*fixtureProcess, string, error) {
				return startPythonFixture(pythonAssets, findFreePort(), transportJSONRPC)
			},
		},
		{
			label:    "rust",
			protocol: transportREST,
			start: func() (*fixtureProcess, string, error) {
				return startRustFixture(rustAssets, findFreePort(), transportREST)
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
			label:    "rust",
			protocol: transportGRPC,
			start: func() (*fixtureProcess, string, error) {
				return startRustFixture(rustAssets, findFreePort(), transportGRPC)
			},
		},
		{
			label:    "python",
			protocol: transportGRPC,
			start: func() (*fixtureProcess, string, error) {
				return startPythonFixture(pythonAssets, findFreePort(), transportGRPC)
			},
		},
	}

	clients := []interopClientMatrixSpec{
		{
			label:       "rust",
			displayName: "Rust",
			harness: newRustProbeHarness(
				func() fixtureBinaries { return rustAssets },
				rustProbeOptions{expectPushSupported: true},
			),
		},
		{
			label:       "python",
			displayName: "Python",
			harness: newPythonProbeHarness(
				func() pythonFixtureAssets { return pythonAssets },
				rustProbeOptions{expectPushSupported: true},
			),
		},
	}
	servers := []interopServerMatrixSpec{
		newInteropServerSpec(runtime, "rust", "Rust", "rust", true, protocols...),
		newInteropServerSpec(runtime, "python", "Python", "python", true, protocols...),
	}

	ginkgo.BeforeAll(func() {
		var err error

		rustAssets, err = buildRustFixtureBinaryOnly()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		pythonAssets, err = buildPythonFixtureAssets()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		err = startInteropSuiteFixtures(runtime, fixtures)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	})

	ginkgo.AfterAll(func() {
		runtime.stopFixtures(fixtures)
		removeTempDir(rustAssets.tempDir)
	})

	registerInteropTransportMatrix(
		protocols,
		clients,
		servers,
		nil,
	)
})
