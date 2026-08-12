// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tests

import (
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = ginkgo.Describe("Agent Consensus Matrix", ginkgo.Label("matrix-cell"), func() {
	ginkgo.It("runs repeated P2P and SLIM benchmarks for one matrix cell", func() {
		if envString("CONSENSUS_RUN_MATRIX", "") == "" {
			ginkgo.Skip("set CONSENSUS_RUN_MATRIX=1 to run a matrix cell benchmark")
		}

		cfg := loadMatrixConfig()
		if cfg.LatencyMs < 0 || cfg.Agents < 0 || cfg.PayloadBytes < 0 {
			ginkgo.Skip("MATRIX_LATENCY_MS, MATRIX_AGENTS, and MATRIX_PAYLOAD_BYTES are required")
		}
		gomega.Expect(cfg.validateBinaries()).To(gomega.Succeed())

		ginkgo.By("running matrix cell repeats")
		runMatrixCell(cfg)

		gomega.Expect(cfg.CellTSV).To(gomega.BeAnExistingFile())
		ginkgo.AddReportEntry("Matrix cell TSV", cfg.CellTSV)
	})
})
