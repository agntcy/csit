// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tests

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"time"

	"github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/a2aproject/a2a-go/v2/a2aclient"
	"github.com/a2aproject/a2a-go/v2/a2aclient/agentcard"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

func (harness goSDKHarness) AssertJSONRPCTransportRecovery(ctx context.Context, target interopTarget) {
	backend, err := url.Parse(target.baseURL)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	proxy := newDisconnectingProxy(backend)
	ginkgo.DeferCleanup(proxy.close)

	card, err := agentcard.DefaultResolver.Resolve(ctx, target.baseURL)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(card.SupportedInterfaces).To(gomega.HaveLen(1))
	endpoint := *card.SupportedInterfaces[0]
	gomega.Expect(endpoint.ProtocolBinding).To(gomega.Equal(a2a.TransportProtocolJSONRPC))
	endpointURL, err := url.Parse(endpoint.URL)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(endpointURL.Host).To(gomega.Equal(backend.Host))
	proxyURL, err := url.Parse(proxy.server.URL)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	endpointURL.Scheme = proxyURL.Scheme
	endpointURL.Host = proxyURL.Host
	endpoint.URL = endpointURL.String()
	card.SupportedInterfaces = []*a2a.AgentInterface{&endpoint}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	ginkgo.DeferCleanup(transport.CloseIdleConnections)
	client, err := a2aclient.NewFromCard(ctx, card, a2aclient.WithJSONRPCTransport(
		&http.Client{Transport: transport, Timeout: 5 * time.Second},
	))
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	ginkgo.DeferCleanup(client.Destroy)

	ginkgo.By("creating an input-required task through the fault boundary")
	startRequest := newInteropRequest(multiTurnStartRequestText, false)
	result, err := client.SendMessage(ctx, startRequest)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	initialTask, ok := result.(*a2a.Task)
	gomega.Expect(ok).To(gomega.BeTrue())
	gomega.Expect(initialTask.ID).NotTo(gomega.BeEmpty())
	gomega.Expect(initialTask.ContextID).NotTo(gomega.BeEmpty())
	harness.assertRecoveryTask(initialTask, initialTask, a2a.TaskStateInputRequired, startRequest.Message)

	ginkgo.By("disconnecting the transport and observing a failed read without forwarding it")
	forwardedBeforeCut := proxy.forwarded.Load()
	proxy.disconnect()
	// Make the failed POST reach the cut proxy instead of racing a stale idle connection.
	transport.CloseIdleConnections()
	readCtx, cancelRead := context.WithTimeout(ctx, 5*time.Second)
	defer cancelRead()
	unavailableTask, err := client.GetTask(readCtx, &a2a.GetTaskRequest{ID: initialTask.ID})
	gomega.Expect(err).To(gomega.HaveOccurred(), "GetTask must fail while the transport is cut")
	gomega.Expect(unavailableTask).To(gomega.BeNil())
	gomega.Expect(readCtx.Err()).NotTo(gomega.HaveOccurred(), "the disconnect must not rely on a timeout")
	gomega.Expect(proxy.dropped.Load()).To(gomega.BeNumerically(">", 0), "the request must hit the fault boundary")
	gomega.Expect(proxy.forwarded.Load()).To(gomega.Equal(forwardedBeforeCut), "the failed read must not reach the fixture")

	ginkgo.By("restoring the transport and fetching the same task with the same client")
	proxy.restore()
	recoveredTask, err := client.GetTask(ctx, &a2a.GetTaskRequest{ID: initialTask.ID})
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	harness.assertRecoveryTask(recoveredTask, initialTask, a2a.TaskStateInputRequired, startRequest.Message)

	ginkgo.By("completing that task with a new message and checking the stored result")
	continueRequest := newInteropRequestWithIDs(
		multiTurnContinueRequestText, false, initialTask.ID, initialTask.ContextID,
	)
	result, err = client.SendMessage(ctx, continueRequest)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	completedTask, ok := result.(*a2a.Task)
	gomega.Expect(ok).To(gomega.BeTrue())
	harness.assertRecoveryTask(
		completedTask, initialTask, a2a.TaskStateCompleted, startRequest.Message, continueRequest.Message,
	)
	completedText, err := taskStatusText(completedTask)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(completedText).To(gomega.Equal(fmt.Sprintf("%s server multi-turn completed", target.serverPrefix)))

	storedTask, err := client.GetTask(ctx, &a2a.GetTaskRequest{ID: initialTask.ID})
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	harness.assertRecoveryTask(
		storedTask, initialTask, a2a.TaskStateCompleted, startRequest.Message, continueRequest.Message,
	)

	ginkgo.AddReportEntry("transport-recovery", map[string]any{
		"taskId":            storedTask.ID,
		"contextId":         storedTask.ContextID,
		"startMessageId":    startRequest.Message.ID,
		"continueMessageId": continueRequest.Message.ID,
		"droppedRequests":   proxy.dropped.Load(),
		"finalState":        storedTask.Status.State,
	})
}

func (goSDKHarness) assertRecoveryTask(task *a2a.Task, initial *a2a.Task, state a2a.TaskState, messages ...*a2a.Message) {
	gomega.Expect(task).NotTo(gomega.BeNil())
	gomega.Expect(task.ID).To(gomega.Equal(initial.ID), "task identity must survive the transport cut")
	gomega.Expect(task.ContextID).To(gomega.Equal(initial.ContextID), "context identity must survive the transport cut")
	gomega.Expect(task.Status.State).To(gomega.Equal(state))

	// Match by sender-assigned identity, not position or delivery count. No send
	// is retried across the cut, and no exactly-once guarantee is assumed.
	for _, expected := range messages {
		index := slices.IndexFunc(task.History, func(message *a2a.Message) bool {
			return message != nil &&
				message.ID == expected.ID
		})
		gomega.Expect(index).To(gomega.BeNumerically(">=", 0), "task history must preserve message ID %s", expected.ID)
		text, err := firstMessageText(expected)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		actual := task.History[index]
		gomega.Expect(actual.Role).To(gomega.Equal(expected.Role))
		assertMessageInteropPayload(actual, text, "recovered task history")
	}
}
