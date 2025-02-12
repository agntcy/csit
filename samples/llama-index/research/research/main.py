# SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
# SPDX-License-Identifier: Apache-2.0

from llama_index.core.workflow import (
    Event,
    StartEvent,
    StopEvent,
    Workflow,
    step,
)

from llama_index.llms.azure_openai import AzureOpenAI
import asyncio
from os import environ


class ResearchEvent(Event):
    research: str
    topic: str

class ResearchFlow(Workflow):

    def set_llm(self, sys_prompt: str):
        azure_openai_api_key = environ.get("AZURE_OPENAI_API_KEY")
        azure_openai_endpoint = environ.get("AZURE_OPENAI_ENDPOINT")
        openai_api_version = environ.get("AZURE_OPENAI_API_VERSION", "2025-02-01-preview")
        azure_deployment_name = environ.get("AZURE_DEPLOYMENT_NAME", "gpt-4o-mini")
        azure_model_version = environ.get("AZURE_MODEL_VERSION", "gpt-4o-mini")

        llm = AzureOpenAI(
            model=azure_model_version,
            deployment_name=azure_deployment_name,
            api_key=azure_openai_api_key,
            azure_endpoint=azure_openai_endpoint,
            api_version=openai_api_version,
            temperature=0.5,
            system_prompt=sys_prompt,
        )

        return llm

    @step
    async def research(self, ev: StartEvent) -> ResearchEvent:
        topic = ev.topic

        sys_prompt =f"""
            You are a {topic} Senior Data Researcher. Goal: Uncover cutting-edge developments in {topic}
            You are a seasoned researcher known for finding the most relevant information and presenting it clearly.
            """
        prompt = f"""
            Conduct a thorough research about {topic} in 2025.
            Provide 10 most relevant and interesting findings.
            """

        llm = self.set_llm(sys_prompt)

        response = await llm.acomplete(prompt)
        return ResearchEvent(research=str(response), topic=str(topic))

    @step
    async def create_report(self, ev: ResearchEvent) -> StopEvent:
        research = ev.research
        topic = ev.topic

        sys_prompt =f"""
            You are a {topic} Reporting Analyst.
            Goal: Create detailed reports based on {topic} data analysis and research findings
            You are known for turning complex data into clear, concise reports.
            """
        prompt = f"""
            Create a detailed markdown report about {topic} based on these research findings: {research}
            Expand each finding into a full section, ensuring comprehensive coverage.
            """

        llm = self.set_llm(sys_prompt)

        response = await llm.acomplete(prompt)
        return StopEvent(result=str(response))


async def run():
    w = ResearchFlow(timeout=60, verbose=False)
    result = await w.run(topic="Artificial Intelligence")
    print(str(result))

if __name__ == "__main__":
    asyncio.run(run())
