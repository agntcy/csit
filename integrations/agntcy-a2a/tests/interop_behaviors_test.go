// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tests

// This file is the shared behavior layer for the interop suite. It defines the behavior slices,
// the harness interface each SDK must satisfy, and the helpers that expand client/server matrices
// into labeled Ginkgo specs.
// To add a new cross-SDK test, add a behavior entry here and implement it for each harness that
// should participate. Use the wrapper files only to declare which clients, servers, and overrides
// are in a suite.

import (
	"context"
	"fmt"

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

type interopTarget struct {
	baseURL             string
	serverPrefix        string
	expectPushSupported bool
}

type interopHarness interface {
	AssertUnaryStreaming(ctx context.Context, target interopTarget)
	AssertTaskLifecycle(ctx context.Context, target interopTarget)
	AssertPushConfig(ctx context.Context, target interopTarget)
	AssertScenarioParity(ctx context.Context, target interopTarget)
}

type interopSpecCase struct {
	name    string
	labels  []string
	harness interopHarness
	target  func() interopTarget
}

type interopBehaviorSpec struct {
	name   string
	labels []string
	run    func(ctx context.Context, harness interopHarness, target interopTarget)
}

type interopClientMatrixSpec struct {
	label       string
	displayName string
	harness     interopHarness
}

type interopServerMatrixSpec struct {
	label               string
	displayName         string
	serverPrefix        string
	expectPushSupported bool
	urls                map[transportProtocol]func() string
}

var sharedInteropBehaviorSpecs = []interopBehaviorSpec{
	{
		name:   "covers unary and streaming requests",
		labels: []string{"behavior-core", "behavior-unary-streaming"},
		run: func(ctx context.Context, harness interopHarness, target interopTarget) {
			harness.AssertUnaryStreaming(ctx, target)
		},
	},
	{
		name:   "covers task lifecycle behavior",
		labels: []string{"behavior-core", "behavior-lifecycle"},
		run: func(ctx context.Context, harness interopHarness, target interopTarget) {
			harness.AssertTaskLifecycle(ctx, target)
		},
	},
	{
		name:   "covers push-config behavior",
		labels: []string{"behavior-core", "behavior-push-config"},
		run: func(ctx context.Context, harness interopHarness, target interopTarget) {
			harness.AssertPushConfig(ctx, target)
		},
	},
	{
		name:   "covers scenario parity behavior",
		labels: []string{"behavior-parity"},
		run: func(ctx context.Context, harness interopHarness, target interopTarget) {
			harness.AssertScenarioParity(ctx, target)
		},
	},
}

func interopTargetFor(getBaseURL func() string, serverPrefix string, expectPushSupported bool) func() interopTarget {
	return func() interopTarget {
		return interopTarget{
			baseURL:             getBaseURL(),
			serverPrefix:        serverPrefix,
			expectPushSupported: expectPushSupported,
		}
	}
}

func runInteropBehavior(
	harness interopHarness,
	target func() interopTarget,
	behavior interopBehaviorSpec,
) func(ctx ginkgo.SpecContext) {
	return func(ctx ginkgo.SpecContext) {
		requestCtx, cancel := context.WithTimeout(ctx, probeTimeout)
		defer cancel()

		behavior.run(requestCtx, harness, target())
	}
}

func registerInteropCaseContexts(cases ...interopSpecCase) {
	for _, specCase := range cases {
		specCase := specCase
		ginkgo.Context(specCase.name, func() {
			for _, behavior := range sharedInteropBehaviorSpecs {
				behavior := behavior
				labels := append(append([]string{}, specCase.labels...), behavior.labels...)
				ginkgo.It(
					behavior.name,
					ginkgo.Label(labels...),
					runInteropBehavior(specCase.harness, specCase.target, behavior),
				)
			}
		})
	}
}

func registerInteropTransportMatrix(
	protocols []transportProtocol,
	clients []interopClientMatrixSpec,
	servers []interopServerMatrixSpec,
	pairLabelOverrides map[string]string,
) {
	registerInteropTransportMatrixWithOverrides(protocols, clients, servers, pairLabelOverrides, nil)
}

func registerInteropTransportMatrixWithOverrides(
	protocols []transportProtocol,
	clients []interopClientMatrixSpec,
	servers []interopServerMatrixSpec,
	pairLabelOverrides map[string]string,
	harnessOverrides map[string]interopHarness,
) {
	for _, protocol := range protocols {
		protocol := protocol
		cases := interopCasesForProtocol(protocol, clients, servers, pairLabelOverrides, harnessOverrides)
		if len(cases) == 0 {
			continue
		}

		ginkgo.Context(interopTransportContextName(protocol), func() {
			registerInteropCaseContexts(cases...)
		})
	}
}

func interopCasesForProtocol(
	protocol transportProtocol,
	clients []interopClientMatrixSpec,
	servers []interopServerMatrixSpec,
	pairLabelOverrides map[string]string,
	harnessOverrides map[string]interopHarness,
) []interopSpecCase {
	cases := make([]interopSpecCase, 0, len(clients)*len(servers))
	for _, client := range clients {
		for _, server := range servers {
			pairKey := interopPairKey(client.label, server.label)
			getBaseURL := server.urls[protocol]
			if getBaseURL == nil {
				continue
			}

			harness := client.harness
			if override, ok := harnessOverrides[pairKey]; ok {
				harness = override
			}

			cases = append(cases, interopSpecCase{
				name:    interopCaseName(protocol, client.displayName, server.displayName),
				labels:  []string{string(protocol), interopPairLabel(client.label, server.label, pairLabelOverrides)},
				harness: harness,
				target:  interopTargetFor(getBaseURL, server.serverPrefix, server.expectPushSupported),
			})
		}
	}

	return cases
}

func interopTransportContextName(protocol transportProtocol) string {
	switch protocol {
	case transportJSONRPC:
		return "JSON-RPC transport"
	case transportREST:
		return "HTTP+JSON transport"
	case transportGRPC:
		return "gRPC transport"
	default:
		return fmt.Sprintf("%s transport", protocol)
	}
}

func interopCaseName(protocol transportProtocol, clientDisplayName string, serverDisplayName string) string {
	baseName := fmt.Sprintf("lets the %s client call the %s fixture", clientDisplayName, serverDisplayName)
	switch protocol {
	case transportREST:
		return baseName + " over REST"
	case transportGRPC:
		return baseName + " over gRPC"
	default:
		return baseName
	}
}

func interopPairLabel(clientLabel string, serverLabel string, overrides map[string]string) string {
	if label, ok := overrides[interopPairKey(clientLabel, serverLabel)]; ok {
		return label
	}

	return clientLabel + "-" + serverLabel
}

func interopPairKey(clientLabel string, serverLabel string) string {
	return clientLabel + ":" + serverLabel
}

type probeHarnessRunner func(
	ctx context.Context,
	baseURL string,
	serverPrefix string,
	options rustProbeOptions,
) (string, error)

type externalProbeHarness struct {
	options rustProbeOptions
	run     probeHarnessRunner
}

func (harness externalProbeHarness) optionsForScenario(scenario probeScenario) rustProbeOptions {
	options := harness.options
	options.scenario = scenario
	return options
}

func (harness externalProbeHarness) assertScenario(
	ctx context.Context,
	target interopTarget,
	scenario probeScenario,
) {
	output, err := harness.run(
		ctx,
		target.baseURL,
		target.serverPrefix,
		harness.optionsForScenario(scenario),
	)
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), output)
}

func (harness externalProbeHarness) AssertUnaryStreaming(ctx context.Context, target interopTarget) {
	harness.assertScenario(ctx, target, probeScenarioUnaryStreaming)
}

func (harness externalProbeHarness) AssertTaskLifecycle(ctx context.Context, target interopTarget) {
	harness.assertScenario(ctx, target, probeScenarioTaskLifecycle)
}

func (harness externalProbeHarness) AssertPushConfig(ctx context.Context, target interopTarget) {
	harness.assertScenario(ctx, target, probeScenarioPushConfig)
}

func (harness externalProbeHarness) AssertScenarioParity(ctx context.Context, target interopTarget) {
	harness.assertScenario(ctx, target, probeScenarioParity)
}

func newRustProbeHarness(
	getBinaries func() fixtureBinaries,
	options rustProbeOptions,
) interopHarness {
	return externalProbeHarness{
		options: options,
		run: func(
			ctx context.Context,
			baseURL string,
			serverPrefix string,
			options rustProbeOptions,
		) (string, error) {
			return runRustProbe(ctx, getBinaries(), baseURL, serverPrefix, options)
		},
	}
}

func newDotNetProbeHarness(
	getBinaries func() dotNetFixtureBinaries,
	options dotNetProbeOptions,
) interopHarness {
	return externalProbeHarness{
		options: options,
		run: func(
			ctx context.Context,
			baseURL string,
			serverPrefix string,
			options rustProbeOptions,
		) (string, error) {
			return runDotNetProbe(ctx, getBinaries(), baseURL, serverPrefix, options)
		},
	}
}

func newPythonProbeHarness(
	getAssets func() pythonFixtureAssets,
	options rustProbeOptions,
) interopHarness {
	return externalProbeHarness{
		options: options,
		run: func(
			ctx context.Context,
			baseURL string,
			serverPrefix string,
			options rustProbeOptions,
		) (string, error) {
			return runPythonProbe(ctx, getAssets(), baseURL, serverPrefix, options)
		},
	}
}
