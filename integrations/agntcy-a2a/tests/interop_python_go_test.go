// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tests

// This file is the Python/Go suite wrapper. It declares the Python v1.0 fixture environment,
// the Go fixture binaries reused in this slice, and the client/server matrix for JSON-RPC and
// HTTP+JSON coverage.
// To expand Python coverage, change the matrix data here and keep the cross-SDK behavior logic in
// interop_behaviors_test.go so the shared slices remain authored once.

import (
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = ginkgo.Describe("Python+Go", ginkgo.Ordered, ginkgo.ContinueOnFailure, ginkgo.Label("suite-python-go"), func() {
	var (
		goAssets     fixtureBinaries
		pythonAssets pythonFixtureAssets
	)

	runtime := newInteropSuiteRuntime()
	protocols := []transportProtocol{transportJSONRPC, transportREST, transportGRPC}
	fixtures := []interopSuiteFixtureSpec{
		{
			label:    "go",
			protocol: transportJSONRPC,
			start: func() (*fixtureProcess, string, error) {
				return startGoFixture(goAssets, findFreePort(), transportJSONRPC)
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
			label:    "go",
			protocol: transportREST,
			start: func() (*fixtureProcess, string, error) {
				return startGoFixture(goAssets, findFreePort(), transportREST)
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
			label:    "go",
			protocol: transportGRPC,
			start: func() (*fixtureProcess, string, error) {
				return startGoFixture(goAssets, findFreePort(), transportGRPC)
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
		{label: "go", displayName: "Go", harness: goSDKHarness{}},
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
		newInteropServerSpec(runtime, "go", "Go", "go", true, protocols...),
		newInteropServerSpec(runtime, "python", "Python", "python", true, protocols...),
	}

	ginkgo.BeforeAll(func() {
		var err error

		goAssets, err = buildGoFixtureBinaryOnly()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		pythonAssets, err = buildPythonFixtureAssets()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		err = startInteropSuiteFixtures(runtime, fixtures)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	})

	ginkgo.AfterAll(func() {
		runtime.stopFixtures(fixtures)
		removeTempDir(goAssets.tempDir)
	})

	registerInteropTransportMatrix(
		protocols,
		clients,
		servers,
		nil,
	)
})
