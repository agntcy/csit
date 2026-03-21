// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tests

import (
	"os/exec"
	"time"

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"github.com/onsi/gomega/gmeasure"
)

var _ = ginkgo.Describe("SLIM Performance Benchmarks", func() {
	ginkgo.BeforeEach(func() {
		startLocalSlimStack()
	})

	ginkgo.AfterEach(func() {
		stopLocalSlimStack()
	})

	// Objective: Measure Messaging Latency
	// Proof Point: Low-latency communication is a core value prop.
	ginkgo.Context("Messaging Latency", func() {
		ginkgo.It("measures request message RTT", func() {
			experiment := gmeasure.NewExperiment("P2P Latency Benchmark")
			ginkgo.AddReportEntry(experiment.Name, experiment)

			experiment.SampleDuration("request-response", func(idx int) {
				cmd := exec.Command(
					slimBenchPath,
					"-mode", "request",
					"-msgs", "10",
					"-rate", "10",
					"-server", serverEndpoint,
					"-dest", "agntcy/demo/echo",
				)
				session, err := gexec.Start(cmd, ginkgo.GinkgoWriter, ginkgo.GinkgoWriter)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				gomega.Eventually(session, 20*time.Second).Should(gexec.Exit(0))
			}, gmeasure.SamplingConfig{N: 1})
		})

		ginkgo.It("measures ping-pong alias RTT", func() {
			experiment := gmeasure.NewExperiment("Ping Pong Alias Benchmark")
			ginkgo.AddReportEntry(experiment.Name, experiment)

			experiment.SampleDuration("ping-pong", func(idx int) {
				cmd := exec.Command(
					slimBenchPath,
					"-mode", "ping-pong",
					"-msgs", "10",
					"-rate", "10",
					"-server", serverEndpoint,
					"-dest", "agntcy/demo/echo",
				)
				session, err := gexec.Start(cmd, ginkgo.GinkgoWriter, ginkgo.GinkgoWriter)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				gomega.Eventually(session, 20*time.Second).Should(gexec.Exit(0))
			}, gmeasure.SamplingConfig{N: 1})
		})
	})

	ginkgo.Context("Unsupported Live Modes", func() {
		ginkgo.It("rejects sub mode against a live SLIM node", func() {
			cmd := exec.Command(
				slimBenchPath,
				"-mode", "sub",
				"-msgs", "10",
				"-server", serverEndpoint,
				"-dest", "agntcy/demo/echo",
			)
			session, err := gexec.Start(cmd, ginkgo.GinkgoWriter, ginkgo.GinkgoWriter)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Eventually(session, 10*time.Second).Should(gexec.Exit(1))
			gomega.Eventually(session.Err, 5*time.Second).Should(gbytes.Say("sub mode is not implemented for live SLIM benchmarks"))
		})
	})
})
