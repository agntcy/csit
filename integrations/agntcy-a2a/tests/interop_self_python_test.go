// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tests

import (
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = ginkgo.Describe("Python", ginkgo.Ordered, ginkgo.ContinueOnFailure, ginkgo.Label("suite-self-python"), func() {
	var pythonAssets pythonFixtureAssets

	runtime := newInteropSuiteRuntime()
	protocols := []transportProtocol{transportJSONRPC, transportREST, transportGRPC}
	fixtures := []interopSuiteFixtureSpec{
		{label: "python", protocol: transportJSONRPC, start: func() (*fixtureProcess, string, error) {
			return startPythonFixture(pythonAssets, findFreePort(), transportJSONRPC)
		}},
		{label: "python", protocol: transportREST, start: func() (*fixtureProcess, string, error) {
			return startPythonFixture(pythonAssets, findFreePort(), transportREST)
		}},
		{label: "python", protocol: transportGRPC, start: func() (*fixtureProcess, string, error) {
			return startPythonFixture(pythonAssets, findFreePort(), transportGRPC)
		}},
	}

	clients := []interopClientMatrixSpec{
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
		newInteropServerSpec(runtime, "python", "Python", "python", true, protocols...),
	}

	ginkgo.BeforeAll(func() {
		var err error
		pythonAssets, err = buildPythonFixtureAssets()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		err = startInteropSuiteFixtures(runtime, fixtures)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	})

	ginkgo.AfterAll(func() {
		runtime.stopFixtures(fixtures)
	})

	registerInteropSelfTestMatrix(protocols, clients, servers)
})
