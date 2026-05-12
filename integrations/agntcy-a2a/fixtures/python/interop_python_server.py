# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0

"""A2A Python v1.0 interoperability fixture server.

This server mirrors the behavior contract already exercised by the shared Go
interop suite, but implements it using the Python SDK from the
1.0-dev branch.

Use this file when adding or adjusting Python server-side parity behavior. Keep
transport bootstrapping in this file and leave cross-SDK assertions in the Go
test harness.
"""

from __future__ import annotations

import argparse
import asyncio
import json
import logging
from collections.abc import AsyncIterator
from datetime import datetime, timezone
from uuid import uuid4

import grpc
import grpc.aio
import uvicorn

from google.protobuf.json_format import MessageToDict, ParseDict
from starlette.applications import Starlette
from starlette.middleware.base import BaseHTTPMiddleware
from starlette.requests import Request
from starlette.responses import JSONResponse, Response, StreamingResponse
from starlette.routing import Route

import a2a.types.a2a_pb2_grpc as a2a_pb2_grpc

from a2a.server.agent_execution import AgentExecutor, RequestContext
from a2a.server.context import ServerCallContext
from a2a.server.events import EventQueue
from a2a.server.request_handlers import GrpcHandler, LegacyRequestHandler
from a2a.server.routes import (
    create_agent_card_routes,
    create_rest_routes,
)
from a2a.server.routes.common import DefaultServerCallContextBuilder
from a2a.server.routes.jsonrpc_dispatcher import JsonRpcDispatcher
from a2a.server.tasks import (
    InMemoryPushNotificationConfigStore,
    InMemoryTaskStore,
)
from a2a.types import a2a_pb2
from a2a.helpers import get_message_text
from a2a.utils import TransportProtocol
from a2a.utils.constants import AGENT_CARD_WELL_KNOWN_PATH, PROTOCOL_VERSION_1_0, VERSION_HEADER
from a2a.utils.error_handlers import rest_error_handler
from a2a.utils.errors import TaskNotFoundError
from a2a.utils.version_validator import validate_version


LOGGER = logging.getLogger(__name__)

SERVER_PREFIX = 'python'
PENDING_REQUEST_TEXT = 'pending'
MESSAGE_ONLY_REQUEST_TEXT = 'message-only'
TASK_FAILURE_REQUEST_TEXT = 'task-failure'
MULTI_TURN_START_REQUEST_TEXT = 'multi-turn start'
MULTI_TURN_CONTINUE_REQUEST_TEXT = 'multi-turn continue'
STREAMING_REQUEST_TEXT = 'streaming'
LONG_RUNNING_REQUEST_TEXT = 'long-running'
DATA_TYPES_REQUEST_TEXT = 'data-types'
EXTENDED_CARD_SCHEME_ID = 'bearer_token'


def build_text_part(text: str) -> a2a_pb2.Part:
    return a2a_pb2.Part(text=text)


def build_data_part(data: dict[str, object]) -> a2a_pb2.Part:
    part = a2a_pb2.Part()
    ParseDict({'data': data}, part)
    return part


def build_agent_message(task_id: str, context_id: str, text: str) -> a2a_pb2.Message:
    return a2a_pb2.Message(
        role=a2a_pb2.Role.ROLE_AGENT,
        task_id=task_id,
        context_id=context_id,
        message_id=str(uuid4()),
        parts=[build_text_part(text)],
    )


def build_task(context: RequestContext, state: int, text: str) -> a2a_pb2.Task:
    if context.current_task is not None:
        task = a2a_pb2.Task()
        task.CopyFrom(context.current_task)
        if task.history:
            user_history = [entry for entry in task.history if entry.role == a2a_pb2.Role.ROLE_USER]
            task.ClearField('history')
            task.history.extend(user_history)
        if context.message is not None:
            if not task.history or task.history[-1].message_id != context.message.message_id:
                task.history.append(context.message)
    else:
        if context.message is None:
            raise RuntimeError('message is required to construct a task')
        task = a2a_pb2.Task(
            id=context.task_id or str(uuid4()),
            context_id=context.context_id or context.message.context_id or str(uuid4()),
            history=[context.message],
        )

    task.status.CopyFrom(
        a2a_pb2.TaskStatus(
            state=state,
            message=build_agent_message(task.id, task.context_id, text),
            timestamp=datetime.now(timezone.utc),
        )
    )
    return task


def build_status_update(task_id: str, context_id: str, state: int, text: str) -> a2a_pb2.TaskStatusUpdateEvent:
    return a2a_pb2.TaskStatusUpdateEvent(
        task_id=task_id,
        context_id=context_id,
        status=a2a_pb2.TaskStatus(
            state=state,
            message=build_agent_message(task_id, context_id, text),
            timestamp=datetime.now(timezone.utc),
        ),
    )


def build_data_types_artifact() -> a2a_pb2.Artifact:
    return a2a_pb2.Artifact(
        artifact_id=str(uuid4()),
        name='data-types-artifact',
        description='Mixed content artifact for scenario parity',
        parts=[
            build_text_part('structured summary'),
            build_data_part({'kind': 'report', 'items': 2}),
            a2a_pb2.Part(
                url='https://example.invalid/diagram.svg',
                media_type='image/svg+xml',
                filename='diagram.svg',
            ),
        ],
    )


def build_skill(skill_id: str, description: str) -> a2a_pb2.AgentSkill:
    return a2a_pb2.AgentSkill(
        id=skill_id,
        name=skill_id,
        description=description,
        tags=['csit', 'scenario-parity'],
    )


def scenario_skills() -> list[a2a_pb2.AgentSkill]:
    return [
        build_skill('message-only', 'Returns a message response without creating a task.'),
        build_skill('task-lifecycle', 'Creates, lists, fetches, and cancels tasks.'),
        build_skill('task-failure', 'Returns a failed task response.'),
        build_skill('task-cancel', 'Creates a cancelable working task.'),
        build_skill('multi-turn', 'Requests more input before completing the task.'),
        build_skill('streaming', 'Streams task and artifact updates.'),
        build_skill('long-running', 'Returns early and completes asynchronously.'),
        build_skill('data-types', 'Produces text, structured data, and URL parts.'),
    ]


def build_agent_card(
    base_url: str,
    protocol: str,
    extended: bool,
    grpc_port: int | None = None,
) -> a2a_pb2.AgentCard:
    if protocol == 'rest':
        name = 'CSIT Python REST Agent'
        interface = a2a_pb2.AgentInterface(
            protocol_binding=TransportProtocol.HTTP_JSON.value,
            protocol_version='1.0',
            url=base_url,
        )
    elif protocol == 'grpc':
        if grpc_port is None:
            raise ValueError('gRPC transport requires --grpc-port')
        name = 'CSIT Python gRPC Agent'
        interface = a2a_pb2.AgentInterface(
            protocol_binding=TransportProtocol.GRPC.value,
            protocol_version='1.0',
            url=f'127.0.0.1:{grpc_port}',
        )
    else:
        name = 'CSIT Python JSON-RPC Agent'
        interface = a2a_pb2.AgentInterface(
            protocol_binding=TransportProtocol.JSONRPC.value,
            protocol_version='1.0',
            url=f'{base_url}/rpc',
        )

    card = a2a_pb2.AgentCard(
        name=name,
        description=(
            'Python interoperability fixture for CSIT (extended)'
            if extended
            else 'Python interoperability fixture for CSIT'
        ),
        version='1.0.0',
        supported_interfaces=[interface],
        capabilities=a2a_pb2.AgentCapabilities(
            streaming=True,
            push_notifications=True,
            extended_agent_card=True,
        ),
        default_input_modes=['text/plain'],
        default_output_modes=['text/plain'],
        skills=scenario_skills(),
    )
    card.security_schemes[EXTENDED_CARD_SCHEME_ID].CopyFrom(
        a2a_pb2.SecurityScheme(
            http_auth_security_scheme=a2a_pb2.HTTPAuthSecurityScheme(
                scheme='Bearer',
                bearer_format='JWT',
                description='Bearer token authentication',
            )
        )
    )
    return card


class InteropPushNotificationConfigStore(InMemoryPushNotificationConfigStore):
    async def set_info(
        self,
        task_id: str,
        notification_config: a2a_pb2.TaskPushNotificationConfig,
        context: ServerCallContext,
    ) -> None:
        authentication = notification_config.authentication
        if (
            not notification_config.url
            and not notification_config.token
            and not authentication.scheme
            and not authentication.credentials
        ):
            return

        await super().set_info(task_id, notification_config, context)








def clone_request_with_body(
    request: Request,
    body: bytes,
    header_overrides: dict[bytes, bytes] | None = None,
) -> Request:
    scope = dict(request.scope)
    normalized_overrides = {
        key.lower(): value
        for key, value in (header_overrides or {}).items()
    }
    headers = [
        (key, value)
        for key, value in request.scope.get('headers', [])
        if key.lower() != b'content-length' and key.lower() not in normalized_overrides
    ]
    headers.extend(normalized_overrides.items())
    headers.append((b'content-length', str(len(body)).encode()))
    scope['headers'] = headers

    sent = False

    async def receive() -> dict[str, object]:
        nonlocal sent
        if sent:
            return {'type': 'http.request', 'body': b'', 'more_body': False}
        sent = True
        return {'type': 'http.request', 'body': body, 'more_body': False}

    return Request(scope, receive)



def forward_response_headers(response: Response) -> dict[str, str]:
    return {
        key: value
        for key, value in response.headers.items()
        if key.lower() not in {'content-length', 'content-type'}
    }


async def normalize_sse_body(response: Response) -> AsyncIterator[bytes]:
    pending = b''

    async for chunk in response.body_iterator:
        data = chunk if isinstance(chunk, bytes) else chunk.encode()
        pending += data

        if not pending:
            continue

        if pending.endswith(b'\r'):
            emit = pending[:-1]
            pending = b'\r'
        else:
            emit = pending
            pending = b''

        if emit:
            yield emit.replace(b'\r\n', b'\n')

    if pending:
        yield pending.replace(b'\r\n', b'\n')


def normalize_sse_response(response: Response) -> Response:
    return StreamingResponse(
        normalize_sse_body(response),
        status_code=response.status_code,
        headers=forward_response_headers(response),
        media_type='text/event-stream',
        background=response.background,
    )


class SSELineEndingsMiddleware(BaseHTTPMiddleware):
    async def dispatch(self, request: Request, call_next) -> Response:
        response = await call_next(request)
        content_type = response.headers.get('content-type', '')
        if 'text/event-stream' not in content_type.lower():
            return response
        return normalize_sse_response(response)


def create_jsonrpc_routes_with_compat(request_handler: LegacyRequestHandler) -> list[Route]:
    dispatcher = JsonRpcDispatcher(request_handler=request_handler)
    version_header = {VERSION_HEADER.lower().encode(): PROTOCOL_VERSION_1_0.encode()}

    async def endpoint(request: Request) -> Response:
        body = await request.body()
        return await dispatcher.handle_requests(
            clone_request_with_body(request, body, version_header)
        )

    return [Route(path='/rpc', endpoint=endpoint, methods=['POST'])]


def create_rest_routes_with_compat(request_handler: LegacyRequestHandler) -> list[Route]:
    rest_routes = create_rest_routes(request_handler=request_handler)
    context_builder = DefaultServerCallContextBuilder()

    def build_context(request: Request) -> ServerCallContext:
        context = context_builder.build(request)
        if 'tenant' in request.path_params:
            context.tenant = request.path_params['tenant']
        return context

    async def create_push_config_endpoint(request: Request) -> Response:
        params = a2a_pb2.TaskPushNotificationConfig(task_id=request.path_params['id'])
        body = await request.body()
        if body:
            payload = json.loads(body)
            if isinstance(payload, dict):
                ParseDict(payload, params)
        response = await request_handler.on_create_task_push_notification_config(
            params,
            build_context(request),
        )
        return JSONResponse(
            content=MessageToDict(response, preserving_proto_field_name=False)
        )

    async def get_push_config_endpoint(request: Request) -> Response:
        response = await request_handler.on_get_task_push_notification_config(
            a2a_pb2.GetTaskPushNotificationConfigRequest(
                task_id=request.path_params['id'],
                id=request.path_params['push_id'],
            ),
            build_context(request),
        )
        return JSONResponse(
            content=MessageToDict(response, preserving_proto_field_name=False)
        )

    async def list_push_configs_endpoint(request: Request) -> Response:
        params = a2a_pb2.ListTaskPushNotificationConfigsRequest(
            task_id=request.path_params['id']
        )
        page_size = request.query_params.get('pageSize')
        if page_size:
            params.page_size = int(page_size)
        page_token = request.query_params.get('pageToken')
        if page_token:
            params.page_token = page_token
        response = await request_handler.on_list_task_push_notification_configs(
            params,
            build_context(request),
        )
        return JSONResponse(
            content=MessageToDict(response, preserving_proto_field_name=False)
        )

    @rest_error_handler
    async def get_task_endpoint(request: Request) -> Response:
        @validate_version(PROTOCOL_VERSION_1_0)
        async def _handler(context: ServerCallContext) -> a2a_pb2.Task:
            params = a2a_pb2.GetTaskRequest(id=request.path_params['id'])

            history_length = request.query_params.get('historyLength')
            if history_length:
                params.history_length = int(history_length)

            task = await request_handler.on_get_task(params, context)
            if task:
                return task
            raise TaskNotFoundError

        response = await _handler(build_context(request))
        return JSONResponse(
            content=MessageToDict(
                response,
                preserving_proto_field_name=False,
                always_print_fields_with_no_presence=True,
            )
        )

    async def delete_push_config_endpoint(request: Request) -> Response:
        await request_handler.on_delete_task_push_notification_config(
            a2a_pb2.DeleteTaskPushNotificationConfigRequest(
                task_id=request.path_params['id'],
                id=request.path_params['push_id'],
            ),
            build_context(request),
        )
        return JSONResponse(content={})

    async def list_tasks_endpoint(request: Request) -> Response:
        params = a2a_pb2.ListTasksRequest()

        context_id = request.query_params.get('contextId')
        if context_id:
            params.context_id = context_id

        status = request.query_params.get('status')
        if status:
            ParseDict({'status': status}, params)

        page_size = request.query_params.get('pageSize')
        if page_size:
            params.page_size = int(page_size)

        page_token = request.query_params.get('pageToken')
        if page_token:
            params.page_token = page_token

        history_length = request.query_params.get('historyLength')
        if history_length:
            params.history_length = int(history_length)

        include_artifacts = request.query_params.get('includeArtifacts')
        if include_artifacts is not None:
            params.include_artifacts = include_artifacts.lower() == 'true'

        response = await request_handler.on_list_tasks(params, build_context(request))
        return JSONResponse(
            content=MessageToDict(
                response,
                preserving_proto_field_name=False,
                always_print_fields_with_no_presence=True,
            )
        )

    return [
        Route(path='/tasks/{id}/pushNotificationConfigs', endpoint=create_push_config_endpoint, methods=['POST']),
        Route(path='/tasks/{id}/pushNotificationConfigs', endpoint=list_push_configs_endpoint, methods=['GET']),
        Route(path='/tasks/{id}/pushNotificationConfigs/{push_id}', endpoint=get_push_config_endpoint, methods=['GET']),
        Route(path='/tasks/{id}/pushNotificationConfigs/{push_id}', endpoint=delete_push_config_endpoint, methods=['DELETE']),
        Route(path='/tasks/{id}', endpoint=get_task_endpoint, methods=['GET']),
        Route(path='/tasks', endpoint=list_tasks_endpoint, methods=['GET']),
        *rest_routes,
    ]


class InteropExecutor(AgentExecutor):
    """Implements the scenario matrix used by the shared interop behaviors."""

    async def execute(self, context: RequestContext, event_queue: EventQueue) -> None:
        request_text = get_message_text(context.message) if context.message else ''

        if request_text == MESSAGE_ONLY_REQUEST_TEXT:
            await event_queue.enqueue_event(
                a2a_pb2.Message(
                    role=a2a_pb2.Role.ROLE_AGENT,
                    message_id=str(uuid4()),
                    parts=[build_text_part(f'{SERVER_PREFIX} server message-only response')],
                )
            )
            return

        if request_text == TASK_FAILURE_REQUEST_TEXT:
            await event_queue.enqueue_event(
                build_task(
                    context,
                    a2a_pb2.TaskState.TASK_STATE_FAILED,
                    f'{SERVER_PREFIX} server failed task',
                )
            )
            return

        if request_text == MULTI_TURN_START_REQUEST_TEXT:
            await event_queue.enqueue_event(
                build_task(
                    context,
                    a2a_pb2.TaskState.TASK_STATE_INPUT_REQUIRED,
                    f'{SERVER_PREFIX} server needs more input',
                )
            )
            return

        if request_text == MULTI_TURN_CONTINUE_REQUEST_TEXT:
            await event_queue.enqueue_event(
                build_task(
                    context,
                    a2a_pb2.TaskState.TASK_STATE_COMPLETED,
                    f'{SERVER_PREFIX} server multi-turn completed',
                )
            )
            return

        if request_text == STREAMING_REQUEST_TEXT:
            working_task = build_task(
                context,
                a2a_pb2.TaskState.TASK_STATE_WORKING,
                f'{SERVER_PREFIX} server streaming started',
            )
            await event_queue.enqueue_event(working_task)

            artifact_id = str(uuid4())
            await event_queue.enqueue_event(
                a2a_pb2.TaskArtifactUpdateEvent(
                    task_id=working_task.id,
                    context_id=working_task.context_id,
                    artifact=a2a_pb2.Artifact(
                        artifact_id=artifact_id,
                        parts=[build_text_part('streaming chunk 1')],
                    ),
                )
            )
            await event_queue.enqueue_event(
                a2a_pb2.TaskArtifactUpdateEvent(
                    task_id=working_task.id,
                    context_id=working_task.context_id,
                    artifact=a2a_pb2.Artifact(
                        artifact_id=artifact_id,
                        parts=[build_text_part('streaming chunk 2')],
                    ),
                    append=True,
                )
            )
            await event_queue.enqueue_event(
                build_status_update(
                    working_task.id,
                    working_task.context_id,
                    a2a_pb2.TaskState.TASK_STATE_COMPLETED,
                    f'{SERVER_PREFIX} server streaming complete',
                )
            )
            return

        if request_text == LONG_RUNNING_REQUEST_TEXT:
            working_task = build_task(
                context,
                a2a_pb2.TaskState.TASK_STATE_WORKING,
                f'{SERVER_PREFIX} server long-running started',
            )
            await event_queue.enqueue_event(working_task)
            await asyncio.sleep(0.15)
            await event_queue.enqueue_event(
                build_status_update(
                    working_task.id,
                    working_task.context_id,
                    a2a_pb2.TaskState.TASK_STATE_WORKING,
                    f'{SERVER_PREFIX} server long-running progress',
                )
            )
            await asyncio.sleep(0.15)
            await event_queue.enqueue_event(
                build_status_update(
                    working_task.id,
                    working_task.context_id,
                    a2a_pb2.TaskState.TASK_STATE_COMPLETED,
                    f'{SERVER_PREFIX} server long-running complete',
                )
            )
            return

        if request_text == DATA_TYPES_REQUEST_TEXT:
            task = build_task(
                context,
                a2a_pb2.TaskState.TASK_STATE_COMPLETED,
                f'{SERVER_PREFIX} server data-types ready',
            )
            task.artifacts.append(build_data_types_artifact())
            await event_queue.enqueue_event(task)
            return

        state = a2a_pb2.TaskState.TASK_STATE_COMPLETED
        if request_text == PENDING_REQUEST_TEXT:
            state = a2a_pb2.TaskState.TASK_STATE_WORKING
        await event_queue.enqueue_event(
            build_task(
                context,
                state,
                f'{SERVER_PREFIX} server received: {request_text}',
            )
        )

    async def cancel(self, context: RequestContext, event_queue: EventQueue) -> None:
        task_id = ''
        context_id = ''
        if context.current_task is not None:
            task_id = context.current_task.id
            context_id = context.current_task.context_id
        if context.task_id:
            task_id = context.task_id
        if context.context_id:
            context_id = context.context_id

        if not task_id:
            raise RuntimeError('task_id is required for cancellation')

        await event_queue.enqueue_event(
            build_status_update(
                task_id,
                context_id,
                a2a_pb2.TaskState.TASK_STATE_CANCELED,
                f'{SERVER_PREFIX} server canceled task',
            )
        )


def build_app(
    port: int,
    protocol: str,
    grpc_port: int | None = None,
) -> tuple[Starlette, LegacyRequestHandler]:
    base_url = f'http://127.0.0.1:{port}'
    public_card = build_agent_card(base_url, protocol, extended=False, grpc_port=grpc_port)
    extended_card = build_agent_card(base_url, protocol, extended=True, grpc_port=grpc_port)
    request_handler = LegacyRequestHandler(
        agent_executor=InteropExecutor(),
        task_store=InMemoryTaskStore(),
        push_config_store=InteropPushNotificationConfigStore(),
        agent_card=public_card,
        extended_agent_card=extended_card,
    )

    routes = [
        *create_agent_card_routes(public_card, card_url=AGENT_CARD_WELL_KNOWN_PATH),
    ]
    if protocol == 'rest':
        routes.extend(create_rest_routes_with_compat(request_handler))
    elif protocol == 'jsonrpc':
        routes.extend(create_jsonrpc_routes_with_compat(request_handler))

    app = Starlette(routes=routes)
    app.add_middleware(SSELineEndingsMiddleware)
    return app, request_handler


class InteropGrpcHandler(GrpcHandler):
    async def ListTaskPushNotificationConfigs(
        self,
        request: a2a_pb2.ListTaskPushNotificationConfigsRequest,
        context: grpc.aio.ServicerContext,
    ) -> a2a_pb2.ListTaskPushNotificationConfigsResponse:
        if request.page_size == 0:
            request.page_size = 1
        return await super().ListTaskPushNotificationConfigs(request, context)

    async def ListTasks(
        self,
        request: a2a_pb2.ListTasksRequest,
        context: grpc.aio.ServicerContext,
    ) -> a2a_pb2.ListTasksResponse:
        if request.page_size == 0:
            request.page_size = 1
        return await super().ListTasks(request, context)


async def run_grpc_fixture(http_port: int, grpc_port: int) -> None:
    app, request_handler = build_app(http_port, 'grpc', grpc_port=grpc_port)
    grpc_server = grpc.aio.server()
    grpc_address = f'127.0.0.1:{grpc_port}'

    a2a_pb2_grpc.add_A2AServiceServicer_to_server(
        InteropGrpcHandler(request_handler),
        grpc_server,
    )
    grpc_server.add_insecure_port(grpc_address)
    await grpc_server.start()

    LOGGER.info(
        'python grpc fixture listening on %s with card http://127.0.0.1:%d',
        grpc_address,
        http_port,
    )

    server = uvicorn.Server(
        uvicorn.Config(
            app,
            host='127.0.0.1',
            port=http_port,
            access_log=False,
            log_level='warning',
            lifespan='off',
        )
    )
    try:
        await server.serve()
    finally:
        await grpc_server.stop(0)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description='Run the Python A2A interop fixture server.')
    parser.add_argument('--port', type=int, required=True)
    parser.add_argument('--protocol', choices=('jsonrpc', 'rest', 'grpc'), required=True)
    parser.add_argument('--grpc-port', type=int)
    return parser.parse_args()


def main() -> None:
    logging.basicConfig(level=logging.INFO, format='%(asctime)s %(levelname)s %(message)s')
    args = parse_args()
    if args.protocol == 'grpc':
        if args.grpc_port is None:
            raise SystemExit('--grpc-port is required when --protocol=grpc')
        asyncio.run(run_grpc_fixture(args.port, args.grpc_port))
        return

    app, _ = build_app(args.port, args.protocol)
    LOGGER.info('python %s fixture listening on http://127.0.0.1:%d', args.protocol, args.port)
    uvicorn.run(
        app,
        host='127.0.0.1',
        port=args.port,
        access_log=False,
        log_level='warning',
        lifespan='off',
    )


if __name__ == '__main__':
    main()