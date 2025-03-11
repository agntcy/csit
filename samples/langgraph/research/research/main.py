# SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
# SPDX-License-Identifier: Apache-2.0

import os
from typing import List, TypedDict

from langchain_core.messages import HumanMessage, SystemMessage
from langchain_ollama import ChatOllama
from langchain_openai import AzureChatOpenAI
from langgraph.graph import END, StateGraph

if os.environ.get("AZURE_OPENAI_API_KEY") is not None:
    llm = AzureChatOpenAI(
        azure_deployment=os.environ.get("AZURE_OPENAI_DEPLOYMENT_NAME", "gpt-4o-mini"),
        api_version=os.environ.get("AZURE_OPENAI_API_VERSION", "2025-02-01-preview"),
        temperature=0,
        max_tokens=None,
        timeout=None,
        max_retries=2,
    )
else:
    llm = ChatOllama(
        base_url=os.environ.get("LOCAL_MODEL_BASE_URL", "http://localhost:11434"),
        model=os.environ.get("LOCAL_MODEL_NAME", "llama3.2"),
        temperature=0,
    )


class ResearchState(TypedDict):
    topic: str

    researcher_system_prompt: str
    researcher_research_prompt: str
    research_findings: List[str]

    reporting_system_prompt: str
    reporting_report_prompt: str
    report: str


def researcher_node(state: ResearchState) -> dict:
    system_prompt = f"""
    You are a {state["topic"]} Senior Data Researcher.
    Goal: Uncover cutting-edge developments in {state["topic"]}
    You are a seasoned researcher known for finding the most relevant information and presenting it clearly.
    """

    research_prompt = f"""
    Conduct a thorough research about {state["topic"]} in 2024.
    Provide 10 most relevant and interesting findings.
    """

    response = llm.invoke(
        [
            SystemMessage(content=system_prompt),
            HumanMessage(content=research_prompt),
        ]
    )

    findings = [
        finding.strip() for finding in response.content.split("\n") if finding.strip()
    ]

    return {
        "researcher_system_prompt": system_prompt,
        "researcher_research_prompt": research_prompt,
        "research_findings": findings,
    }


def reporting_node(state: ResearchState) -> dict:
    system_prompt = f"""
    You are a {state["topic"]} Reporting Analyst.
    Goal: Create detailed reports based on {state["topic"]} data analysis and research findings
    You are known for turning complex data into clear, concise reports.
    """

    report_prompt = f"""
    Create a detailed markdown report about {state["topic"]} based on these research findings:
    {"\n".join(state["research_findings"])}

    Expand each finding into a full section, ensuring comprehensive coverage.
    """

    response = llm.invoke(
        [
            SystemMessage(content=system_prompt),
            HumanMessage(content=report_prompt),
        ]
    )

    return {
        "reporting_system_prompt": system_prompt,
        "reporting_report_prompt": report_prompt,
        "report": response.content,
    }


def build_workflow():
    workflow = StateGraph(ResearchState)

    workflow.add_node("researcher", researcher_node)
    workflow.add_node("reporting_analyst", reporting_node)

    workflow.set_entry_point("researcher")
    workflow.add_edge("researcher", "reporting_analyst")
    workflow.add_edge("reporting_analyst", END)

    return workflow.compile()


def main(topic: str):
    initial_state = {
        "topic": topic,
        "research_findings": [],
        "report": "",
    }

    workflow = build_workflow()
    result = workflow.invoke(initial_state)

    print("Research Report:")
    print(result["report"])

    return result


if __name__ == "__main__":
    topic = "Artificial Intelligence"
    main(topic)
