// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

using A2A;
using A2A.AspNetCore;
using AgentTaskStatus = A2A.TaskStatus;
using Microsoft.Extensions.DependencyInjection;
using System.Text;
using System.Text.Json;
using System.Text.Json.Nodes;
using System.Text.Json.Serialization;

namespace InteropServer;

internal static class Program
{
    private static readonly JsonSerializerOptions RestCompatJsonOptions = CreateRestCompatJsonOptions();

    private static bool UsesJsonRpc(string protocol) =>
        string.Equals(protocol, "jsonrpc", StringComparison.OrdinalIgnoreCase);

    private static bool UsesRest(string protocol) =>
        string.Equals(protocol, "rest", StringComparison.OrdinalIgnoreCase);

    private static JsonSerializerOptions CreateRestCompatJsonOptions()
    {
        var options = new JsonSerializerOptions(A2AJsonUtilities.DefaultOptions)
        {
            DefaultIgnoreCondition = JsonIgnoreCondition.WhenWritingNull,
        };

        return options;
    }

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

        if (!UsesJsonRpc(options.Protocol) && !UsesRest(options.Protocol))
        {
            throw new ArgumentException($"unsupported protocol: {options.Protocol}");
        }

        var baseUrl = $"http://127.0.0.1:{options.Port}";
        var builder = WebApplication.CreateBuilder(args);
        builder.WebHost.UseUrls(baseUrl);

        var agentCard = InteropAgent.BuildAgentCard(baseUrl, options.Protocol);
        builder.Services.AddA2AAgent<InteropAgent>(agentCard);

        var app = builder.Build();
        if (UsesJsonRpc(options.Protocol))
        {
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
        }

        app.MapGet("/health", () => Results.Ok(new { status = "ok" }));

        if (UsesJsonRpc(options.Protocol))
        {
            app.MapA2A("/rpc");
        }
        else
        {
            var requestHandler = app.Services.GetRequiredService<IA2ARequestHandler>();

            app.MapGet("/rest/card", () => Results.Ok(agentCard));
            app.MapPost("/rest/message:send", (SendMessageRequest request, CancellationToken ct) =>
                HandleRestAsync(() => requestHandler.SendMessageAsync(request, ct)));
            app.MapPost("/rest/message:stream", (HttpContext context, SendMessageRequest request, CancellationToken ct) =>
                StreamRestAsync(context, requestHandler.SendStreamingMessageAsync(request, ct), ct));
            app.MapGet("/rest/tasks/{id}", (string id, int? historyLength, CancellationToken ct) =>
                HandleRestAsync(() => requestHandler.GetTaskAsync(new GetTaskRequest
                {
                    Id = id,
                    HistoryLength = historyLength,
                }, ct)));
            app.MapGet("/rest/tasks", (string? contextId, TaskState? status, int? pageSize, string? pageToken, int? historyLength, CancellationToken ct) =>
                HandleRestAsync(() => requestHandler.ListTasksAsync(new ListTasksRequest
                {
                    ContextId = contextId,
                    Status = status,
                    PageSize = pageSize,
                    PageToken = pageToken,
                    HistoryLength = historyLength,
                }, ct)));
            app.MapPost("/rest/tasks/{id}:cancel", (string id, CancellationToken ct) =>
                HandleRestAsync(() => requestHandler.CancelTaskAsync(new CancelTaskRequest
                {
                    Id = id,
                }, ct)));
            app.MapPost("/rest/tasks/{id}/pushNotificationConfigs", (string id, PushNotificationConfig config, CancellationToken ct) =>
                HandleRestAsync(() => requestHandler.CreateTaskPushNotificationConfigAsync(new CreateTaskPushNotificationConfigRequest
                {
                    TaskId = id,
                    ConfigId = config.Id ?? string.Empty,
                    Config = config,
                }, ct)));
            app.MapGet("/rest/tasks/{id}/pushNotificationConfigs", (string id, int? pageSize, string? pageToken, CancellationToken ct) =>
                HandleRestAsync(() => requestHandler.ListTaskPushNotificationConfigAsync(new ListTaskPushNotificationConfigRequest
                {
                    TaskId = id,
                    PageSize = pageSize,
                    PageToken = pageToken,
                }, ct)));
            app.MapGet("/rest/tasks/{id}/pushNotificationConfigs/{configId}", (string id, string configId, CancellationToken ct) =>
                HandleRestAsync(() => requestHandler.GetTaskPushNotificationConfigAsync(new GetTaskPushNotificationConfigRequest
                {
                    TaskId = id,
                    Id = configId,
                }, ct)));
            app.MapDelete("/rest/tasks/{id}/pushNotificationConfigs/{configId}", (string id, string configId, CancellationToken ct) =>
                HandleRestNoContentAsync(() => requestHandler.DeleteTaskPushNotificationConfigAsync(new DeleteTaskPushNotificationConfigRequest
                {
                    TaskId = id,
                    Id = configId,
                }, ct)));
        }

        app.MapWellKnownAgentCard(agentCard);
        app.Run();
    }

    private static async Task<IResult> HandleRestAsync<T>(Func<Task<T>> action)
    {
        try
        {
            return CreateRestSuccessResult(await action().ConfigureAwait(false));
        }
        catch (A2AException error)
        {
            return CreateRestErrorResult(error);
        }
    }

    private static async Task<IResult> HandleRestNoContentAsync(Func<Task> action)
    {
        try
        {
            await action().ConfigureAwait(false);
            return Results.NoContent();
        }
        catch (A2AException error)
        {
            return CreateRestErrorResult(error);
        }
    }

    private static async Task StreamRestAsync(HttpContext context, IAsyncEnumerable<StreamResponse> stream, CancellationToken cancellationToken)
    {
        try
        {
            context.Response.StatusCode = StatusCodes.Status200OK;
            context.Response.ContentType = "text/event-stream";

            await foreach (var response in stream.WithCancellation(cancellationToken))
            {
                var payload = JsonSerializer.Serialize(response, RestCompatJsonOptions);
                await context.Response.WriteAsync($"data: {payload}\n\n", cancellationToken).ConfigureAwait(false);
                await context.Response.Body.FlushAsync(cancellationToken).ConfigureAwait(false);
            }
        }
        catch (A2AException error) when (!context.Response.HasStarted)
        {
            var restError = CreateRestError(error);
            context.Response.StatusCode = restError.Error.Code;
            context.Response.ContentType = "application/json";
            await JsonSerializer.SerializeAsync(context.Response.Body, restError, A2AJsonUtilities.DefaultOptions, cancellationToken).ConfigureAwait(false);
        }
    }

    private static IResult CreateRestSuccessResult<T>(T response)
    {
        return Results.Json(response, RestCompatJsonOptions);
    }

    private static IResult CreateRestErrorResult(A2AException error)
    {
        var restError = CreateRestError(error);
        return Results.Json(restError, RestCompatJsonOptions, statusCode: restError.Error.Code);
    }

    private static RestErrorResponse CreateRestError(A2AException error)
    {
        var statusCode = ErrorStatusCode(error.ErrorCode);
        return new RestErrorResponse
        {
            Error = new RestErrorStatus
            {
                Code = statusCode,
                Status = ErrorStatus(error.ErrorCode),
                Message = error.Message,
                Details =
                [
                    new RestErrorDetail
                    {
                        Type = "type.googleapis.com/google.rpc.ErrorInfo",
                        Reason = ErrorReason(error.ErrorCode),
                        Domain = "a2a-protocol.org",
                    },
                ],
            },
        };
    }

    private static int ErrorStatusCode(A2AErrorCode errorCode) => errorCode switch
    {
        A2AErrorCode.TaskNotFound => StatusCodes.Status404NotFound,
        A2AErrorCode.TaskNotCancelable => StatusCodes.Status409Conflict,
        A2AErrorCode.PushNotificationNotSupported => StatusCodes.Status501NotImplemented,
        A2AErrorCode.UnsupportedOperation => StatusCodes.Status501NotImplemented,
        A2AErrorCode.ContentTypeNotSupported => StatusCodes.Status415UnsupportedMediaType,
        A2AErrorCode.MethodNotFound => StatusCodes.Status404NotFound,
        A2AErrorCode.InvalidRequest => StatusCodes.Status400BadRequest,
        A2AErrorCode.InvalidParams => StatusCodes.Status400BadRequest,
        A2AErrorCode.ParseError => StatusCodes.Status400BadRequest,
        _ => StatusCodes.Status500InternalServerError,
    };

    private static string ErrorStatus(A2AErrorCode errorCode) => errorCode switch
    {
        A2AErrorCode.TaskNotFound => "NOT_FOUND",
        A2AErrorCode.TaskNotCancelable => "FAILED_PRECONDITION",
        A2AErrorCode.PushNotificationNotSupported => "UNIMPLEMENTED",
        A2AErrorCode.UnsupportedOperation => "UNIMPLEMENTED",
        A2AErrorCode.ContentTypeNotSupported => "UNSUPPORTED_MEDIA_TYPE",
        A2AErrorCode.MethodNotFound => "NOT_FOUND",
        A2AErrorCode.InvalidRequest => "INVALID_ARGUMENT",
        A2AErrorCode.InvalidParams => "INVALID_ARGUMENT",
        A2AErrorCode.ParseError => "INVALID_ARGUMENT",
        _ => "INTERNAL",
    };

    private static string ErrorReason(A2AErrorCode errorCode) => errorCode switch
    {
        A2AErrorCode.TaskNotFound => "TASK_NOT_FOUND",
        A2AErrorCode.TaskNotCancelable => "TASK_NOT_CANCELABLE",
        A2AErrorCode.PushNotificationNotSupported => "PUSH_NOTIFICATION_NOT_SUPPORTED",
        A2AErrorCode.UnsupportedOperation => "UNSUPPORTED_OPERATION",
        A2AErrorCode.ContentTypeNotSupported => "CONTENT_TYPE_NOT_SUPPORTED",
        A2AErrorCode.InvalidAgentResponse => "INVALID_AGENT_RESPONSE",
        A2AErrorCode.ExtendedAgentCardNotConfigured => "EXTENDED_AGENT_CARD_NOT_CONFIGURED",
        A2AErrorCode.ExtensionSupportRequired => "EXTENSION_SUPPORT_REQUIRED",
        A2AErrorCode.VersionNotSupported => "VERSION_NOT_SUPPORTED",
        A2AErrorCode.MethodNotFound => "METHOD_NOT_FOUND",
        A2AErrorCode.InvalidParams => "INVALID_PARAMS",
        A2AErrorCode.ParseError => "PARSE_ERROR",
        A2AErrorCode.InvalidRequest => "INVALID_REQUEST",
        _ => "INTERNAL_ERROR",
    };
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

    public static AgentCard BuildAgentCard(string baseUrl, string protocol)
    {
        var usesRest = string.Equals(protocol, "rest", StringComparison.OrdinalIgnoreCase);

        return new AgentCard
        {
            Name = usesRest ? "CSIT DotNet HTTP+JSON Agent" : "CSIT DotNet JSON-RPC Agent",
            Description = "DotNet interoperability fixture for CSIT",
            Version = "1.0.0-preview",
            SupportedInterfaces =
            [
                new AgentInterface
                {
                    Url = usesRest ? $"{baseUrl}/rest" : $"{baseUrl}/rpc",
                    ProtocolBinding = usesRest ? "HTTP+JSON" : "JSONRPC",
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
    }

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

internal sealed class RestErrorResponse
{
    [JsonPropertyName("error")]
    public required RestErrorStatus Error { get; init; }
}

internal sealed class RestErrorStatus
{
    [JsonPropertyName("code")]
    public required int Code { get; init; }

    [JsonPropertyName("status")]
    public required string Status { get; init; }

    [JsonPropertyName("message")]
    public required string Message { get; init; }

    [JsonPropertyName("details")]
    public required List<RestErrorDetail> Details { get; init; }
}

internal sealed class RestErrorDetail
{
    [JsonPropertyName("@type")]
    public required string Type { get; init; }

    [JsonPropertyName("reason")]
    public required string Reason { get; init; }

    [JsonPropertyName("domain")]
    public required string Domain { get; init; }
}