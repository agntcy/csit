using System.Text.Json;
using System.Text;
using A2A;

namespace InteropProbe;

internal static class Program
{
    private const string RequestText = "ping";
    private const string PendingRequestText = "pending";
    private const string RequestDataKind = "structured";
    private const string RequestDataScope = "interop";
    private const string RequestMetadataKey = "csit";
    private const string RequestMetadataValue = "multipart";

    public static async Task<int> Main(string[] args)
    {
        ProbeOptions options;

        try
        {
            options = ProbeOptions.Parse(args);
        }
        catch (Exception error)
        {
            Console.Error.WriteLine(error.Message);
            return 2;
        }

        try
        {
            await RunAsync(options);
            return 0;
        }
        catch (Exception error)
        {
            Console.Error.WriteLine(error.Message);
            return 1;
        }
    }

    private static async Task RunAsync(ProbeOptions options)
    {
        var resolver = new A2ACardResolver(new Uri(options.CardUrl));
        var card = await resolver.GetAgentCardAsync().ConfigureAwait(false);
        var client = CreateClient(card);

        var expectedPingText = ExpectedResponseText(options.ServerPrefix, RequestText);
        var expectedPendingText = ExpectedResponseText(options.ServerPrefix, PendingRequestText);
        var expectedCancelText = ExpectedCancelText(options.ServerPrefix);

        var request = BuildRequest(RequestText, false);

        var completedTask = TaskFromResponse(
            await SendMessageAsync(client, card, request, false).ConfigureAwait(false),
            "unary");
        AssertState(completedTask.Status.State, TaskState.Completed, "unary");
        AssertText(TaskText(completedTask), expectedPingText, "unary");
        AssertTaskHistory(completedTask, RequestText, "unary");

        var fetchedTask = await client.GetTaskAsync(new GetTaskRequest
        {
            Id = completedTask.Id,
            HistoryLength = 1,
        }).ConfigureAwait(false);
        AssertState(fetchedTask.Status.State, TaskState.Completed, "get_task");
        AssertText(TaskText(fetchedTask), expectedPingText, "get_task");
        AssertTaskHistory(fetchedTask, RequestText, "get_task");

        var listedTasks = await ListTasksAsync(card, new ListTasksRequest
        {
            ContextId = completedTask.ContextId,
        }).ConfigureAwait(false);
        if (!listedTasks.Any(task => task.Id == completedTask.Id))
        {
            throw new InvalidOperationException($"list_tasks did not include expected task {completedTask.Id}");
        }

        var streamingText = await ReadStreamingTextAsync(client.SendStreamingMessageAsync(request)).ConfigureAwait(false);
        AssertText(streamingText, expectedPingText, "streaming");

        var pendingTask = TaskFromResponse(
            await SendMessageAsync(client, card, BuildRequest(PendingRequestText, true), true).ConfigureAwait(false),
            "pending unary");
        AssertState(pendingTask.Status.State, TaskState.Working, "pending unary");
        AssertText(TaskText(pendingTask), expectedPendingText, "pending unary");

        var canceledTask = await client.CancelTaskAsync(new CancelTaskRequest
        {
            Id = pendingTask.Id,
        }).ConfigureAwait(false);
        AssertState(canceledTask.Status.State, TaskState.Canceled, "cancel_task");
        AssertText(TaskText(canceledTask), expectedCancelText, "cancel_task");

        var fetchedCanceledTask = await client.GetTaskAsync(new GetTaskRequest
        {
            Id = pendingTask.Id,
        }).ConfigureAwait(false);
        AssertState(fetchedCanceledTask.Status.State, TaskState.Canceled, "get_task after cancel");
        AssertText(TaskText(fetchedCanceledTask), expectedCancelText, "get_task after cancel");

        if (options.RelaxedErrorChecks)
        {
            await ExpectFailureAsync(() => client.GetTaskAsync(new GetTaskRequest { Id = Guid.NewGuid().ToString("N") }), "get missing task").ConfigureAwait(false);
            await ExpectFailureAsync(() => client.CancelTaskAsync(new CancelTaskRequest { Id = completedTask.Id }), "cancel completed task").ConfigureAwait(false);
        }
        else
        {
            await ExpectA2AErrorAsync(() => client.GetTaskAsync(new GetTaskRequest { Id = Guid.NewGuid().ToString("N") }), A2AErrorCode.TaskNotFound, "get missing task").ConfigureAwait(false);
            await ExpectA2AErrorAsync(() => client.CancelTaskAsync(new CancelTaskRequest { Id = completedTask.Id }), A2AErrorCode.TaskNotCancelable, "cancel completed task").ConfigureAwait(false);
        }

        if (options.ExpectPushUnsupported)
        {
            var pushConfig = new PushNotificationConfig
            {
                Id = "interop-config",
                Url = "https://example.invalid/webhook",
            };

            if (options.RelaxedErrorChecks)
            {
                await ExpectFailureAsync(() => client.CreateTaskPushNotificationConfigAsync(new CreateTaskPushNotificationConfigRequest
                {
                    TaskId = completedTask.Id,
                    ConfigId = "interop-config",
                    Config = pushConfig,
                }), "create_push_config").ConfigureAwait(false);
                await ExpectFailureAsync(() => client.GetTaskPushNotificationConfigAsync(new GetTaskPushNotificationConfigRequest
                {
                    TaskId = completedTask.Id,
                    Id = "interop-config",
                }), "get_push_config").ConfigureAwait(false);
                await ExpectFailureAsync(() => client.ListTaskPushNotificationConfigAsync(new ListTaskPushNotificationConfigRequest
                {
                    TaskId = completedTask.Id,
                }), "list_push_configs").ConfigureAwait(false);
                await ExpectFailureAsync(() => client.DeleteTaskPushNotificationConfigAsync(new DeleteTaskPushNotificationConfigRequest
                {
                    TaskId = completedTask.Id,
                    Id = "interop-config",
                }), "delete_push_config").ConfigureAwait(false);
            }
            else
            {
                await ExpectA2AErrorAsync(() => client.CreateTaskPushNotificationConfigAsync(new CreateTaskPushNotificationConfigRequest
                {
                    TaskId = completedTask.Id,
                    ConfigId = "interop-config",
                    Config = pushConfig,
                }), options.ExpectedPushErrorCode, "create_push_config").ConfigureAwait(false);
                await ExpectA2AErrorAsync(() => client.GetTaskPushNotificationConfigAsync(new GetTaskPushNotificationConfigRequest
                {
                    TaskId = completedTask.Id,
                    Id = "interop-config",
                }), options.ExpectedPushErrorCode, "get_push_config").ConfigureAwait(false);
                await ExpectA2AErrorAsync(() => client.ListTaskPushNotificationConfigAsync(new ListTaskPushNotificationConfigRequest
                {
                    TaskId = completedTask.Id,
                }), options.ExpectedPushErrorCode, "list_push_configs").ConfigureAwait(false);
                await ExpectA2AErrorAsync(() => client.DeleteTaskPushNotificationConfigAsync(new DeleteTaskPushNotificationConfigRequest
                {
                    TaskId = completedTask.Id,
                    Id = "interop-config",
                }), options.ExpectedPushErrorCode, "delete_push_config").ConfigureAwait(false);
            }
        }
        else if (options.ExpectPushSupported)
        {
            var pushConfig = new PushNotificationConfig
            {
                Id = "interop-config",
                Url = "https://example.invalid/webhook",
                Token = "interop-token",
                Authentication = new AuthenticationInfo
                {
                    Scheme = "Bearer",
                    Credentials = "interop-credential",
                },
            };

            var createdConfig = await CreateTaskPushNotificationConfigAsync(card, new CreateTaskPushNotificationConfigRequest
            {
                TaskId = completedTask.Id,
                ConfigId = "interop-config",
                Config = pushConfig,
            }).ConfigureAwait(false);
            AssertPushConfig(createdConfig, completedTask.Id, pushConfig, "create_push_config");

            var fetchedConfig = await GetTaskPushNotificationConfigAsync(card, new GetTaskPushNotificationConfigRequest
            {
                TaskId = completedTask.Id,
                Id = "interop-config",
            }).ConfigureAwait(false);
            AssertPushConfig(fetchedConfig, completedTask.Id, pushConfig, "get_push_config");

            var listedConfigs = await ListTaskPushNotificationConfigAsync(card, new ListTaskPushNotificationConfigRequest
            {
                TaskId = completedTask.Id,
            }).ConfigureAwait(false);
            if (listedConfigs.Count != 1)
            {
                throw new InvalidOperationException($"unexpected list_push_configs result count: got {listedConfigs.Count}, want 1");
            }
            AssertPushConfig(listedConfigs[0], completedTask.Id, pushConfig, "list_push_configs");

            await DeleteTaskPushNotificationConfigAsync(card, new DeleteTaskPushNotificationConfigRequest
            {
                TaskId = completedTask.Id,
                Id = "interop-config",
            }).ConfigureAwait(false);

            var listedAfterDelete = await ListTaskPushNotificationConfigAsync(card, new ListTaskPushNotificationConfigRequest
            {
                TaskId = completedTask.Id,
            }).ConfigureAwait(false);
            if (listedAfterDelete.Count > 0)
            {
                throw new InvalidOperationException($"expected list_push_configs after delete to be empty, got {listedAfterDelete.Count}");
            }
        }

        var protocol = card.SupportedInterfaces.FirstOrDefault()?.ProtocolBinding ?? "unknown";
        Console.WriteLine($"validated {options.ServerPrefix} {protocol} lifecycle against {options.CardUrl}");
    }

    private static SendMessageRequest BuildRequest(string text, bool returnImmediately)
    {
        return new SendMessageRequest
        {
            Message = new Message
            {
                Role = Role.User,
                MessageId = Guid.NewGuid().ToString("N"),
                Parts =
                [
                    Part.FromText(text),
                    Part.FromData(JsonSerializer.SerializeToElement(new { kind = RequestDataKind, scope = RequestDataScope })),
                ],
                Metadata = new Dictionary<string, JsonElement>
                {
                    [RequestMetadataKey] = JsonSerializer.SerializeToElement(RequestMetadataValue),
                },
            },
            Configuration = returnImmediately
                ? new SendMessageConfiguration { Blocking = false }
                : null,
        };
    }

    private static async Task<SendMessageResponse> SendMessageAsync(
        IA2AClient client,
        AgentCard card,
        SendMessageRequest request,
        bool returnImmediately)
    {
        if (!returnImmediately)
        {
            return await client.SendMessageAsync(request).ConfigureAwait(false);
        }

        var payload = new
        {
            message = request.Message,
            configuration = new
            {
                returnImmediately = true,
            },
        };

        return await SendJsonRpcAsync<SendMessageResponse>(card, "SendMessage", payload).ConfigureAwait(false);
    }

    private static async Task<List<AgentTask>> ListTasksAsync(AgentCard card, ListTasksRequest request)
    {
        var response = await SendJsonRpcAsync<CompatibleListTasksResponse>(card, "ListTasks", request).ConfigureAwait(false);
        return response.Tasks;
    }

    private static Task<CompatibleTaskPushNotificationConfig> CreateTaskPushNotificationConfigAsync(
        AgentCard card,
        CreateTaskPushNotificationConfigRequest request)
    {
        return SendJsonRpcAsync<CompatibleTaskPushNotificationConfig>(card, "CreateTaskPushNotificationConfig", request);
    }

    private static Task<CompatibleTaskPushNotificationConfig> GetTaskPushNotificationConfigAsync(
        AgentCard card,
        GetTaskPushNotificationConfigRequest request)
    {
        return SendJsonRpcAsync<CompatibleTaskPushNotificationConfig>(card, "GetTaskPushNotificationConfig", request);
    }

    private static async Task<List<CompatibleTaskPushNotificationConfig>> ListTaskPushNotificationConfigAsync(
        AgentCard card,
        ListTaskPushNotificationConfigRequest request)
    {
        var response = await SendJsonRpcAsync<JsonElement>(
            card,
            "ListTaskPushNotificationConfigs",
            request).ConfigureAwait(false);

        return response.ValueKind switch
        {
            JsonValueKind.Array => JsonSerializer.Deserialize<List<CompatibleTaskPushNotificationConfig>>(
                response.GetRawText(),
                A2AJsonUtilities.DefaultOptions) ?? [],
            JsonValueKind.Object => JsonSerializer.Deserialize<CompatibleListTaskPushNotificationConfigResponse>(
                response.GetRawText(),
                A2AJsonUtilities.DefaultOptions)?.Configs ?? [],
            _ => throw new InvalidOperationException("unexpected list push config JSON-RPC result shape"),
        };
    }

    private static async Task DeleteTaskPushNotificationConfigAsync(
        AgentCard card,
        DeleteTaskPushNotificationConfigRequest request)
    {
        await SendJsonRpcWithoutResultAsync(card, "DeleteTaskPushNotificationConfig", request).ConfigureAwait(false);
    }

    private static IA2AClient CreateClient(AgentCard card)
    {
        var jsonRpcInterface = GetJsonRpcInterface(card);

        return new A2AClient(new Uri(jsonRpcInterface.Url));
    }

    private static AgentInterface GetJsonRpcInterface(AgentCard card)
    {
        var jsonRpcInterface = card.SupportedInterfaces.FirstOrDefault(candidate =>
            string.Equals(candidate.ProtocolBinding, "JSONRPC", StringComparison.OrdinalIgnoreCase));

        if (jsonRpcInterface is null)
        {
            throw new InvalidOperationException("agent card did not advertise a JSON-RPC interface");
        }

        return jsonRpcInterface;
    }

    private static async Task<TResult> SendJsonRpcAsync<TResult>(AgentCard card, string method, object payload)
    {
        using var httpClient = new HttpClient();
        var request = new JsonRpcRequest
        {
            Id = Guid.NewGuid().ToString("N"),
            Method = method,
            Params = JsonSerializer.SerializeToElement(payload, A2AJsonUtilities.DefaultOptions),
        };

        using var message = new HttpRequestMessage(HttpMethod.Post, GetJsonRpcInterface(card).Url)
        {
            Content = new StringContent(
                JsonSerializer.Serialize(request, A2AJsonUtilities.DefaultOptions),
                Encoding.UTF8,
                "application/json"),
        };
        message.Headers.TryAddWithoutValidation("A2A-Version", "1.0");

        using var response = await httpClient.SendAsync(message).ConfigureAwait(false);
        response.EnsureSuccessStatusCode();

        await using var stream = await response.Content.ReadAsStreamAsync().ConfigureAwait(false);
        var rpcResponse = await JsonSerializer.DeserializeAsync<JsonRpcResponse>(
            stream,
            A2AJsonUtilities.DefaultOptions).ConfigureAwait(false)
            ?? throw new InvalidOperationException("failed to deserialize JSON-RPC response");

        if (rpcResponse.Error is not null)
        {
            throw new A2AException(rpcResponse.Error.Message, (A2AErrorCode)rpcResponse.Error.Code);
        }

        if (rpcResponse.Result is null)
        {
            throw new InvalidOperationException($"failed to deserialize JSON-RPC result for {method}: null result payload");
        }

        var rawResult = rpcResponse.Result.ToJsonString();

        var result = JsonSerializer.Deserialize<TResult>(
            rawResult,
            A2AJsonUtilities.DefaultOptions);

        return result ?? throw new InvalidOperationException($"failed to deserialize JSON-RPC result for {method}: {rawResult}");
    }

    private static async Task SendJsonRpcWithoutResultAsync(AgentCard card, string method, object payload)
    {
        using var httpClient = new HttpClient();
        var request = new JsonRpcRequest
        {
            Id = Guid.NewGuid().ToString("N"),
            Method = method,
            Params = JsonSerializer.SerializeToElement(payload, A2AJsonUtilities.DefaultOptions),
        };

        using var message = new HttpRequestMessage(HttpMethod.Post, GetJsonRpcInterface(card).Url)
        {
            Content = new StringContent(
                JsonSerializer.Serialize(request, A2AJsonUtilities.DefaultOptions),
                Encoding.UTF8,
                "application/json"),
        };
        message.Headers.TryAddWithoutValidation("A2A-Version", "1.0");

        using var response = await httpClient.SendAsync(message).ConfigureAwait(false);
        response.EnsureSuccessStatusCode();

        await using var stream = await response.Content.ReadAsStreamAsync().ConfigureAwait(false);
        var rpcResponse = await JsonSerializer.DeserializeAsync<JsonRpcResponse>(
            stream,
            A2AJsonUtilities.DefaultOptions).ConfigureAwait(false)
            ?? throw new InvalidOperationException($"failed to deserialize JSON-RPC response for {method}");

        if (rpcResponse.Error is not null)
        {
            throw new A2AException(rpcResponse.Error.Message, (A2AErrorCode)rpcResponse.Error.Code);
        }
    }

    private static string ExpectedResponseText(string serverPrefix, string requestText) =>
        $"{serverPrefix} server received: {requestText}";

    private static string ExpectedCancelText(string serverPrefix) =>
        $"{serverPrefix} server canceled task";

    private static AgentTask TaskFromResponse(SendMessageResponse response, string kind)
    {
        return response.Task ?? throw new InvalidOperationException($"unexpected {kind} response type: Message");
    }

    private static string TaskText(AgentTask task)
    {
        return task.Status.Message is null
            ? throw new InvalidOperationException("task response contained no message")
            : FirstText(task.Status.Message);
    }

    private static string FirstText(Message message)
    {
        var part = message.Parts.FirstOrDefault(value => value.Text is not null);
        return part?.Text ?? throw new InvalidOperationException("message contained no text parts");
    }

    private static void AssertText(string actual, string expected, string kind)
    {
        if (!string.Equals(actual, expected, StringComparison.Ordinal))
        {
            throw new InvalidOperationException($"unexpected {kind} response text: got '{actual}', want '{expected}'");
        }
    }

    private static void AssertState(TaskState actual, TaskState expected, string kind)
    {
        if (actual != expected)
        {
            throw new InvalidOperationException($"unexpected {kind} task state: got {actual}, want {expected}");
        }
    }

    private static void AssertTaskHistory(AgentTask task, string expectedText, string kind)
    {
        if (task.History is null || task.History.Count != 1)
        {
            throw new InvalidOperationException($"{kind} task did not include a single history entry");
        }

        var message = task.History[0];
        AssertText(FirstText(message), expectedText, kind);

        if (message.Parts.Count != 2)
        {
            throw new InvalidOperationException($"{kind} task history had {message.Parts.Count} parts, want 2");
        }

        var dataPart = message.Parts[1].Data ?? throw new InvalidOperationException($"{kind} task history second part was not a structured data part");
        var kindValue = dataPart.GetProperty("kind").GetString();
        var scopeValue = dataPart.GetProperty("scope").GetString();
        if (!string.Equals(kindValue, RequestDataKind, StringComparison.Ordinal) || !string.Equals(scopeValue, RequestDataScope, StringComparison.Ordinal))
        {
            throw new InvalidOperationException($"{kind} task history data part mismatch: got kind={kindValue} scope={scopeValue}");
        }

        if (message.Metadata is null || !message.Metadata.TryGetValue(RequestMetadataKey, out var metadataValue))
        {
            throw new InvalidOperationException($"{kind} task history metadata was missing {RequestMetadataKey}");
        }

        if (!string.Equals(metadataValue.GetString(), RequestMetadataValue, StringComparison.Ordinal))
        {
            throw new InvalidOperationException($"{kind} task history metadata mismatch: got '{metadataValue.GetString()}', want '{RequestMetadataValue}'");
        }
    }

    private static async Task<string> ReadStreamingTextAsync(IAsyncEnumerable<StreamResponse> stream)
    {
        await foreach (var response in stream.ConfigureAwait(false))
        {
            var text = StreamResponseText(response);
            if (text is not null)
            {
                return text;
            }
        }

        throw new InvalidOperationException("stream completed without a terminal response event");
    }

    private static string? StreamResponseText(StreamResponse response)
    {
        return response.PayloadCase switch
        {
            StreamResponseCase.Message => FirstText(response.Message!),
            StreamResponseCase.Task => TaskText(response.Task!),
            StreamResponseCase.StatusUpdate when response.StatusUpdate?.Status.Message is not null => FirstText(response.StatusUpdate.Status.Message),
            _ => null,
        };
    }

    private static void AssertPushConfig(CompatibleTaskPushNotificationConfig actual, string taskId, PushNotificationConfig expected, string kind)
    {
        if (!string.Equals(actual.TaskId, taskId, StringComparison.Ordinal))
        {
            throw new InvalidOperationException($"unexpected {kind} task id: got '{actual.TaskId}', want '{taskId}'");
        }

        if (!string.Equals(actual.Config.Id, expected.Id, StringComparison.Ordinal)
            || !string.Equals(actual.Config.Url, expected.Url, StringComparison.Ordinal)
            || !string.Equals(actual.Config.Token, expected.Token, StringComparison.Ordinal)
            || !string.Equals(actual.Config.Authentication?.Scheme, expected.Authentication?.Scheme, StringComparison.Ordinal)
            || !string.Equals(actual.Config.Authentication?.Credentials, expected.Authentication?.Credentials, StringComparison.Ordinal))
        {
            throw new InvalidOperationException($"unexpected {kind} push config result");
        }
    }

    private static async Task ExpectFailureAsync(Func<Task> action, string kind)
    {
        try
        {
            await action().ConfigureAwait(false);
        }
        catch
        {
            return;
        }

        throw new InvalidOperationException($"expected {kind} to fail, but it succeeded");
    }

    private static async Task ExpectFailureAsync<T>(Func<Task<T>> action, string kind)
    {
        try
        {
            _ = await action().ConfigureAwait(false);
        }
        catch
        {
            return;
        }

        throw new InvalidOperationException($"expected {kind} to fail, but it succeeded");
    }

    private static async Task ExpectA2AErrorAsync(Func<Task> action, A2AErrorCode expectedCode, string kind)
    {
        try
        {
            await action().ConfigureAwait(false);
        }
        catch (A2AException error) when (error.ErrorCode == expectedCode)
        {
            return;
        }
        catch (A2AException error)
        {
            throw new InvalidOperationException($"unexpected {kind} error code: got {(int)error.ErrorCode}, want {(int)expectedCode} ({error.Message})");
        }

        throw new InvalidOperationException($"expected {kind} to fail with code {(int)expectedCode}, but it succeeded");
    }

    private static async Task ExpectA2AErrorAsync<T>(Func<Task<T>> action, A2AErrorCode expectedCode, string kind)
    {
        try
        {
            _ = await action().ConfigureAwait(false);
        }
        catch (A2AException error) when (error.ErrorCode == expectedCode)
        {
            return;
        }
        catch (A2AException error)
        {
            throw new InvalidOperationException($"unexpected {kind} error code: got {(int)error.ErrorCode}, want {(int)expectedCode} ({error.Message})");
        }

        throw new InvalidOperationException($"expected {kind} to fail with code {(int)expectedCode}, but it succeeded");
    }
}

internal sealed class CompatibleListTasksResponse
{
    public List<AgentTask> Tasks { get; set; } = [];

    public string? NextPageToken { get; set; }

    public int? PageSize { get; set; }

    public int? TotalSize { get; set; }
}

internal sealed class CompatibleTaskPushNotificationConfig
{
    public string TaskId { get; set; } = string.Empty;

    public PushNotificationConfig Config { get; set; } = new();
}

internal sealed class CompatibleListTaskPushNotificationConfigResponse
{
    public List<CompatibleTaskPushNotificationConfig> Configs { get; set; } = [];
}

internal sealed class ProbeOptions
{
    public required string CardUrl { get; init; }

    public required string ServerPrefix { get; init; }

    public bool ExpectPushSupported { get; init; }

    public bool ExpectPushUnsupported { get; init; }

    public bool RelaxedErrorChecks { get; init; }

    public A2AErrorCode ExpectedPushErrorCode { get; init; } = A2AErrorCode.PushNotificationNotSupported;

    public static ProbeOptions Parse(string[] args)
    {
        string? cardUrl = null;
        string? serverPrefix = null;
        var expectPushSupported = false;
        var expectPushUnsupported = false;
        var relaxedErrorChecks = false;
        var expectedPushErrorCode = A2AErrorCode.PushNotificationNotSupported;

        for (var index = 0; index < args.Length; index++)
        {
            switch (args[index])
            {
                case "--card-url":
                    index++;
                    cardUrl = args[index];
                    break;
                case "--server-prefix":
                    index++;
                    serverPrefix = args[index];
                    break;
                case "--expect-push-supported":
                    expectPushSupported = true;
                    break;
                case "--expect-push-unsupported":
                    expectPushUnsupported = true;
                    break;
                case "--relaxed-error-checks":
                    relaxedErrorChecks = true;
                    break;
                case "--expected-push-error-code":
                    index++;
                    expectedPushErrorCode = (A2AErrorCode)int.Parse(args[index], System.Globalization.CultureInfo.InvariantCulture);
                    break;
                default:
                    throw new ArgumentException($"unknown argument: {args[index]}");
            }
        }

        if (expectPushSupported && expectPushUnsupported)
        {
            throw new ArgumentException("--expect-push-supported and --expect-push-unsupported are mutually exclusive");
        }

        return new ProbeOptions
        {
            CardUrl = cardUrl ?? throw new ArgumentException("missing --card-url"),
            ServerPrefix = serverPrefix ?? throw new ArgumentException("missing --server-prefix"),
            ExpectPushSupported = expectPushSupported,
            ExpectPushUnsupported = expectPushUnsupported,
            RelaxedErrorChecks = relaxedErrorChecks,
            ExpectedPushErrorCode = expectedPushErrorCode,
        };
    }
}