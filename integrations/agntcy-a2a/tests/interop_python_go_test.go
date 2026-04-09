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

var _ = ginkgo.Describe("A2A Python and Go interoperability", ginkgo.Ordered, ginkgo.Label("suite-python-go"), func() {
	var (
		goAssets             fixtureBinaries
		pythonAssets         pythonFixtureAssets
		goJSONRPCFixture     *fixtureProcess
		pythonJSONRPCFixture *fixtureProcess
		goRESTFixture        *fixtureProcess
		pythonRESTFixture    *fixtureProcess
		goJSONRPCFixtureURL  string
		pythonJSONRPCURL     string
		goRESTFixtureURL     string
		pythonRESTURL        string
	)

	goClient := goSDKHarness{}
	pythonClient := pythonProbeHarness{
		getAssets: func() pythonFixtureAssets { return pythonAssets },
		options: rustProbeOptions{
			expectPushSupported: true,
		},
	}
	clients := []interopClientMatrixSpec{
		{label: "go", displayName: "Go", harness: goClient},
		{label: "python", displayName: "Python", harness: pythonClient},
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
			label:               "python",
			displayName:         "Python",
			serverPrefix:        "python",
			expectPushSupported: true,
			urls: map[transportProtocol]func() string{
				transportJSONRPC: func() string { return pythonJSONRPCURL },
				transportREST:    func() string { return pythonRESTURL },
			},
		},
	}

	ginkgo.BeforeAll(func() {
		var err error

		goAssets, err = buildGoFixtureBinaryOnly()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		pythonAssets, err = buildPythonFixtureAssets()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		goJSONRPCFixture, goJSONRPCFixtureURL, err = startGoFixture(goAssets, findFreePort(), transportJSONRPC)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		pythonJSONRPCFixture, pythonJSONRPCURL, err = startPythonFixture(pythonAssets, findFreePort(), transportJSONRPC)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		goRESTFixture, goRESTFixtureURL, err = startGoFixture(goAssets, findFreePort(), transportREST)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		pythonRESTFixture, pythonRESTURL, err = startPythonFixture(pythonAssets, findFreePort(), transportREST)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	})

	ginkgo.AfterAll(func() {
		stopFixtureIfRunning(pythonRESTFixture)
		stopFixtureIfRunning(goRESTFixture)
		stopFixtureIfRunning(pythonJSONRPCFixture)
		stopFixtureIfRunning(goJSONRPCFixture)
		removeTempDir(pythonAssets.tempDir)
		removeTempDir(goAssets.tempDir)
	})

	registerInteropTransportMatrix(
		[]transportProtocol{transportJSONRPC, transportREST},
		clients,
		servers,
		nil,
	)
})
