# SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
# SPDX-License-Identifier: Apache-2.0

FROM python:3.12-slim

WORKDIR /app

ENV POETRY_VERSION=1.8.3

# Install libraries for necessary python package builds
RUN apt-get update && apt-get install build-essential python3-dev libpq-dev -y && \
    pip install --upgrade pip && \
    pip install --upgrade poetry==${POETRY_VERSION}

# Install ssh wget 
RUN apt-get -yq update && apt-get -yqq install ssh wget

COPY poetry.lock pyproject.toml ./

RUN poetry install

# Copy source code
COPY ./llama_sum ./llama_sum