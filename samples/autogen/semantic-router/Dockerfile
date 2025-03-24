# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0

# The builder image, used to build the virtual environment
FROM python:3.13.2-bookworm AS builder

RUN pip install poetry

WORKDIR /app

COPY pyproject.toml poetry.lock ./
RUN touch README.md

RUN poetry config virtualenvs.in-project true
RUN poetry sync --no-root

# # The runtime image, used to just run the code provided its virtual environment
FROM python:3.13.2-slim-bookworm AS base

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
