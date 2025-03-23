# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0

import logging
import os
import json

from common._semantic_router_components import (
    TerminationMessage,
    UserProxyMessage,
    WorkerAgentMessage,
)

from autogen_core.application.logging import TRACE_LOGGER_NAME
from autogen_core.application import WorkerAgentRuntime
from autogen_core.base import MessageContext
from autogen_core.components import DefaultTopicId, RoutedAgent, message_handler
from openai import AzureOpenAI

logging.basicConfig(level=logging.DEBUG)
logger = logging.getLogger(f"{TRACE_LOGGER_NAME}.workers")


def worker_agent_runtime() -> WorkerAgentRuntime:
    return WorkerAgentRuntime(
        host_address=os.getenv("RUNTIME_ADDRESS", "localhost:50051"),
        extra_grpc_config=[
            (
                "grpc.service_config",
                json.dumps(
                    {
                        "methodConfig": [
                            {
                                "name": [{}],
                                "retryPolicy": {
                                    "maxAttempts": 10,
                                    "initialBackoff": "1s",
                                    "maxBackoff": "5s",
                                    "backoffMultiplier": 2,
                                    "retryableStatusCodes": ["UNAVAILABLE"],
                                },
                            }
                        ],
                    }
                ),
            )
        ],
    )


class WorkerAgent(RoutedAgent):
    def __init__(self, name: str) -> None:
        super().__init__("A Worker Agent")
        self._name = name

        self.reset_conversation()

        self.client: AzureOpenAI | None = None

        # Initiate openai session
        if os.getenv("AZURE_OPENAI_API_KEY"):
            self.client = AzureOpenAI(
                api_key=os.getenv("AZURE_OPENAI_API_KEY"),
                api_version=os.getenv("AZURE_OPENAI_API_VERSION"),
                azure_endpoint=os.getenv("AZURE_OPENAI_ENDPOINT"),
            )
        else:
            logger.warning("AZURE_OPENAI_API_KEY not found. Using default key.")

    def reset_conversation(self):
        self.messages = [
            {"role": "system", "content": "You are an export HR assistant!"}
        ]

    @message_handler
    async def my_message_handler(
        self, message: UserProxyMessage, ctx: MessageContext
    ) -> None:
        assert ctx.topic_id is not None
        logger.debug(f"Received message from {message.source}: {message.content}")
        if "END" in message.content:
            self.reset_conversation()

            await self.publish_message(
                TerminationMessage(
                    reason="user terminated conversation",
                    content=message.content,
                    intent=message.intent,
                    source=self.type,
                ),
                topic_id=DefaultTopicId(type="user_proxy", source=ctx.topic_id.source),
            )
        else:
            # Add message to the list
            self.messages.append({"role": "user", "content": message.content})

            answer = ""

            # LLM Call
            if self.client:
                response = self.client.chat.completions.create(
                    model=os.getenv("AZURE_OPENAI_DEPLOYMENT_NAME"),
                    messages=self.messages,
                )

                answer = response.choices[0].message.content
            else:
                answer = f"I am an expert {self._name} assistant!"

            ret = WorkerAgentMessage(
                agent_type=self.type,
                agent_id=self.id.key,
                agent_instance=hex(id(self)),
                question=message.content,
                answer=answer,
                source=ctx.topic_id.type,
            )

            logger.debug(f"Returning message: {ret}")
            await self.publish_message(
                ret,
                topic_id=DefaultTopicId(
                    type="user_proxy",
                    source=ctx.topic_id.source,
                ),
            )
