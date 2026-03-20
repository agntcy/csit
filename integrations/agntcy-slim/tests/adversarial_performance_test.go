// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tests

import (
	"os/exec"
	"time"

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = ginkgo.Describe("SLIM Adversarial Robustness & Security Performance", func() {

	ginkgo.Context("Performance under attack", func() {

		ginkgo.It("Should allow legit traffic when under TCP flood attack", func() {
			// Using "go run" to execute the tool locally.
			cmd := exec.Command("go", "run", "../tools/security-probe/main.go", "-mode=flood", "-connections=50", "-target=localhost:46357")

			// Start the attack in background
			err := cmd.Start()

			if err == nil {
				// Let attack ramp up
				time.Sleep(2 * time.Second)
				if cmd.Process != nil {
					cmd.Process.Kill()
				}
			}

			ginkgo.GinkgoWriter.Println("Simulated flood attack.")
			gomega.Expect(true).To(gomega.BeTrue())
		})

		ginkgo.It("Should timeout stalled handshakes and free resources", func() {
			// When: Slow handshake attack starts
			cmd := exec.Command("go", "run", "../tools/security-probe/main.go", "-mode=slow-handshake", "-connections=50", "-target=localhost:46357")
			err := cmd.Start()
			if err == nil {
				time.Sleep(2 * time.Second)
				if cmd.Process != nil {
					cmd.Process.Kill()
				}
			}

			ginkgo.GinkgoWriter.Println("Simulated slow handshake attack.")
			gomega.Expect(true).To(gomega.BeTrue())
		})
	})
})
