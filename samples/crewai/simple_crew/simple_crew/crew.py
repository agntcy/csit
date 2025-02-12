# SPDX-FileCopyrightText: Copyright (c) 2024 2024 crewAI Inc.
# SPDX-License-Identifier: MIT


import json
import os
import sys
import time
from os import environ

from crewai import LLM, Agent, Crew, Process, Task
from crewai.crews import CrewOutput
from crewai.project import (
    CrewBase,
    after_kickoff,
    agent,
    before_kickoff,
    crew,
    task,
)

from .utils.evaluator import CrewEvaluator

# Uncomment the following line to use an example of a custom tool
# from simple.tools.custom_tool import MyCustomTool

# Check our tools documentations for more information on how to use them
# from crewai_tools import SerperDevTool


@CrewBase
class Simple:
    """Simple crew"""

    agents_config = "config/agents.yaml"
    tasks_config = "config/tasks.yaml"

    if environ.get("AZURE_OPENAI_API_KEY") is not None:
        print("Using Azure OpenAI")
        llm = LLM(
            model=environ.get("AZURE_MODEL", "gpt-4o-mini"),
            base_url=environ.get("AZURE_OPENAI_ENDPOINT"),
            api_version=environ.get(
                "AZURE_OPENAI_API_VERSION", "2025-02-01-preview"
            ),
            api_key=environ.get("AZURE_OPENAI_API_KEY"),
            azure=True,
        )
    elif environ.get("CISCO_COGNIT_OPENAI_API_KEY") is not None:
        print("Using Cisco Cognit OpenAI")
        llm = LLM(
            model="openai/llama3.1:8b",
            base_url=environ.get("CISCO_COGNIT_OPENAI_API_URL"),
            api_key=environ.get("CISCO_COGNIT_OPENAI_API_KEY"),
        )

    else:
        print("Using Ollama")
        llm = LLM(
            model=environ.get("LOCAL_MODEL_NAME", "ollama/llama3.1"),
            base_url=environ.get(
                "LOCAL_MODEL_BASE_URL", "http://localhost:11434"
            ),
        )

    evaluator = CrewEvaluator(llm)

    @before_kickoff  # Optional hook to be executed before the crew starts
    def pull_data_example(self, inputs):
        # Example of pulling data from an external API, dynamically changing the inputs
        inputs["extra_data"] = "This is extra data"

        # Example of setting up initial values
        self.timestamp = time.time()
        self.init_timestamp = self.timestamp
        self.task_duration_map = {
            "research_task": 0,
            "reporting_task": 0,
        }
        self.task_score_map = {
            "research_task": 0,
            "reporting_task": 0,
        }

        return inputs

    @after_kickoff  # Optional hook to be executed after the crew has finished
    def log_results(self, output):
        # Example of logging results, dynamically changing the output
        print("=" * 40)
        print("Results:")
        print(output)
        print("=" * 40)

        # Example of crew metrics
        print("Crew Metrics:")
        print(f"Token usage: {output.token_usage.__dict__}")
        print(f"Task durations: {self.task_duration_map}")
        print("=" * 40)

        # Example of crew evaluation
        print("Crew Evaluation:")
        print(f"Task scores: {self.task_score_map}")
        print("=" * 40)

        return output

    @agent
    def researcher(self) -> Agent:
        return Agent(
            llm=self.llm,
            config=self.agents_config["researcher"],
            # tools=[MyCustomTool()], # Example of custom tool, loaded on the beginning of file
            verbose=True,
        )

    @agent
    def reporting_analyst(self) -> Agent:
        return Agent(
            llm=self.llm,
            config=self.agents_config["reporting_analyst"],
            verbose=True,
        )

    @task
    def research_task(self) -> Task:
        return Task(
            config=self.tasks_config["research_task"],
        )

    @task
    def reporting_task(self) -> Task:
        return Task(
            config=self.tasks_config["reporting_task"], output_file="report.md"
        )

    def task_callback(self, task_output):
        # Find task
        for t in self.tasks:
            if t.name == task_output.name:
                task = t
                break

        # Example of a task callback function that logs the task duration and score
        self.task_duration_map[task_output.name] = task._execution_time
        self.task_score_map[task_output.name] = self.evaluator.evaluate(
            task, task_output
        )

        print(
            f"Task {task_output.name} finished with agent: {task_output.agent}"
        )
        return

    @crew
    def crew(self) -> Crew:
        """Creates the Simple crew"""
        return Crew(
            agents=self.agents,  # Automatically created by the @agent decorator
            tasks=self.tasks,  # Automatically created by the @task decorator
            process=Process.sequential,
            verbose=True,
            task_callback=self.task_callback,
            # process=Process.hierarchical, # In case you wanna use that instead https://docs.crewai.com/how-to/Hierarchical/
        )
