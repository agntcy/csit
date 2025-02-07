# SPDX-FileCopyrightText: Copyright (c) 2024 2024 crewAI Inc.
# SPDX-License-Identifier: MIT


from collections import defaultdict
import json

from crewai.agent import Agent
from crewai.task import Task
from crewai.tasks.task_output import TaskOutput


class CrewEvaluator:
    """
    A class to evaluate the performance of the agents in the crew based on the tasks they have performed.

    Attributes:
        llm (LLM): The language model to use for evaluation.
    """

    tasks_scores: defaultdict = defaultdict(list)

    def __init__(self, llm):
        self.llm = llm

    def _evaluator_agent(self):
        return Agent(
            role="Task Execution Evaluator",
            goal=(
                "Your goal is to evaluate the performance of the agents in the crew based on the tasks they have performed using score from 1 to 10 evaluating on completion, quality, and overall performance."
            ),
            backstory="Evaluator agent for crew evaluation with precise capabilities to evaluate the performance of the agents in the crew based on the tasks they have performed",
            verbose=True,
            llm=self.llm,
        )

    def _evaluation_task(
        self, evaluator_agent: Agent, task_to_evaluate: Task, task_output: str
    ) -> Task:
        return Task(
            description=(
                "Based on the task description and the expected output, compare and evaluate the performance of the agents in the crew based on the Task Output they have performed using score from 1 to 10 evaluating on completion, quality, and overall performance."
                f"task_description: {task_to_evaluate.description} "
                f"task_expected_output: {task_to_evaluate.expected_output} "
                f"agent: {task_to_evaluate.agent.role if task_to_evaluate.agent else None} "
                f"agent_goal: {task_to_evaluate.agent.goal if task_to_evaluate.agent else None} "
                f"Task Output: {task_output}"
            ),
            expected_output="Evaluation Score from 1 to 10 based on the performance of the agents on the tasks. The output should have the following format: {\"score\": 10, \"comment\": \"The agent performed well on the task.\"}",
            agent=evaluator_agent,
            verbose=True
        )

    def evaluate(self, task: Task, task_output: TaskOutput):
        """Evaluates the performance of the agents in the crew based on the tasks they have performed."""
        evaluator_agent = self._evaluator_agent()
        evaluation_task = self._evaluation_task(
            evaluator_agent, task, task_output.raw
        )

        evaluation_result = evaluation_task.execute_sync()

        # convert the evaluation result json to dictionary
        try:
            evaluation_result_dict = json.loads(evaluation_result.raw)
        except json.JSONDecodeError:
            raise Exception(
                "Error: The evaluation result is not in a valid JSON format. Result: " + evaluation_result.raw
            )

        return evaluation_result_dict['score']
