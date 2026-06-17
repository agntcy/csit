// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tests

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var (
	sharedAssets *fixtureAssets
	skipAll      bool
)

var _ = ginkgo.BeforeSuite(func() {
	if os.Getenv("SKIP_SLIM_A2A") == "1" {
		skipAll = true
		return
	}

	root := componentRoot()
	cache := filepath.Join(root, ".cache")
	if err := os.MkdirAll(cache, 0o755); err != nil {
		ginkgo.Fail(fmt.Sprintf("mkdir cache: %v", err))
	}

	buildCtx, cancel := context.WithTimeout(context.Background(), buildTimeout)
	defer cancel()

	if err := ensureSlimBindingsSetup(buildCtx); err != nil {
		ginkgo.Fail(err.Error())
	}

	goSrv := filepath.Join(cache, "csit-slim-go-server-"+slimBindingsCacheTag)
	goPrb := filepath.Join(cache, "csit-slim-go-probe-"+slimBindingsCacheTag)
	if err := buildGoFixture(buildCtx, root, goSrv, goPrb); err != nil {
		ginkgo.Fail(err.Error())
	}

	pyDir := filepath.Join(root, "fixtures", "python")
	assets := &fixtureAssets{
		goServerBin:      goSrv,
		goProbeBin:       goPrb,
		pythonFixtureDir: pyDir,
	}
	if !slimGoOnly() {
		venv := filepath.Join(cache, "csit-slim-venv")
		pyBin, err := ensurePythonVenv(buildCtx, root, venv)
		if err != nil {
			ginkgo.Fail(err.Error())
		}
		assets.pythonBin = pyBin
	}
	sharedAssets = assets
})

var _ = ginkgo.Describe("A2A SLIMRPC interoperability", ginkgo.Ordered, ginkgo.Label("suite-slim-a2a"), func() {
	ginkgo.BeforeEach(func() {
		if skipAll {
			ginkgo.Skip("SKIP_SLIM_A2A=1")
		}
		if !slimReachable(slimServerURL()) {
			ginkgo.Skip("SLIM node not reachable at " + slimServerURL() + " (set SLIM_SERVER or start slimctl)")
		}
		gomega.Expect(sharedAssets).NotTo(gomega.BeNil())
	})

	langs := interopLanguages()
	payloads := interopPayloads()
	for _, srv := range langs {
		for _, cli := range langs {
			srv := srv
			cli := cli

			for _, pc := range payloads {
				pc := pc
				ginkgo.It(
					fmt.Sprintf("SLIMRPC client %s calls server %s and echoes the %s payload", cli, srv, pc.name),
					ginkgo.Label("behavior-core", "pair-"+cli+"-"+srv, "payload-"+pc.name),
					func() {
						out, err := runInteropProbe(srv, cli, scenarioEcho, pc.text)
						gomega.Expect(err).NotTo(gomega.HaveOccurred(), fmt.Sprintf("probe output:\n%s", out))
						gomega.Expect(out).To(gomega.ContainSubstring(pc.text), fmt.Sprintf("probe output:\n%s", out))
					},
				)
			}

			for _, sc := range interopScenarios() {
				sc := sc
				ginkgo.It(
					fmt.Sprintf("SLIMRPC client %s observes the %s response from server %s", cli, sc.name, srv),
					ginkgo.Label("behavior-scenario", "pair-"+cli+"-"+srv, "scenario-"+sc.name),
					func() {
						out, err := runInteropProbe(srv, cli, sc.scenario, "")
						gomega.Expect(err).NotTo(gomega.HaveOccurred(), fmt.Sprintf("probe output:\n%s", out))

						kind, ok := parseProbeField(out, "CSIT_SLIM_RESULT_KIND")
						gomega.Expect(ok).To(gomega.BeTrue(), fmt.Sprintf("probe output missing CSIT_SLIM_RESULT_KIND:\n%s", out))
						gomega.Expect(kind).To(gomega.Equal(sc.wantKind), fmt.Sprintf("probe output:\n%s", out))

						if sc.wantState != "" {
							state, ok := parseProbeField(out, "CSIT_SLIM_TASK_STATE")
							gomega.Expect(ok).To(gomega.BeTrue(), fmt.Sprintf("probe output missing CSIT_SLIM_TASK_STATE:\n%s", out))
							gomega.Expect(state).To(gomega.Equal(sc.wantState), fmt.Sprintf("probe output:\n%s", out))
						}

						if len(sc.wantArtifactSubstrings) > 0 {
							artifactText, ok := parseProbeField(out, "CSIT_SLIM_ARTIFACT_TEXT")
							gomega.Expect(ok).To(gomega.BeTrue(), fmt.Sprintf("probe output missing CSIT_SLIM_ARTIFACT_TEXT:\n%s", out))
							for _, sub := range sc.wantArtifactSubstrings {
								gomega.Expect(artifactText).To(gomega.ContainSubstring(sub), fmt.Sprintf("probe output:\n%s", out))
							}
						}
					},
				)
			}

			// Lifecycle assertions (terminal task state + echoed artifact presence) are
			// payload-independent, so they run once per pair on the canonical ASCII payload.
			ginkgo.It(
				fmt.Sprintf("SLIMRPC client %s observes a completed task with an echoed artifact from server %s", cli, srv),
				ginkgo.Label("behavior-lifecycle", "pair-"+cli+"-"+srv),
				func() {
					out, err := runInteropProbe(srv, cli, scenarioEcho, probeText)
					gomega.Expect(err).NotTo(gomega.HaveOccurred(), fmt.Sprintf("probe output:\n%s", out))

					state, ok := parseProbeField(out, "CSIT_SLIM_TASK_STATE")
					gomega.Expect(ok).To(gomega.BeTrue(), fmt.Sprintf("probe output missing CSIT_SLIM_TASK_STATE:\n%s", out))
					gomega.Expect(state).To(gomega.Equal("TASK_STATE_COMPLETED"), fmt.Sprintf("probe output:\n%s", out))

					present, ok := parseProbeField(out, "CSIT_SLIM_ARTIFACT_PRESENT")
					gomega.Expect(ok).To(gomega.BeTrue(), fmt.Sprintf("probe output missing CSIT_SLIM_ARTIFACT_PRESENT:\n%s", out))
					gomega.Expect(present).To(gomega.Equal("true"), fmt.Sprintf("probe output:\n%s", out))

					artifactText, ok := parseProbeField(out, "CSIT_SLIM_ARTIFACT_TEXT")
					gomega.Expect(ok).To(gomega.BeTrue(), fmt.Sprintf("probe output missing CSIT_SLIM_ARTIFACT_TEXT:\n%s", out))
					gomega.Expect(artifactText).To(gomega.ContainSubstring(probeText), fmt.Sprintf("probe output:\n%s", out))
				},
			)
		}
	}
})

// runInteropProbe starts the server fixture for srv, waits for its ready marker, runs the
// cli probe against it with the given outbound text, and returns the probe's combined output.
// Cleanup of the server process is registered on the calling spec via ginkgo.DeferCleanup.
func runInteropProbe(srv, cli, scenario, text string) (string, error) {
	ctx := context.Background()
	logs := &lockedBuffer{}
	u := slimServerURL()
	sec := slimSharedSecret()

	cmd, err := startServer(ctx, srv, u, sec, sharedAssets, logs)
	if err != nil {
		return "", err
	}
	ginkgo.DeferCleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Signal(syscall.SIGTERM)
			_, _ = cmd.Process.Wait()
		}
	})

	if err := waitServerReady(cmd, logs, readyMarkerForServer(srv)); err != nil {
		return "", err
	}

	remote := serverIdentity(srv)
	return runProbe(ctx, cli, u, sec, remote, scenario, text, sharedAssets)
}

// payloadCase is an echo payload variation exercised across every language pair.
// Texts are deliberately single-line: probes emit CSIT_SLIM_* results as newline
// delimited KEY=value lines, so an embedded newline would break parseProbeField.
type payloadCase struct {
	name string
	text string
}

// interopPayloads returns the echo payloads driven through the behavior-core specs.
// Each exercises wire-level text handling — UTF-8/emoji, shell/JSON metacharacters, and a
// larger multi-KiB frame — round-tripped verbatim, since both echo executors return the
// input text unchanged. Filter a single case with: ginkgo --label-filter 'payload-unicode'.
func interopPayloads() []payloadCase {
	return []payloadCase{
		{name: "ascii", text: probeText},
		{name: "unicode", text: "Ünïcödé こんにちは 🌐 ☃ Привет"},
		{name: "symbols", text: "sym &<>\"'`%${}|;:!?#@*()[]/ end"},
		{name: "long", text: strings.Repeat("slimrpc-echo-0123456789 ", 512)},
	}
}

// scenarioCase is a non-echo server behavior exercised across every language pair.
// scenario is the probe's --scenario selector; wantKind is the expected
// CSIT_SLIM_RESULT_KIND ("message" or "task"); wantState is the expected terminal
// CSIT_SLIM_TASK_STATE (empty for a bare message, which carries no task state).
type scenarioCase struct {
	name      string
	scenario  string
	wantKind  string
	wantState string
	// wantArtifactSubstrings, when set, asserts the aggregated CSIT_SLIM_ARTIFACT_TEXT
	// contains each substring. For streaming this proves multiple streamed artifact
	// chunks were delivered and aggregated in order (a portable signal: the Python
	// client coalesces stream events, so a raw event count is not comparable to Go).
	wantArtifactSubstrings []string
}

// interopScenarios returns the scenario behaviors driven through the behavior-scenario
// specs, mirroring the agntcy-a2a sibling taxonomy. Filter a single case with:
// ginkgo --label-filter 'scenario-task-failure'.
func interopScenarios() []scenarioCase {
	return []scenarioCase{
		{name: scenarioMessageOnly, scenario: scenarioMessageOnly, wantKind: "message", wantState: ""},
		{name: scenarioTaskFailure, scenario: scenarioTaskFailure, wantKind: "task", wantState: "TASK_STATE_FAILED"},
		{name: scenarioInputRequired, scenario: scenarioInputRequired, wantKind: "task", wantState: "TASK_STATE_INPUT_REQUIRED"},
		{
			name:                   scenarioStreaming,
			scenario:               scenarioStreaming,
			wantKind:               "task",
			wantState:              "TASK_STATE_COMPLETED",
			wantArtifactSubstrings: []string{"streaming chunk 1", "streaming chunk 2"},
		},
		{name: scenarioTaskCancel, scenario: scenarioTaskCancel, wantKind: "task", wantState: "TASK_STATE_CANCELED"},
	}
}

// parseProbeField extracts the value of a `KEY=value` line emitted by the probe fixtures.
func parseProbeField(out, key string) (string, bool) {
	prefix := key + "="
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.HasPrefix(line, prefix) {
			return strings.TrimPrefix(line, prefix), true
		}
	}
	return "", false
}
