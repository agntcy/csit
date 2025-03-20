# SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
# SPDX-License-Identifier: Apache-2.0

import asyncio
from dataclasses import dataclass
from os import environ

from llama_index.core.workflow import (
    Event,
    StartEvent,
    StopEvent,
    Workflow,
    step,
)
from llama_index.llms.azure_openai import AzureOpenAI


@dataclass
class ResearchLog:
    research_system_prompt: str = ""
    research_prompt: str = ""

    create_report_system_prompt: str = ""
    create_report_prompt: str = ""

    llm_model: str = ""

    result: str = ""


class ResearchEvent(Event):
    research: str
    topic: str


log = ResearchLog()


class ResearchFlow(Workflow):
    def set_llm(self, sys_prompt: str):
        azure_openai_api_key = environ.get("AZURE_OPENAI_API_KEY")
        azure_openai_endpoint = environ.get("AZURE_OPENAI_ENDPOINT")
        openai_api_version = environ.get(
            "AZURE_OPENAI_API_VERSION", "2025-02-01-preview"
        )
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

        log.llm_model = azure_model_version

        return llm

    @step
    async def research(self, ev: StartEvent) -> ResearchEvent:
        topic = ev.topic

        sys_prompt = f"""
            You are a {topic} Senior Data Researcher. Goal: Uncover cutting-edge developments in {topic}
            You are a seasoned researcher known for finding the most relevant information and presenting it clearly.
            """
        prompt = f"""
            Conduct a thorough research about {topic} in 2025.
            Provide 10 most relevant and interesting findings.
            """

        llm = self.set_llm(sys_prompt)

        response = await llm.acomplete(prompt)

        log.research_system_prompt = sys_prompt
        log.research_prompt = prompt

        return ResearchEvent(research=str(response), topic=str(topic))

    @step
    async def create_report(self, ev: ResearchEvent) -> StopEvent:
        research = ev.research
        topic = ev.topic

        sys_prompt = f"""
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

        log.create_report_system_prompt = sys_prompt
        log.create_report_prompt = prompt

        return StopEvent(result=str(response))


async def run(topic: str):
    w = ResearchFlow(timeout=60, verbose=False)
    result = await w.run(topic=topic)
    print(str(result))

    log.result = result


def main(topic: str):
    asyncio.run(run(topic))


if __name__ == "__main__":
    main(topic="Artificial Intelligence")
