// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

using A2A;
using A2A.AspNetCore;
using AgentTaskStatus = A2A.TaskStatus;
using System.Text;
using System.Text.Json;
using System.Text.Json.Nodes;

namespace InteropServer;

internal static class Program
{
    private static string? RewriteReturnImmediatelyCompat(string body)
    {
        var root = JsonNode.Parse(body) as JsonObject;
        var changed = false;

        if (root?["method"]?.GetValue<string>() == "ListTaskPushNotificationConfigs")
        {
            root["method"] = "ListTaskPushNotificationConfig";
            changed = true;
        }

        var configuration = root?["params"]?["configuration"] as JsonObject;

        if (configuration is null)
        {
            return changed ? root?.ToJsonString() : null;
        }

        JsonNode? returnImmediatelyNode = null;
        if (configuration.TryGetPropertyValue("returnImmediately", out var camelValue))
        {
            returnImmediatelyNode = camelValue;
            configuration.Remove("returnImmediately");
        }
        else if (configuration.TryGetPropertyValue("return_immediately", out var snakeValue))
        {
            returnImmediatelyNode = snakeValue;
            configuration.Remove("return_immediately");
        }

        if (returnImmediatelyNode is null)
        {
            return changed ? root?.ToJsonString() : null;
        }

        var returnImmediately = returnImmediatelyNode.GetValue<bool>();
        configuration["blocking"] = !returnImmediately;
        changed = true;
        return root!.ToJsonString();
    }

    public static void Main(string[] args)
    {
        var options = ServerOptions.Parse(args);

        if (!string.Equals(options.Protocol, "jsonrpc", StringComparison.OrdinalIgnoreCase))
        {
            throw new ArgumentException($"unsupported protocol: {options.Protocol}");
        }

        var baseUrl = $"http://127.0.0.1:{options.Port}";
        var builder = WebApplication.CreateBuilder(args);
        builder.WebHost.UseUrls(baseUrl);

        var agentCard = InteropAgent.BuildAgentCard(baseUrl);
        builder.Services.AddA2AAgent<InteropAgent>(agentCard);

        var app = builder.Build();
        app.Use(async (context, next) =>
        {
            if (!HttpMethods.IsPost(context.Request.Method) || context.Request.Path != "/rpc")
            {
                await next(context);
                return;
            }

            context.Request.EnableBuffering();
            using var reader = new StreamReader(context.Request.Body, Encoding.UTF8, leaveOpen: true);
            var body = await reader.ReadToEndAsync();
            context.Request.Body.Position = 0;

            var rewritten = RewriteReturnImmediatelyCompat(body);
            if (rewritten is null)
            {
                await next(context);
                return;
            }

            var bytes = Encoding.UTF8.GetBytes(rewritten);
            context.Request.Body = new MemoryStream(bytes);
            context.Request.ContentLength = bytes.Length;

            await next(context);
        });
        app.MapGet("/health", () => Results.Ok(new { status = "ok" }));
        app.MapA2A("/rpc");
        app.MapWellKnownAgentCard(agentCard);
        app.Run();
    }
}

internal sealed class InteropAgent : IAgentHandler
{
    private const string PendingRequestText = "pending";

    public Task ExecuteAsync(RequestContext context, AgentEventQueue eventQueue, CancellationToken cancellationToken)
    {
        var responseText = $"dotnet server received: {context.UserText ?? string.Empty}";
        var state = string.Equals(context.UserText, PendingRequestText, StringComparison.Ordinal)
            ? TaskState.Working
            : TaskState.Completed;

        var task = new AgentTask
        {
            Id = context.TaskId,
            ContextId = context.ContextId,
            History = [context.Message],
            Status = new AgentTaskStatus
            {
                State = state,
                Timestamp = DateTimeOffset.UtcNow,
                Message = BuildStatusMessage(context, responseText),
            },
        };

        return eventQueue.EnqueueTaskAsync(task, cancellationToken).AsTask();
    }

    public Task CancelAsync(RequestContext context, AgentEventQueue eventQueue, CancellationToken cancellationToken)
    {
        var update = new TaskStatusUpdateEvent
        {
            TaskId = context.TaskId,
            ContextId = context.ContextId,
            Status = new AgentTaskStatus
            {
                State = TaskState.Canceled,
                Timestamp = DateTimeOffset.UtcNow,
                Message = BuildStatusMessage(context, "dotnet server canceled task"),
            },
        };

        return eventQueue.EnqueueStatusUpdateAsync(update, cancellationToken).AsTask();
    }

    public static AgentCard BuildAgentCard(string baseUrl) =>
        new()
        {
            Name = "CSIT DotNet JSON-RPC Agent",
            Description = "DotNet interoperability fixture for CSIT",
            Version = "1.0.0-preview",
            SupportedInterfaces =
            [
                new AgentInterface
                {
                    Url = $"{baseUrl}/rpc",
                    ProtocolBinding = "JSONRPC",
                    ProtocolVersion = "1.0",
                },
            ],
            Capabilities = new AgentCapabilities
            {
                Streaming = true,
                PushNotifications = false,
            },
            DefaultInputModes = ["text/plain"],
            DefaultOutputModes = ["text/plain"],
            Skills = [],
        };

    private static Message BuildStatusMessage(RequestContext context, string text) =>
        new()
        {
            Role = Role.Agent,
            MessageId = Guid.NewGuid().ToString("N"),
            ContextId = context.ContextId,
            TaskId = context.TaskId,
            Parts = [Part.FromText(text)],
        };
}

internal sealed class ServerOptions
{
    public required int Port { get; init; }

    public required string Protocol { get; init; }

    public static ServerOptions Parse(string[] args)
    {
        var port = 19093;
        var protocol = "jsonrpc";

        for (var index = 0; index < args.Length; index++)
        {
            switch (args[index])
            {
                case "--port":
                    index++;
                    port = int.Parse(args[index], System.Globalization.CultureInfo.InvariantCulture);
                    break;
                case "--protocol":
                    index++;
                    protocol = args[index];
                    break;
                default:
                    break;
            }
        }

        return new ServerOptions { Port = port, Protocol = protocol };
    }
}