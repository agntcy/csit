# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0

FROM python:3.13.2-bookworm

WORKDIR /app

# Install libraries for necessary python package builds
RUN apt-get update && apt-get install build-essential python3-dev libpq-dev -y && \
    pip install --upgrade pip && \
    pip install poetry

# Install ssh wget
RUN apt-get -yq update && apt-get -yqq install ssh wget

COPY poetry.lock pyproject.toml ./

RUN poetry install --no-root

# Copy source code
COPY ./llama_sum ./llama_sum