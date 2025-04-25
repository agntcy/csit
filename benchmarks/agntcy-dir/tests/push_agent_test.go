// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tests

import (
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega/gmeasure"
)

var _ = ginkgo.Describe("Benchmarking agent push", func() {
	ginkgo.It("Measures agent push run 10 times", func() {
		experiment := gmeasure.NewExperiment("Agent push Benchmark")
		experiment.SampleDuration("agent push test", func(_ int) {
			runTest()
		}, gmeasure.SamplingConfig{N: 10})

		ginkgo.GinkgoWriter.Println(experiment.String())
	})
})

func runTest() {
}
