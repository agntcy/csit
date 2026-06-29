// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

namespace Csit;

/// <summary>
/// Scenario sentinels: the probe sends one of these as the request text to drive a
/// non-echo server response. These travel over the wire, so they MUST match the
/// sentinels in fixtures/go (cmd/server, cmd/probe) and fixtures/python byte-for-byte.
/// </summary>
internal static class Scenarios
{
    public const string Echo = "echo";

    public const string SentinelMessageOnly = "csit-scenario:message-only";
    public const string SentinelTaskFailure = "csit-scenario:task-failure";
    public const string SentinelInputRequired = "csit-scenario:input-required";
    public const string SentinelStreaming = "csit-scenario:streaming";
    public const string SentinelCancel = "csit-scenario:cancel";
    public const string SentinelMultiTurn = "csit-scenario:multi-turn";
    public const string SentinelMultiTurnContinue = "csit-scenario:multi-turn-continue";

    // Artifact emitted on the multi-turn continuation turn. Must match the Go/Python
    // fixtures and the harness's multiTurnCompleteMarker.
    public const string MultiTurnCompleteText = "multi-turn complete";

    // Bare-message response text for the message-only scenario (kind=message; the text
    // itself is not asserted, only the result kind).
    public const string MessageOnlyText = "dotnet server message-only response";

    // Probe transport modes selected per scenario.
    public const string ModeUnary = "unary";
    public const string ModeStreaming = "streaming";
    public const string ModeCancel = "cancel";
    public const string ModeMultiTurn = "multi-turn";

    /// <summary>
    /// Map a scenario selector to (outbound request text, whether the response must echo
    /// it, transport mode). Non-echo scenarios send a fixed sentinel and only emit the
    /// observation block (asserted by the harness).
    /// </summary>
    public static (string Want, bool EnforceEcho, string Mode) Request(string scenario, string text) =>
        scenario switch
        {
            Echo or "" => (text, true, ModeUnary),
            "message-only" => (SentinelMessageOnly, false, ModeUnary),
            "task-failure" => (SentinelTaskFailure, false, ModeUnary),
            "input-required" => (SentinelInputRequired, false, ModeUnary),
            "streaming" => (SentinelStreaming, false, ModeStreaming),
            "task-cancel" => (SentinelCancel, false, ModeCancel),
            "multi-turn" => (SentinelMultiTurn, false, ModeMultiTurn),
            _ => throw new ArgumentException($"unknown scenario {scenario}"),
        };
}
