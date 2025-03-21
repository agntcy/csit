# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0

FROM python:3.13.2-bookworm

WORKDIR /app

COPY . /app

RUN chmod +x /app/script.py

RUN pip install poetry
RUN poetry config virtualenvs.in-project true

# install python requirements
RUN poetry install --no-root

ENTRYPOINT ["poetry", "run", "python", "/app/script.py"]