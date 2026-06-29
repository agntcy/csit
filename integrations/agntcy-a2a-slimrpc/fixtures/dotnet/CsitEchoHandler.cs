// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

using A2A;

namespace Csit;

/// <summary>
/// CSIT echo agent. Mirrors the Go echoExecutor (fixtures/go/cmd/server/main.go): the
/// default behavior echoes the request text back as a completed task's artifact; the
/// scenario sentinels drive specific task lifecycle states the harness asserts.
/// </summary>
internal sealed class CsitEchoHandler : IAgentHandler
{
    public async Task ExecuteAsync(RequestContext context, AgentEventQueue queue, CancellationToken cancellationToken)
    {
        var text = context.UserText ?? string.Empty;
        var updater = new TaskUpdater(queue, context.TaskId, context.ContextId);
        var isNew = context.Task is null;

        switch (text)
        {
            case Scenarios.SentinelMessageOnly:
                // Bare message response: no task is created.
                await queue.EnqueueMessageAsync(
                    new Message
                    {
                        Role = Role.Agent,
                        MessageId = Guid.NewGuid().ToString("N"),
                        ContextId = context.ContextId,
                        Parts = [Part.FromText(Scenarios.MessageOnlyText)],
                    },
                    cancellationToken).ConfigureAwait(false);
                break;

            case Scenarios.SentinelTaskFailure:
                if (isNew) await updater.SubmitAsync(cancellationToken).ConfigureAwait(false);
                await updater.FailAsync(null, cancellationToken).ConfigureAwait(false);
                break;

            case Scenarios.SentinelInputRequired:
                if (isNew) await updater.SubmitAsync(cancellationToken).ConfigureAwait(false);
                await updater.RequireInputAsync(null, cancellationToken).ConfigureAwait(false);
                break;

            case Scenarios.SentinelMultiTurn:
                // Turn 1: pause for more input. The probe continues this same task (by ID)
                // with SentinelMultiTurnContinue.
                if (isNew) await updater.SubmitAsync(cancellationToken).ConfigureAwait(false);
                await updater.RequireInputAsync(null, cancellationToken).ConfigureAwait(false);
                break;

            case Scenarios.SentinelMultiTurnContinue:
                // Turn 2: the task already exists (continuation). The unary SendMessage response
                // is built from the first Task/Message event, and only SubmitAsync emits a Task
                // event — so re-surface the existing task before the status/artifact updates,
                // otherwise the server reports "no response events".
                if (isNew)
                    await updater.SubmitAsync(cancellationToken).ConfigureAwait(false);
                else
                    await queue.EnqueueTaskAsync(context.Task!, cancellationToken).ConfigureAwait(false);
                await updater.AddArtifactAsync([Part.FromText(Scenarios.MultiTurnCompleteText)],
                    cancellationToken: cancellationToken).ConfigureAwait(false);
                await updater.CompleteAsync(null, cancellationToken).ConfigureAwait(false);
                break;

            case Scenarios.SentinelStreaming:
                // Multiple status + artifact events so a streaming client observes a stream.
                if (isNew) await updater.SubmitAsync(cancellationToken).ConfigureAwait(false);
                await updater.StartWorkAsync(null, cancellationToken).ConfigureAwait(false);
                await updater.AddArtifactAsync([Part.FromText("streaming chunk 1 ")],
                    cancellationToken: cancellationToken).ConfigureAwait(false);
                await updater.AddArtifactAsync([Part.FromText("streaming chunk 2")],
                    cancellationToken: cancellationToken).ConfigureAwait(false);
                await updater.CompleteAsync(null, cancellationToken).ConfigureAwait(false);
                break;

            case Scenarios.SentinelCancel:
                // Leave the task in a non-terminal working state so CancelTask can cancel it.
                if (isNew) await updater.SubmitAsync(cancellationToken).ConfigureAwait(false);
                await updater.StartWorkAsync(null, cancellationToken).ConfigureAwait(false);
                break;

            default:
                // Default echo: submit, work, emit the input text as an artifact, complete.
                if (isNew) await updater.SubmitAsync(cancellationToken).ConfigureAwait(false);
                await updater.StartWorkAsync(null, cancellationToken).ConfigureAwait(false);
                await updater.AddArtifactAsync([Part.FromText(text)],
                    cancellationToken: cancellationToken).ConfigureAwait(false);
                await updater.CompleteAsync(null, cancellationToken).ConfigureAwait(false);
                break;
        }
    }

    public async Task CancelAsync(RequestContext context, AgentEventQueue queue, CancellationToken cancellationToken)
    {
        var updater = new TaskUpdater(queue, context.TaskId, context.ContextId);
        await updater.CancelAsync(cancellationToken).ConfigureAwait(false);
    }
}
