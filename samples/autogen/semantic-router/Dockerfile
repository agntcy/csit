# SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
# SPDX-License-Identifier: Apache-2.0

# The builder image, used to build the virtual environment
FROM python:3.12.5-bookworm AS builder

RUN pip install uv==0.5.1

WORKDIR /app

COPY pyproject.toml uv.lock ./
RUN touch README.md

RUN uv sync --no-dev

# # The runtime image, used to just run the code provided its virtual environment
FROM python:3.12.5-slim-bookworm AS base

ENV VIRTUAL_ENV=/app/.venv
ENV PATH="/app/.venv/bin:$PATH"

COPY --from=builder ${VIRTUAL_ENV} ${VIRTUAL_ENV}

WORKDIR /app

COPY agents agents
COPY common common
COPY runtime runtime

ENTRYPOINT ["python"]

FROM base AS runtime

CMD ["-m", "runtime.host"]

FROM base AS agent-rh

CMD ["-m", "agents.hr"]

FROM base AS agent-finance

CMD ["-m", "agents.finance"]

FROM base AS agent-router

CMD ["-m", "agents.router"]

FROM base AS agent-proxy

CMD ["-m", "agents.proxy"]
