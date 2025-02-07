# SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
# SPDX-License-Identifier: Apache-2.0

import asyncio
import logging

from common._semantic_router_components import (
    AgentRegistryBase,
    IntentClassifierBase,
    TerminationMessage,
    UserProxyMessage,
)
from common._agents import worker_agent_runtime
from autogen_core.application import WorkerAgentRuntime
from autogen_core.application.logging import TRACE_LOGGER_NAME
from autogen_core.base import MessageContext, try_get_known_serializers_for_type
from autogen_core.components import (
    DefaultTopicId,
    RoutedAgent,
    default_subscription,
    message_handler,
)

logging.basicConfig(level=logging.DEBUG)
logger = logging.getLogger(f"{TRACE_LOGGER_NAME}.semantic_router")


class MockIntentClassifier(IntentClassifierBase):
    def __init__(self):
        self.intents = {
            "finance_intent": ["finance", "money", "budget"],
            "hr_intent": ["hr", "human resources", "employee"],
        }

    async def classify_intent(self, message: str) -> str:
        for intent, keywords in self.intents.items():
            for keyword in keywords:
                if keyword in message:
                    return intent
        return "general"


class MockAgentRegistry(AgentRegistryBase):
    def __init__(self):
        self.agents = {"finance_intent": "finance", "hr_intent": "hr"}

    async def get_agent(self, intent: str) -> str:
        return self.agents[intent]


@default_subscription
class SemanticRouterAgent(RoutedAgent):
    def __init__(
        self,
        name: str,
        agent_registry: AgentRegistryBase,
        intent_classifier: IntentClassifierBase,
    ) -> None:
        super().__init__("Semantic Router Agent")
        self._name = name
        self._registry = agent_registry
        self._classifier = intent_classifier

    # The User has sent a message that needs to be routed
    @message_handler
    async def route_to_agent(
        self, message: UserProxyMessage, ctx: MessageContext
    ) -> None:
        assert ctx.topic_id is not None
        logger.debug(f"Received message from {message.source}: {message.content}")
        session_id = ctx.topic_id.source
        intent = await self._identify_intent(message)
        agent = await self._find_agent(intent)
        await self.contact_agent(agent, message, session_id)

    ## Identify the intent of the user message
    async def _identify_intent(self, message: UserProxyMessage) -> str:
        return await self._classifier.classify_intent(message.intent)

    ## Use a lookup, search, or LLM to identify the most relevant agent for the intent
    async def _find_agent(self, intent: str) -> str:
        logger.debug(f"Identified intent: {intent}")
        try:
            agent = await self._registry.get_agent(intent)
            return agent
        except KeyError:
            logger.debug("No relevant agent found for intent: " + intent)
            return "termination"

    ## Forward user message to the appropriate agent, or end the thread.
    async def contact_agent(
        self, agent: str, message: UserProxyMessage, session_id: str
    ) -> None:
        if agent == "termination":
            logger.debug("No relevant agent found")
            await self.publish_message(
                TerminationMessage(
                    reason=TerminationMessage.REASON_NO_AGENT_FOUND,
                    intent=message.intent,
                    content=message.content,
                    source=self.type,
                ),
                DefaultTopicId(type="user_proxy", source=session_id),
            )
        else:
            logger.debug("Routing to agent: " + agent)
            await self.publish_message(
                message,
                DefaultTopicId(type=agent, source=session_id),
            )


async def run_workers():
    agent_runtime: WorkerAgentRuntime = worker_agent_runtime()
    agent_runtime.start()

    serializer = try_get_known_serializers_for_type(TerminationMessage)
    agent_runtime.add_message_serializer(serializer)

    # Create the Semantic Router
    agent_registry = MockAgentRegistry()
    intent_classifier = MockIntentClassifier()
    await SemanticRouterAgent.register(
        agent_runtime,
        "router",
        lambda: SemanticRouterAgent(
            name="router",
            agent_registry=agent_registry,
            intent_classifier=intent_classifier,
        ),
    )

    await agent_runtime.stop_when_signal()


if __name__ == "__main__":
    asyncio.run(run_workers())
