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

var _ = ginkgo.Describe("A2A Rust and Go interoperability", ginkgo.Ordered, ginkgo.Label("suite-rust-go"), func() {
	var (
		binaries              fixtureBinaries
		goJSONRPCFixture      *fixtureProcess
		rustJSONRPCFixture    *fixtureProcess
		goRESTFixture         *fixtureProcess
		rustRESTFixture       *fixtureProcess
		goGRPCFixture         *fixtureProcess
		rustGRPCFixture       *fixtureProcess
		goJSONRPCFixtureURL   string
		rustJSONRPCFixtureURL string
		goRESTFixtureURL      string
		rustRESTFixtureURL    string
		goGRPCFixtureURL      string
		rustGRPCFixtureURL    string
	)

	goClient := goSDKHarness{}
	rustClient := rustProbeHarness{
		getBinaries: func() fixtureBinaries { return binaries },
		options: rustProbeOptions{
			expectPushSupported: true,
		},
	}
	clients := []interopClientMatrixSpec{
		{label: "go", displayName: "Go", harness: goClient},
		{label: "rust", displayName: "Rust", harness: rustClient},
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
				transportGRPC:    func() string { return goGRPCFixtureURL },
			},
		},
		{
			label:               "rust",
			displayName:         "Rust",
			serverPrefix:        "rust",
			expectPushSupported: true,
			urls: map[transportProtocol]func() string{
				transportJSONRPC: func() string { return rustJSONRPCFixtureURL },
				transportREST:    func() string { return rustRESTFixtureURL },
				transportGRPC:    func() string { return rustGRPCFixtureURL },
			},
		},
	}

	ginkgo.BeforeAll(func() {
		var err error

		binaries, err = buildFixtureBinaries()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		goJSONRPCFixture, goJSONRPCFixtureURL, err = startGoFixture(binaries, findFreePort(), transportJSONRPC)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		rustJSONRPCFixture, rustJSONRPCFixtureURL, err = startRustFixture(binaries, findFreePort(), transportJSONRPC)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		goRESTFixture, goRESTFixtureURL, err = startGoFixture(binaries, findFreePort(), transportREST)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		rustRESTFixture, rustRESTFixtureURL, err = startRustFixture(binaries, findFreePort(), transportREST)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		goGRPCFixture, goGRPCFixtureURL, err = startGoFixture(binaries, findFreePort(), transportGRPC)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		rustGRPCFixture, rustGRPCFixtureURL, err = startRustFixture(binaries, findFreePort(), transportGRPC)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	})

	ginkgo.AfterAll(func() {
		stopFixtureIfRunning(rustGRPCFixture)
		stopFixtureIfRunning(goGRPCFixture)
		stopFixtureIfRunning(rustRESTFixture)
		stopFixtureIfRunning(goRESTFixture)
		stopFixtureIfRunning(rustJSONRPCFixture)
		stopFixtureIfRunning(goJSONRPCFixture)
		removeTempDir(binaries.tempDir)
	})

	registerInteropTransportMatrix(
		[]transportProtocol{transportJSONRPC, transportREST, transportGRPC},
		clients,
		servers,
		nil,
	)
})
