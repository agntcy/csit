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
		ginkgo.It("measures request-reply message RTT", func() {
			experiment := gmeasure.NewExperiment("P2P Latency Benchmark")
			ginkgo.AddReportEntry(experiment.Name, experiment)

			experiment.SampleDuration("request-reply", func(idx int) {
				stopEchoResponder()
				startEchoResponder("echo", 1, "")
				cmd := exec.Command(
					rateClientPath,
					"-mode", "request-reply",
					"-msgs", "10",
					"-rate", "10",
					"-local", "agntcy/demo/client",
					"-server", serverEndpoint,
					"-dest", "agntcy/demo/echo",
				)
				session, err := gexec.Start(cmd, ginkgo.GinkgoWriter, ginkgo.GinkgoWriter)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				gomega.Eventually(session, 20*time.Second).Should(gexec.Exit(0))
			}, gmeasure.SamplingConfig{N: 1})
		})

		ginkgo.It("measures sender write throughput without a responder", func() {
			experiment := gmeasure.NewExperiment("Sender Write Benchmark")
			ginkgo.AddReportEntry(experiment.Name, experiment)

			experiment.SampleDuration("write", func(idx int) {
				stopEchoResponder()
				startEchoResponder("blackhole", 1, "")
				cmd := exec.Command(
					rateClientPath,
					"-mode", "write",
					"-msgs", "100",
					"-local", "agntcy/demo/client",
					"-server", serverEndpoint,
					"-dest", "agntcy/demo/echo",
				)
				session, err := gexec.Start(cmd, ginkgo.GinkgoWriter, ginkgo.GinkgoWriter)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				gomega.Eventually(session, 20*time.Second).Should(gexec.Exit(0))
			}, gmeasure.SamplingConfig{N: 1})
		})

		ginkgo.It("measures fire-and-forget throughput", func() {
			experiment := gmeasure.NewExperiment("Fire-And-Forget Benchmark")
			ginkgo.AddReportEntry(experiment.Name, experiment)

			experiment.SampleDuration("fire-and-forget", func(idx int) {
				stopEchoResponder()
				startEchoResponder("sink", 1, "")
				cmd := exec.Command(
					rateClientPath,
					"-mode", "fire-and-forget",
					"-msgs", "100",
					"-rate", "100",
					"-local", "agntcy/demo/client",
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
				rateClientPath,
				"-mode", "sub",
				"-msgs", "10",
				"-local", "agntcy/demo/client",
				"-server", serverEndpoint,
				"-dest", "agntcy/demo/echo",
			)
			session, err := gexec.Start(cmd, ginkgo.GinkgoWriter, ginkgo.GinkgoWriter)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Eventually(session, 10*time.Second).Should(gexec.Exit(1))
			gomega.Eventually(session.Err, 5*time.Second).Should(gbytes.Say("unsupported mode \"sub\""))
		})
	})
})
