# SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
# SPDX-License-Identifier: Apache-2.0

from abc import ABC, abstractmethod
from dataclasses import dataclass


class IntentClassifierBase(ABC):
    @abstractmethod
    async def classify_intent(self, message: str) -> str:
        pass


class AgentRegistryBase(ABC):
    @abstractmethod
    async def get_agent(self, intent: str) -> str:
        pass


@dataclass(kw_only=True)
class BaseMessage:
    """A basic message that stores the source of the message."""

    source: str


@dataclass
class TextMessage(BaseMessage):
    content: str

    def __len__(self):
        return len(self.content)


@dataclass
class UserProxyMessage(TextMessage):
    """A message that is sent from the user to the system, and needs to be routed to the appropriate agent."""
    intent: str

    pass


@dataclass
class TerminationMessage(UserProxyMessage):
    """A message that is sent from the system to the user, indicating that the conversation has ended."""

    REASON_USER_TERMINATED = "user terminated conversation"
    REASON_NO_AGENT_FOUND = "no relevant agent found"

    reason: str


@dataclass
class WorkerAgentMessage(BaseMessage):
    """A message that is sent from a worker agent to the user."""
    agent_type: str
    agent_id: str
    agent_instance: str
    question: str
    answer: str


@dataclass
class FinalResult(TextMessage):
    """A message sent from the agent to the user, indicating the end of a conversation"""

    pass
