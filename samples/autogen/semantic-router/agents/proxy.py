# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0

import asyncio
import logging
from dataclasses import asdict
from typing import Mapping

import uvicorn
from autogen_core import (
    TRACE_LOGGER_NAME,
    DefaultSubscription,
    DefaultTopicId,
    MessageContext,
    RoutedAgent,
    message_handler,
    try_get_known_serializers_for_type,
)
from autogen_ext.runtimes.grpc import GrpcWorkerAgentRuntime
from common._agents import worker_agent_runtime
from common._semantic_router_components import (
    FinalResult,
    TerminationMessage,
    UserProxyMessage,
    WorkerAgentMessage,
)
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel

logging.basicConfig(level=logging.DEBUG)
logger = logging.getLogger(f"{TRACE_LOGGER_NAME}.proxy")


class UserProxyAgent(RoutedAgent):
    """An agent that proxies user input from the console. Override the `get_user_input`
    method to customize how user input is retrieved.

    Args:
        description (str): The description of the agent.
    """

    def __init__(
        self,
        description: str,
        contexts: Mapping[str, asyncio.Future],
    ) -> None:
        self.contexts = contexts
        super().__init__(description)

    # When a conversation ends
    @message_handler
    async def on_terminate(
        self, message: TerminationMessage, ctx: MessageContext
    ) -> None:
        assert ctx.topic_id is not None
        """Handle a publish now message. This method prompts the user for input, then publishes it."""
        logger.debug(f"Ending conversation with {ctx.sender} because {message.reason}")
        self.contexts[ctx.topic_id.source].set_result(message)

    # When the agent responds back, user proxy adds it to history and then
    # sends to Closure Agent for API to respond
    @message_handler
    async def on_agent_message(
        self, message: WorkerAgentMessage, ctx: MessageContext
    ) -> None:
        assert ctx.topic_id is not None
        logger.debug(f"Received message from {message.source}. Content: {message}")
        logger.debug("Returning message to user")
        self.contexts[ctx.topic_id.source].set_result(message)


class Message(BaseModel):
    intent: str
    message: str
    context: str


class Proxy:
    def __init__(self):
        self.app = FastAPI()
        self.setup_routes()
        self.contexts = {}

        config = uvicorn.Config(
            self.app,
            host="0.0.0.0",
        )
        self.server = uvicorn.Server(config)

    async def run(self):
        # start agent
        asyncio.create_task(self.run_workers())

        # start uvicorn server
        await self.server.serve()

        # stop when signal
        await self.agent_runtime.stop()

    def setup_routes(self):
        @self.app.get("/healthz")
        async def health():
            return {"status": "ok"}

        @self.app.post("/message")
        async def receive_message(data: Message):
            logger.info(
                f"Received message: intent: {data.intent}; message: {data.message} ctx: {data.context}"
            )

            # Create a future for the context
            self.contexts[data.context] = asyncio.Future()

            await self.agent_runtime.publish_message(
                UserProxyMessage(
                    intent=data.intent, content=data.message, source=data.context
                ),
                topic_id=DefaultTopicId(type="default", source=data.context),
            )

            # Wait for the response
            try:
                response: (
                    WorkerAgentMessage | TerminationMessage
                ) = await asyncio.wait_for(
                    self.contexts[data.context],
                    timeout=30,
                )
            except asyncio.TimeoutError:
                raise HTTPException(status_code=500, detail="Internal server error")

            if isinstance(response, TerminationMessage):
                if response.reason == TerminationMessage.REASON_NO_AGENT_FOUND:
                    raise HTTPException(status_code=404, detail="No agent found")

            return asdict(response)

    async def run_workers(self):
        self.agent_runtime: GrpcWorkerAgentRuntime = worker_agent_runtime()
        await self.agent_runtime.start()

        serializer_proxy_message = try_get_known_serializers_for_type(UserProxyMessage)
        serializer_final_result = try_get_known_serializers_for_type(FinalResult)
        self.agent_runtime.add_message_serializer(
            [serializer_proxy_message, serializer_final_result]
        )

        # Create the User Proxy Agent
        await UserProxyAgent.register(
            self.agent_runtime,
            "user_proxy",
            lambda: UserProxyAgent("user_proxy", self.contexts),
        )
        await self.agent_runtime.add_subscription(
            DefaultSubscription(topic_type="user_proxy", agent_type="user_proxy")
        )


if __name__ == "__main__":
    proxy = Proxy()
    asyncio.run(proxy.run())
