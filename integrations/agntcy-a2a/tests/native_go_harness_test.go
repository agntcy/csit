// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tests

// This file isolates the native Go SDK harness used by the shared interop behaviors.
// Keep the matrix wiring in interop_behaviors_test.go and add Go-specific request and
// assertion helpers here so the cross-SDK behavior registry stays readable and the
// test-only native Go path is easy to find.

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/a2aproject/a2a-go/v2/a2aclient"
	"github.com/a2aproject/a2a-go/v2/a2aclient/agentcard"
	a2agrpc "github.com/a2aproject/a2a-go/v2/a2agrpc/v1"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type goSDKHarness struct{}

func (goSDKHarness) AssertTaskStreaming(ctx context.Context, target interopTarget) {
	client, err := newGoClient(ctx, target.baseURL)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	unaryText, err := goClientUnaryText(ctx, client)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(unaryText).To(gomega.Equal(expectedServerText(target.serverPrefix, requestText)))

	streamText, err := goClientStreamingText(ctx, client)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(streamText).To(gomega.Equal(expectedServerText(target.serverPrefix, requestText)))
}

func (goSDKHarness) AssertTaskLifecycle(ctx context.Context, target interopTarget) {
	client, err := newGoClient(ctx, target.baseURL)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	goClientAssertTaskLifecycle(ctx, client, target.serverPrefix)
}

func (goSDKHarness) AssertPushConfig(ctx context.Context, target interopTarget) {
	client, err := newGoClient(ctx, target.baseURL)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	completedTask, err := goClientSendTask(ctx, client, requestText, false)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(completedTask.Status.State).To(gomega.Equal(a2a.TaskStateCompleted))

	completedText, err := taskStatusText(completedTask)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(completedText).To(gomega.Equal(expectedServerText(target.serverPrefix, requestText)))
	assertTaskHistoryPayload(completedTask, requestText, "push config task history")

	if !target.expectPushSupported {
		goClientAssertPushUnsupported(ctx, client, completedTask.ID)
		return
	}

	goClientAssertPushLifecycle(ctx, client, completedTask.ID)
}

func (goSDKHarness) AssertScenarioParity(ctx context.Context, target interopTarget) {
	client, err := newGoClient(ctx, target.baseURL)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	goClientAssertScenarioParity(ctx, client, target.serverPrefix)
}

func newGoClient(ctx context.Context, baseURL string) (*a2aclient.Client, error) {
	card, err := agentcard.DefaultResolver.Resolve(ctx, baseURL)
	if err != nil {
		return nil, err
	}

	return a2aclient.NewFromCard(
		ctx,
		card,
		a2agrpc.WithGRPCTransport(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
}

func newInteropRequest(text string, returnImmediately bool) *a2a.SendMessageRequest {
	return newInteropRequestWithIDs(text, returnImmediately, "", "")
}

func newInteropRequestWithIDs(text string, returnImmediately bool, taskID a2a.TaskID, contextID string) *a2a.SendMessageRequest {
	message := a2a.NewMessage(
		a2a.MessageRoleUser,
		a2a.NewTextPart(text),
		a2a.NewDataPart(map[string]any{
			"kind":  requestDataKind,
			"scope": requestDataScope,
		}),
	)
	if taskID != "" {
		message.TaskID = taskID
	}
	if contextID != "" {
		message.ContextID = contextID
	}
	message.Metadata = map[string]any{
		requestMetadataKey: requestMetadataValue,
	}

	return &a2a.SendMessageRequest{
		Message: message,
		Config: &a2a.SendMessageConfig{
			ReturnImmediately: returnImmediately,
		},
	}
}

func assertMessageInteropPayload(message *a2a.Message, expectedText string, kind string) {
	gomega.Expect(message).NotTo(gomega.BeNil(), kind)

	text, err := firstMessageText(message)
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), kind)
	gomega.Expect(text).To(gomega.Equal(expectedText), kind)
	gomega.Expect(message.Parts).To(gomega.HaveLen(2), kind)

	dataPart, ok := message.Parts[1].Data().(map[string]any)
	gomega.Expect(ok).To(gomega.BeTrue(), kind)
	gomega.Expect(dataPart).To(gomega.HaveKeyWithValue("kind", requestDataKind), kind)
	gomega.Expect(dataPart).To(gomega.HaveKeyWithValue("scope", requestDataScope), kind)
	gomega.Expect(message.Metadata).To(gomega.HaveKeyWithValue(requestMetadataKey, requestMetadataValue), kind)
}

func assertTaskHistoryPayload(task *a2a.Task, expectedText string, kind string) {
	assertTaskHistoryPayloads(task, []string{expectedText}, kind)
}

func assertTaskHistoryPayloads(task *a2a.Task, expectedTexts []string, kind string) {
	gomega.Expect(task.History).To(gomega.HaveLen(len(expectedTexts)), kind)
	for index, expectedText := range expectedTexts {
		assertMessageInteropPayload(task.History[index], expectedText, kind)
	}
}

func firstMessageText(message *a2a.Message) (string, error) {
	for _, part := range message.Parts {
		if text := part.Text(); text != "" {
			return text, nil
		}
	}

	return "", errors.New("message did not include a text part")
}

func newInteropPushConfig() *a2a.PushConfig {
	return &a2a.PushConfig{
		ID:    "interop-config",
		URL:   "https://example.invalid/webhook",
		Token: "interop-token",
		Auth: &a2a.PushAuthInfo{
			Scheme:      "Bearer",
			Credentials: "interop-credential",
		},
	}
}

func assertTaskPushConfig(config *a2a.PushConfig, taskID a2a.TaskID, expected *a2a.PushConfig, kind string) {
	gomega.Expect(config).NotTo(gomega.BeNil(), kind)
	gomega.Expect(config.TaskID).To(gomega.Equal(taskID), kind)
	gomega.Expect(config.ID).To(gomega.Equal(expected.ID), kind)
	gomega.Expect(config.URL).To(gomega.Equal(expected.URL), kind)
	gomega.Expect(config.Token).To(gomega.Equal(expected.Token), kind)
	gomega.Expect(config.Auth).To(gomega.Equal(expected.Auth), kind)
}

func goClientAssertPushLifecycle(ctx context.Context, client *a2aclient.Client, taskID a2a.TaskID) {
	pushConfig := newInteropPushConfig()

	pushConfig.TaskID = taskID
	createdConfig, err := client.CreateTaskPushConfig(ctx, pushConfig)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	assertTaskPushConfig(createdConfig, taskID, pushConfig, "created push config")

	fetchedConfig, err := client.GetTaskPushConfig(ctx, &a2a.GetTaskPushConfigRequest{
		TaskID: taskID,
		ID:     pushConfig.ID,
	})
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	assertTaskPushConfig(fetchedConfig, taskID, pushConfig, "fetched push config")

	listedConfigs, err := client.ListTaskPushConfigs(ctx, &a2a.ListTaskPushConfigRequest{TaskID: taskID})
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(listedConfigs).To(gomega.HaveLen(1))
	assertTaskPushConfig(listedConfigs[0], taskID, pushConfig, "listed push config")

	err = client.DeleteTaskPushConfig(ctx, &a2a.DeleteTaskPushConfigRequest{
		TaskID: taskID,
		ID:     pushConfig.ID,
	})
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	listedConfigs, err = client.ListTaskPushConfigs(ctx, &a2a.ListTaskPushConfigRequest{TaskID: taskID})
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(listedConfigs).To(gomega.BeEmpty())
}

func goClientAssertPushUnsupported(ctx context.Context, client *a2aclient.Client, taskID a2a.TaskID) {
	pushConfig := newInteropPushConfig()
	expectUnsupportedError := func(err error, kind string) {
		gomega.Expect(err).To(gomega.HaveOccurred(), kind)
		gomega.Expect(strings.ToLower(err.Error())).To(gomega.Or(
			gomega.ContainSubstring("push notification not supported"),
			gomega.ContainSubstring("push_notification_not_supported"),
			gomega.ContainSubstring("not supported"),
			gomega.ContainSubstring("unsupported"),
			gomega.ContainSubstring("unimplemented"),
			gomega.ContainSubstring("server error"),
		), kind)
	}

	pushConfig.TaskID = taskID
	_, err := client.CreateTaskPushConfig(ctx, pushConfig)
	expectUnsupportedError(err, "create push config unsupported")

	_, err = client.GetTaskPushConfig(ctx, &a2a.GetTaskPushConfigRequest{
		TaskID: taskID,
		ID:     pushConfig.ID,
	})
	expectUnsupportedError(err, "get push config unsupported")

	_, err = client.ListTaskPushConfigs(ctx, &a2a.ListTaskPushConfigRequest{TaskID: taskID})
	expectUnsupportedError(err, "list push config unsupported")

	err = client.DeleteTaskPushConfig(ctx, &a2a.DeleteTaskPushConfigRequest{
		TaskID: taskID,
		ID:     pushConfig.ID,
	})
	expectUnsupportedError(err, "delete push config unsupported")
}

func assertTaskNotFoundError(err error, kind string) {
	gomega.Expect(err).To(gomega.HaveOccurred(), kind)
	gomega.Expect(strings.ToLower(err.Error())).To(gomega.Or(
		gomega.ContainSubstring("task not found"),
		gomega.ContainSubstring("not found"),
	), kind)
}

func assertTaskNotCancelableError(err error, kind string) {
	gomega.Expect(err).To(gomega.HaveOccurred(), kind)
	gomega.Expect(strings.ToLower(err.Error())).To(gomega.Or(
		gomega.ContainSubstring("cancel"),
		gomega.ContainSubstring("terminal state"),
		gomega.ContainSubstring("not cancelable"),
	), kind)
}

func taskStatusText(task *a2a.Task) (string, error) {
	if task == nil || task.Status.Message == nil {
		return "", errors.New("task status did not include a message")
	}

	return firstMessageText(task.Status.Message)
}

func eventText(event a2a.Event) (string, bool, error) {
	switch value := event.(type) {
	case *a2a.Message:
		text, err := firstMessageText(value)
		return text, true, err
	case *a2a.Task:
		text, err := taskStatusText(value)
		return text, true, err
	case *a2a.TaskStatusUpdateEvent:
		if value.Status.Message == nil {
			return "", false, nil
		}
		text, err := firstMessageText(value.Status.Message)
		return text, true, err
	default:
		return "", false, nil
	}
}

func firstArtifactText(artifact *a2a.Artifact) (string, error) {
	if artifact == nil {
		return "", errors.New("artifact was nil")
	}

	for _, part := range artifact.Parts {
		if text := part.Text(); text != "" {
			return text, nil
		}
	}

	return "", errors.New("artifact did not include a text part")
}

func goClientSendResult(
	ctx context.Context,
	client *a2aclient.Client,
	text string,
	returnImmediately bool,
	taskID a2a.TaskID,
	contextID string,
) (a2a.SendMessageResult, error) {
	return client.SendMessage(ctx, newInteropRequestWithIDs(text, returnImmediately, taskID, contextID))
}

func goClientSendTask(ctx context.Context, client *a2aclient.Client, text string, returnImmediately bool) (*a2a.Task, error) {
	result, err := goClientSendResult(ctx, client, text, returnImmediately, "", "")
	if err != nil {
		return nil, err
	}

	task, ok := result.(*a2a.Task)
	if !ok {
		return nil, fmt.Errorf("unexpected unary response type %T", result)
	}

	return task, nil
}

func goClientSendMessage(ctx context.Context, client *a2aclient.Client, text string) (*a2a.Message, error) {
	result, err := goClientSendResult(ctx, client, text, false, "", "")
	if err != nil {
		return nil, err
	}

	message, ok := result.(*a2a.Message)
	if !ok {
		return nil, fmt.Errorf("unexpected unary response type %T", result)
	}

	return message, nil
}

func goClientUnaryText(ctx context.Context, client *a2aclient.Client) (string, error) {
	task, err := goClientSendTask(ctx, client, requestText, false)
	if err != nil {
		return "", err
	}

	return taskStatusText(task)
}

func goClientStreamingText(ctx context.Context, client *a2aclient.Client) (string, error) {
	request := newInteropRequest(requestText, false)

	for event, err := range client.SendStreamingMessage(ctx, request) {
		if err != nil {
			return "", err
		}

		text, hasText, textErr := eventText(event)
		if textErr != nil {
			return "", textErr
		}
		if !hasText {
			continue
		}

		return text, nil
	}

	return "", errors.New("stream completed without a message event")
}

func goClientAssertTaskLifecycle(ctx context.Context, client *a2aclient.Client, serverPrefix string) {
	completedTask, err := goClientSendTask(ctx, client, requestText, false)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(completedTask.Status.State).To(gomega.Equal(a2a.TaskStateCompleted))

	completedText, err := taskStatusText(completedTask)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(completedText).To(gomega.Equal(expectedServerText(serverPrefix, requestText)))
	assertTaskHistoryPayload(completedTask, requestText, "completed task history")

	fetchedTask, err := client.GetTask(ctx, &a2a.GetTaskRequest{ID: completedTask.ID})
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(fetchedTask.Status.State).To(gomega.Equal(a2a.TaskStateCompleted))

	fetchedText, err := taskStatusText(fetchedTask)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(fetchedText).To(gomega.Equal(expectedServerText(serverPrefix, requestText)))
	assertTaskHistoryPayload(fetchedTask, requestText, "fetched task history")

	listedTasks, err := client.ListTasks(ctx, &a2a.ListTasksRequest{ContextID: completedTask.ContextID})
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(listedTasks.Tasks).NotTo(gomega.BeEmpty())
	gomega.Expect(listedTasks.Tasks).To(gomega.ContainElement(gomega.HaveField("ID", completedTask.ID)))

	pendingTask, err := goClientSendTask(ctx, client, pendingRequestText, true)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(pendingTask.Status.State).To(gomega.Equal(a2a.TaskStateWorking))

	pendingText, err := taskStatusText(pendingTask)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(pendingText).To(gomega.Equal(expectedServerText(serverPrefix, pendingRequestText)))

	canceledTask, err := client.CancelTask(ctx, &a2a.CancelTaskRequest{ID: pendingTask.ID})
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(canceledTask.Status.State).To(gomega.Equal(a2a.TaskStateCanceled))

	canceledText, err := taskStatusText(canceledTask)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(canceledText).To(gomega.Equal(expectedCancelText(serverPrefix)))

	fetchedCanceledTask, err := client.GetTask(ctx, &a2a.GetTaskRequest{ID: pendingTask.ID})
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(fetchedCanceledTask.Status.State).To(gomega.Equal(a2a.TaskStateCanceled))

	fetchedCanceledText, err := taskStatusText(fetchedCanceledTask)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(fetchedCanceledText).To(gomega.Equal(expectedCancelText(serverPrefix)))

	_, err = client.GetTask(ctx, &a2a.GetTaskRequest{ID: a2a.NewTaskID()})
	assertTaskNotFoundError(err, "missing task should fail")

	_, err = client.CancelTask(ctx, &a2a.CancelTaskRequest{ID: completedTask.ID})
	assertTaskNotCancelableError(err, "canceling a terminal task should fail")
}

func goClientWaitForTaskState(ctx context.Context, client *a2aclient.Client, taskID a2a.TaskID, expectedState a2a.TaskState) *a2a.Task {
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		task, err := client.GetTask(ctx, &a2a.GetTaskRequest{ID: taskID})
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		if task.Status.State == expectedState {
			return task
		}
		time.Sleep(50 * time.Millisecond)
	}

	ginkgo.Fail(fmt.Sprintf("timed out waiting for task %s to reach state %s", taskID, expectedState))
	return nil
}

func assertDataTypesTask(task *a2a.Task, kind string) {
	gomega.Expect(task.Artifacts).To(gomega.HaveLen(1), kind)
	artifact := task.Artifacts[0]
	gomega.Expect(artifact.Parts).To(gomega.HaveLen(3), kind)
	gomega.Expect(artifact.Parts[0].Text()).To(gomega.Equal("structured summary"), kind)
	dataPart, ok := artifact.Parts[1].Data().(map[string]any)
	gomega.Expect(ok).To(gomega.BeTrue(), kind)
	gomega.Expect(dataPart).To(gomega.HaveKeyWithValue("kind", "report"), kind)
	gomega.Expect(dataPart).To(gomega.HaveKeyWithValue("items", float64(2)), kind)
	gomega.Expect(string(artifact.Parts[2].URL())).To(gomega.Equal("https://example.invalid/diagram.svg"), kind)
	gomega.Expect(artifact.Parts[2].MediaType).To(gomega.Equal("image/svg+xml"), kind)
}

func assertExtendedCardMetadata(card *a2a.AgentCard, kind string) {
	gomega.Expect(card).NotTo(gomega.BeNil(), kind)
	gomega.Expect(card.Capabilities.ExtendedAgentCard).To(gomega.BeTrue(), kind)
	gomega.Expect(card.Description).To(gomega.ContainSubstring("(extended)"), kind)

	if len(card.SecuritySchemes) > 0 {
		scheme, ok := card.SecuritySchemes[extendedCardSchemeID]
		gomega.Expect(ok).To(gomega.BeTrue(), kind)
		httpScheme, ok := scheme.(a2a.HTTPAuthSecurityScheme)
		gomega.Expect(ok).To(gomega.BeTrue(), kind)
		gomega.Expect(httpScheme.Scheme).To(gomega.Equal("Bearer"), kind)
	}

	for _, expectedSkill := range expectedSkillIDs {
		gomega.Expect(card.Skills).To(gomega.ContainElement(gomega.HaveField("ID", expectedSkill)), kind)
	}
}

func goClientAssertScenarioParity(ctx context.Context, client *a2aclient.Client, serverPrefix string) {
	messageOnly, err := goClientSendMessage(ctx, client, messageOnlyRequestText)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	messageText, err := firstMessageText(messageOnly)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(messageText).To(gomega.Equal(fmt.Sprintf("%s server message-only response", serverPrefix)))

	failedTask, err := goClientSendTask(ctx, client, taskFailureRequestText, false)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(failedTask.Status.State).To(gomega.Equal(a2a.TaskStateFailed))
	failedText, err := taskStatusText(failedTask)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(failedText).To(gomega.Equal(fmt.Sprintf("%s server failed task", serverPrefix)))

	inputRequiredTask, err := goClientSendTask(ctx, client, multiTurnStartRequestText, false)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(inputRequiredTask.Status.State).To(gomega.Equal(a2a.TaskStateInputRequired))
	inputRequiredText, err := taskStatusText(inputRequiredTask)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(inputRequiredText).To(gomega.Equal(fmt.Sprintf("%s server needs more input", serverPrefix)))
	assertTaskHistoryPayload(inputRequiredTask, multiTurnStartRequestText, "multi-turn start")

	continuedResult, err := goClientSendResult(ctx, client, multiTurnContinueRequestText, false, inputRequiredTask.ID, inputRequiredTask.ContextID)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	continuedTask, ok := continuedResult.(*a2a.Task)
	gomega.Expect(ok).To(gomega.BeTrue())
	gomega.Expect(continuedTask.Status.State).To(gomega.Equal(a2a.TaskStateCompleted))
	continuedText, err := taskStatusText(continuedTask)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(continuedText).To(gomega.Equal(fmt.Sprintf("%s server multi-turn completed", serverPrefix)))
	assertTaskHistoryPayloads(continuedTask, []string{multiTurnStartRequestText, multiTurnContinueRequestText}, "multi-turn continuation")

	streamingChunks := []string{}
	sawStreamingStart := false
	sawStreamingComplete := false
	sawAppend := false
	for event, err := range client.SendStreamingMessage(ctx, newInteropRequest(streamingRequestText, false)) {
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		switch value := event.(type) {
		case *a2a.Task:
			sawStreamingStart = true
			gomega.Expect(value.Status.State).To(gomega.Equal(a2a.TaskStateWorking))
			text, textErr := taskStatusText(value)
			gomega.Expect(textErr).NotTo(gomega.HaveOccurred())
			gomega.Expect(text).To(gomega.Equal(fmt.Sprintf("%s server streaming started", serverPrefix)))
		case *a2a.TaskArtifactUpdateEvent:
			text, textErr := firstArtifactText(value.Artifact)
			gomega.Expect(textErr).NotTo(gomega.HaveOccurred())
			streamingChunks = append(streamingChunks, text)
			if value.Append {
				sawAppend = true
			}
		case *a2a.TaskStatusUpdateEvent:
			sawStreamingComplete = true
			gomega.Expect(value.Status.State).To(gomega.Equal(a2a.TaskStateCompleted))
			text, textErr := firstMessageText(value.Status.Message)
			gomega.Expect(textErr).NotTo(gomega.HaveOccurred())
			gomega.Expect(text).To(gomega.Equal(fmt.Sprintf("%s server streaming complete", serverPrefix)))
		default:
			ginkgo.Fail(fmt.Sprintf("unexpected streaming scenario event type %T", event))
		}
	}
	gomega.Expect(sawStreamingStart).To(gomega.BeTrue())
	gomega.Expect(sawStreamingComplete).To(gomega.BeTrue())
	gomega.Expect(sawAppend).To(gomega.BeTrue())
	gomega.Expect(streamingChunks).To(gomega.Equal([]string{"streaming chunk 1", "streaming chunk 2"}))

	longRunningTask, err := goClientSendTask(ctx, client, longRunningRequestText, true)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	var longRunningCompleted *a2a.Task
	switch longRunningTask.Status.State {
	case a2a.TaskStateWorking:
		longRunningText, textErr := taskStatusText(longRunningTask)
		gomega.Expect(textErr).NotTo(gomega.HaveOccurred())
		gomega.Expect(longRunningText).To(gomega.Equal(fmt.Sprintf("%s server long-running started", serverPrefix)))
		longRunningCompleted = goClientWaitForTaskState(ctx, client, longRunningTask.ID, a2a.TaskStateCompleted)
	case a2a.TaskStateCompleted:
		longRunningCompleted = longRunningTask
	default:
		ginkgo.Fail(fmt.Sprintf("unexpected long-running task state: got %s, want WORKING or COMPLETED", longRunningTask.Status.State))
	}
	longRunningCompletedText, err := taskStatusText(longRunningCompleted)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(longRunningCompletedText).To(gomega.Equal(fmt.Sprintf("%s server long-running complete", serverPrefix)))

	dataTypesTask, err := goClientSendTask(ctx, client, dataTypesRequestText, false)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(dataTypesTask.Status.State).To(gomega.Equal(a2a.TaskStateCompleted))
	dataTypesText, err := taskStatusText(dataTypesTask)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(dataTypesText).To(gomega.Equal(fmt.Sprintf("%s server data-types ready", serverPrefix)))
	assertDataTypesTask(dataTypesTask, "data-types")

	extendedCard, err := client.GetExtendedAgentCard(ctx, &a2a.GetExtendedAgentCardRequest{})
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	assertExtendedCardMetadata(extendedCard, "extended-card")
}
