# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0

FROM python:3.9-slim

WORKDIR /app

COPY . /app

RUN chmod +x /app/script.py

# install python requirements
RUN pip install --no-cache-dir -r requirements.txt

ENTRYPOINT ["python", "/app/script.py"]