# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0

FROM python:3.12-slim AS builder

ENV PIP_DISABLE_PIP_VERSION_CHECK=1 \
    PIP_NO_CACHE_DIR=1 \
    VIRTUAL_ENV=/opt/venv \
    RUSTUP_HOME=/opt/rustup \
    CARGO_HOME=/opt/cargo \
    CC=cc \
    CXX=c++ \
    CC_x86_64_unknown_linux_gnu=cc \
    CC_aarch64_unknown_linux_gnu=cc \
    AR_x86_64_unknown_linux_gnu=ar \
    AR_aarch64_unknown_linux_gnu=ar \
    STRIP_x86_64_unknown_linux_gnu=strip \
    STRIP_aarch64_unknown_linux_gnu=strip \
    PATH=/opt/venv/bin:/opt/cargo/bin:${PATH}

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    build-essential \
    ca-certificates \
    curl \
    pkg-config \
    libssl-dev && \
    rm -rf /var/lib/apt/lists/*

RUN curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | \
    sh -s -- -y --profile minimal --default-toolchain stable

RUN python -m venv /opt/venv

WORKDIR /src
COPY data-plane /src/data-plane

WORKDIR /src/data-plane/bindings/python
RUN pip install --upgrade pip maturin uniffi-bindgen==0.29.5 && \
    maturin develop --release && \
    pip install \
    "pydantic>=2.0.0" \
    "pydantic-settings>=2.0.0" \
    "prompt-toolkit>=3.0.52" \
    "pyyaml>=6.0.3" \
    "tomli>=2.4.0" \
    "protobuf>=6.33.4" \
    "googleapis-common-protos>=1.72.0"

FROM python:3.12-slim

ENV PIP_DISABLE_PIP_VERSION_CHECK=1 \
    PIP_NO_CACHE_DIR=1 \
    VIRTUAL_ENV=/opt/venv \
    PATH=/opt/venv/bin:${PATH}

COPY --from=builder /opt/venv /opt/venv
COPY --from=builder /src/data-plane/bindings/python /src/data-plane/bindings/python
COPY --from=builder /src/data-plane/bindings/python/examples /opt/venv/lib/python3.12/site-packages/examples

# Example binaries are available directly, e.g.:
# slim-bindings-p2p --local org/ns/alice --shared-secret ... --slim ...
CMD ["slim-bindings-group", "--help"]
