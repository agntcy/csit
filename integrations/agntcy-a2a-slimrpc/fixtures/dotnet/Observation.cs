// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

using System.Text;
using A2A;

namespace Csit;

/// <summary>
/// Parseable view of a SendMessage result, mirroring the Go probe's observation struct.
/// Emitted as CSIT_SLIM_* KEY=value lines consumed by matrix_test.go.
/// </summary>
internal sealed class Observation
{
    public string Kind { get; set; } = "unknown"; // "task" | "message" | "unknown"
    public string State { get; set; } = "";        // TASK_STATE_* token; empty for a bare message
    public bool ArtifactPresent { get; set; }
    public string Text { get; set; } = "";
    public int StreamEvents { get; set; }

    public static Observation FromResponse(SendMessageResponse resp) => resp.PayloadCase switch
    {
        SendMessageResponseCase.Message => new Observation { Kind = "message", Text = MessageText(resp.Message) },
        SendMessageResponseCase.Task => FromTask(resp.Task!),
        _ => new Observation { Kind = "unknown" },
    };

    public static Observation FromTask(AgentTask task)
    {
        var (text, present) = TaskArtifactText(task);
        return new Observation
        {
            Kind = "task",
            State = StateToken(task.Status?.State),
            ArtifactPresent = present,
            Text = text,
        };
    }

    /// <summary>Fold one streamed event into this observation (matches the Go runStreaming aggregation).</summary>
    public void Absorb(StreamResponse ev)
    {
        switch (ev.PayloadCase)
        {
            case StreamResponseCase.Task when ev.Task is not null:
                State = StateToken(ev.Task.Status?.State);
                var (t, present) = TaskArtifactText(ev.Task);
                if (present) { Text += t; ArtifactPresent = true; }
                break;
            case StreamResponseCase.StatusUpdate when ev.StatusUpdate is not null:
                State = StateToken(ev.StatusUpdate.Status?.State);
                break;
            case StreamResponseCase.ArtifactUpdate when ev.ArtifactUpdate?.Artifact is not null:
                foreach (var part in ev.ArtifactUpdate.Artifact.Parts ?? [])
                    if (part.ContentCase == PartContentCase.Text) { Text += part.Text; ArtifactPresent = true; }
                break;
            case StreamResponseCase.Message when ev.Message is not null:
                Kind = "message";
                Text += MessageText(ev.Message);
                break;
        }
    }

    private static string MessageText(Message? msg)
    {
        if (msg?.Parts is null) return "";
        var sb = new StringBuilder();
        foreach (var part in msg.Parts)
            if (part.ContentCase == PartContentCase.Text) sb.Append(part.Text);
        return sb.ToString();
    }

    private static (string Text, bool Present) TaskArtifactText(AgentTask task)
    {
        var sb = new StringBuilder();
        bool present = false;
        foreach (var artifact in task.Artifacts ?? [])
            foreach (var part in artifact.Parts ?? [])
                if (part.ContentCase == PartContentCase.Text) { sb.Append(part.Text); present = true; }
        return (sb.ToString(), present);
    }

    /// <summary>
    /// Map the .NET TaskState enum to the canonical proto token a2a-go/python emit
    /// (e.g. InputRequired -> TASK_STATE_INPUT_REQUIRED), so the harness assertions match.
    /// </summary>
    private static string StateToken(TaskState? state)
    {
        if (state is null || state == TaskState.Unspecified) return "";
        var sb = new StringBuilder("TASK_STATE_");
        var name = state.ToString()!; // PascalCase, e.g. "InputRequired"
        for (int i = 0; i < name.Length; i++)
        {
            if (i > 0 && char.IsUpper(name[i])) sb.Append('_');
            sb.Append(char.ToUpperInvariant(name[i]));
        }
        return sb.ToString();
    }
}
