# SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
# SPDX-License-Identifier: Apache-2.0

from os import environ

from datasets import Dataset
from langchain_community.chat_models import ChatOllama
from langchain_community.embeddings import OllamaEmbeddings
from langchain_openai import AzureChatOpenAI, AzureOpenAIEmbeddings
from model.crew import run_crew
from ragas import evaluate
from ragas.embeddings import LangchainEmbeddingsWrapper
from ragas.llms import LangchainLLMWrapper
from ragas.metrics import (
    answer_correctness,
    answer_relevancy,
    answer_similarity,
    context_entity_recall,
    context_precision,
    context_recall,
    faithfulness,
    summarization_score,
)
from ragas.metrics._aspect_critic import (
    coherence,
    conciseness,
    correctness,
    harmfulness,
    maliciousness,
)

azure_openai_api_key = environ.get("AZURE_OPENAI_API_KEY", "NA")
azure_openai_endpoint = environ.get("AZURE_OPENAI_ENDPOINT", "NA")
openai_api_version = environ.get("AZURE_OPENAI_API_VERSION", "2025-02-01-preview")
azure_deployment_name = environ.get("AZURE_DEPLOYMENT_NAME", "gpt-4o-mini")
azure_model_version = environ.get("AZURE_MODEL_VERSION", "gpt-4o-mini")

eval_model_name = environ.get("LOCAL_MODEL_NAME", "llama3.1")
eval_base_url = environ.get("LOCAL_MODEL_BASE_URL", "http://localhost:11434/v1/")


def eval():
    if azure_openai_api_key != "NA":
        print("Set Azure OpenAI model for evaluation")
        evaluator_llm = LangchainLLMWrapper(
            AzureChatOpenAI(
                api_version=openai_api_version,
                azure_endpoint=azure_openai_endpoint,
                azure_deployment=azure_deployment_name,
                model=azure_model_version,
                validate_base_url=False,
            )
        )
        evaluator_embeddings = LangchainEmbeddingsWrapper(
            AzureOpenAIEmbeddings(
                api_version=openai_api_version,
                azure_endpoint=azure_openai_endpoint,
                azure_deployment=azure_deployment_name,
                model=azure_model_version,
            )
        )
    else:
        print("Set local model for evaluation")
        evaluator_llm = LangchainLLMWrapper(ChatOllama(model=eval_model_name))
        evaluator_embeddings = LangchainEmbeddingsWrapper(
            OllamaEmbeddings(model=eval_model_name)
        )

    test_input = "Gather data about new Critical CVEs from todays date"
    test_output = run_crew().raw

    print("Task input: " + test_input)
    print("Task output: " + test_output)

    d = dict()
    d["question"] = [test_input]
    d["answer"] = [test_output]
    d["context"] = [[]]
    d["retrieval_context"] = [[]]
    d["reference"] = [""]
    d["reference_contexts"] = [[]]
    d["retrieved_contexts"] = [[]]
    dataset = Dataset.from_dict(d)

    score = evaluate(
        dataset,
        [
            answer_correctness,
            answer_relevancy,
            answer_similarity,
            harmfulness,
            maliciousness,
            coherence,
            correctness,
            conciseness,
            context_entity_recall,
            context_precision,
            context_recall,
            faithfulness,
            summarization_score,
        ],
        evaluator_llm,
        evaluator_embeddings,
    )
    print("SCORE", score)

    return (dataset, score)


if __name__ == "__main__":
    eval()
