# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0

FROM python:3.11

# Configure Poetry
ENV POETRY_VERSION=1.8.3
ENV POETRY_HOME=/opt/poetry
ENV POETRY_VENV=/opt/poetry-venv
ENV POETRY_CACHE_DIR=/opt/.cache

# Install rust
RUN curl https://sh.rustup.rs -sSf | bash -s -- -y

# Install poetry separated from system interpreter
RUN python3 -m venv $POETRY_VENV \
	&& $POETRY_VENV/bin/pip install -U pip setuptools \
	&& $POETRY_VENV/bin/pip install poetry==${POETRY_VERSION}

RUN apt update && apt install -y protobuf-compiler

# Add `poetry` to PATH
ENV PATH="${PATH}:${POETRY_VENV}/bin:/root/.cargo/bin"

WORKDIR /app

# Install dependencies
COPY poetry.lock pyproject.toml ./
RUN poetry install

# Run your app
COPY . /app
CMD [ "poetry", "run", "python", "langchain_agent.py", "-m", "Budapest" ]