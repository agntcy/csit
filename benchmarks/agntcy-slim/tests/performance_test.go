// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tests

import (
	"os/exec"

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/gmeasure"
)

var _ = ginkgo.Describe("SLIM Performance Benchmarks", func() {

	// Objective: Measure Messaging Latency
	// Proof Point: Low-latency communication is a core value prop.
	ginkgo.Context("Messaging Latency", func() {
		ginkgo.It("measures point-to-point message RTT", func() {
			experiment := gmeasure.NewExperiment("P2P Latency Benchmark")
			ginkgo.AddReportEntry(experiment.Name, experiment)

			experiment.SampleDuration("request-response", func(idx int) {
				// Run slim-bench via local go execution (docker may be unavailable)
				// Path assumes running from 'benchmarks/agntcy-slim' dir (where go test runs)
				cmd := exec.Command("go", "run", "../tools/slim-bench/main.go", "-mode=ping-pong", "-duration=1s", "-rate=10")
				output, err := cmd.CombinedOutput()
				gomega.Expect(err).NotTo(gomega.HaveOccurred(), "slim-bench failed: %s", string(output))
			}, gmeasure.SamplingConfig{N: 5})
		})
	})
})
