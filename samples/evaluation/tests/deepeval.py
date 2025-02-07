#!/usr/bin/env python
# SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
# SPDX-License-Identifier: Apache-2.0

from os import environ
from deepeval.test_case import LLMTestCase
from deepeval.metrics import AnswerRelevancyMetric, BiasMetric, ToxicityMetric
from deepeval.cli.main import set_local_model_env, unset_local_model_env, set_azure_openai_env, unset_azure_openai_env
from deepeval.dataset import EvaluationDataset
from model.crew import run_crew

azure_openai_api_key = environ.get("AZURE_OPENAI_API_KEY", "NA")
azure_openai_endpoint = environ.get("AZURE_OPENAI_ENDPOINT", "NA")
openai_api_version = environ.get("AZURE_OPENAI_API_VERSION", "2024-08-01-preview")
azure_deployment_name = environ.get("AZURE_DEPLOYMENT_NAME", "gpt-4o-mini")
azure_model_version = environ.get("AZURE_MODEL_VERSION", "gpt-4o-mini")

eval_model_name = environ.get("LOCAL_MODEL_NAME", "llama3.1")
eval_base_url = environ.get("LOCAL_MODEL_BASE_URL", "http://localhost:11434/v1/")

def eval():
    if azure_openai_api_key != "NA":
        print("Set Azure OpenAI model for evaluation")
        set_azure_openai_env(
            azure_openai_api_key=azure_openai_api_key,
            azure_openai_endpoint=azure_openai_endpoint,
            openai_api_version=openai_api_version,
            azure_deployment_name=azure_deployment_name,
            azure_model_version=azure_model_version,
        )
    else:
        print("Set local model for evaluation")
        set_local_model_env(model_name=eval_model_name,
                            base_url=eval_base_url,
                            api_key="dummy-key",
                            format='json',
        )

    test_input = "Gather data about new Critical CVEs from todays date"
    test_output = run_crew().raw

    print("Task input: " + test_input)
    print("Task output: " + test_output)

    test_cases = []
    test_case = LLMTestCase(
        input = test_input,
        actual_output = test_output,
    )
    test_cases.append(test_case)

    print("Start tasks analysis")
    answer_relevancy_metric = AnswerRelevancyMetric(threshold=0.5)
    bias_metric = BiasMetric(threshold=0.5)
    toxicity_metric = ToxicityMetric(threshold=0.5)

    dataset = EvaluationDataset(test_cases=test_cases)
    dataset.evaluate([answer_relevancy_metric, bias_metric, toxicity_metric])


    print("Test End")
    if azure_openai_api_key != "NA":
        unset_azure_openai_env()
    else:
        unset_local_model_env()

if __name__ == "__main__":
    eval()
