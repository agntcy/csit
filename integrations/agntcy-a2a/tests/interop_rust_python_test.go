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

var _ = ginkgo.Describe("A2A Rust and Python interoperability", ginkgo.Ordered, ginkgo.Label("suite-rust-python"), func() {
	var (
		rustAssets           fixtureBinaries
		pythonAssets         pythonFixtureAssets
		rustJSONRPCFixture   *fixtureProcess
		pythonJSONRPCFixture *fixtureProcess
		rustRESTFixture      *fixtureProcess
		pythonRESTFixture    *fixtureProcess
		rustJSONRPCURL       string
		pythonJSONRPCURL     string
		rustRESTURL          string
		pythonRESTURL        string
	)

	rustClient := rustProbeHarness{
		getBinaries: func() fixtureBinaries { return rustAssets },
		options: rustProbeOptions{
			expectPushSupported: true,
		},
	}
	pythonClient := pythonProbeHarness{
		getAssets: func() pythonFixtureAssets { return pythonAssets },
		options: rustProbeOptions{
			expectPushSupported: true,
		},
	}
	clients := []interopClientMatrixSpec{
		{label: "rust", displayName: "Rust", harness: rustClient},
		{label: "python", displayName: "Python", harness: pythonClient},
	}
	servers := []interopServerMatrixSpec{
		{
			label:               "rust",
			displayName:         "Rust",
			serverPrefix:        "rust",
			expectPushSupported: true,
			urls: map[transportProtocol]func() string{
				transportJSONRPC: func() string { return rustJSONRPCURL },
				transportREST:    func() string { return rustRESTURL },
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

		rustAssets, err = buildRustFixtureBinaryOnly()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		pythonAssets, err = buildPythonFixtureAssets()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		rustJSONRPCFixture, rustJSONRPCURL, err = startRustFixture(rustAssets, findFreePort(), transportJSONRPC)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		pythonJSONRPCFixture, pythonJSONRPCURL, err = startPythonFixture(pythonAssets, findFreePort(), transportJSONRPC)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		rustRESTFixture, rustRESTURL, err = startRustFixture(rustAssets, findFreePort(), transportREST)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		pythonRESTFixture, pythonRESTURL, err = startPythonFixture(pythonAssets, findFreePort(), transportREST)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	})

	ginkgo.AfterAll(func() {
		stopFixtureIfRunning(pythonRESTFixture)
		stopFixtureIfRunning(rustRESTFixture)
		stopFixtureIfRunning(pythonJSONRPCFixture)
		stopFixtureIfRunning(rustJSONRPCFixture)
		removeTempDir(pythonAssets.tempDir)
		removeTempDir(rustAssets.tempDir)
	})

	registerInteropTransportMatrix(
		[]transportProtocol{transportJSONRPC, transportREST},
		clients,
		servers,
		nil,
	)
})
