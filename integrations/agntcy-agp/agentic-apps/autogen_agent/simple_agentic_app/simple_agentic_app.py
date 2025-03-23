# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0

import os, asyncio
from typing_extensions import Annotated
from autogen_agentchat.messages import TextMessage
from autogen_agentchat.agents import AssistantAgent
from autogen_ext.models.openai import AzureOpenAIChatCompletionClient
from autogen_core import CancellationToken
from autogen_agentchat.base import Response


azure_openai_api_key = os.environ.get("AZURE_OPENAI_API_KEY")
azure_openai_endpoint = os.environ.get("AZURE_OPENAI_ENDPOINT")
openai_api_version = os.environ.get("AZURE_OPENAI_API_VERSION", "2025-02-01-preview")
azure_model_version = os.environ.get("AZURE_MODEL_VERSION", "gpt-4o-mini")
azure_deployment_name = os.environ.get("AZURE_DEPLOYMENT_NAME", "gpt-4o-mini")

def weather_forecast(city: Annotated[str, "City of the weather forecast"]) -> str:
   return f"WEATHER: What is the current weather in {city}"

class simple_autogen_app:
    def __init__(self):
        self.model_client = AzureOpenAIChatCompletionClient(
            azure_deployment=azure_deployment_name,
            azure_endpoint=azure_openai_endpoint,
            model=azure_model_version,
            api_version=openai_api_version,
            api_key=azure_openai_api_key,
        )
        self.assistant = AssistantAgent(
            name="Assistant",
            system_message="""
                For weather forecast tasks, only use the functions you have been provided with. Reply TERMINATE
                when the task is done.
                """,
            model_client=self.model_client,
            tools=[weather_forecast],
            reflect_on_tool_use=True
        )

    async def initate_chat(self, msg: str) -> Response:
        cancellation_token = CancellationToken()
        response = await self.assistant.on_messages([TextMessage(content=msg, source="user")], cancellation_token)
        return response


if __name__ == "__main__":
    agent = simple_autogen_app()
    result = asyncio.run(agent.initate_chat("What is the weather in Budapest?"))
    if result.chat_message.content == "TERMINATE":
        try:
            result.inner_messages
            weatherQ = result.inner_messages[-1].content[-1].content.split(":")
            print(weatherQ)
        except:
            print(result)
    else:
        print(result)
