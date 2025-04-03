FROM debian:bookworm-slim

ENV DEBIAN_FRONTEND=noninteractive

# Install required tools
RUN apt-get update && \
    apt-get install -y curl git && \
    apt-get clean

# Install Foundry
RUN curl -L https://foundry.paradigm.xyz | bash && \
    /root/.foundry/bin/foundryup

ENV PATH="/root/.foundry/bin:$PATH"

# Clone contracts
WORKDIR /app

ARG CONTRACTS_REPO
ARG CONTRACTS_REF
ARG CONTRACTS_PATH="."

RUN git init && \
    git remote add origin ${CONTRACTS_REPO} && \
    git fetch --depth 1 origin ${CONTRACTS_REF} && \
    git checkout FETCH_HEAD && \
    git submodule update --init --recursive --depth 1 --single-branch

WORKDIR /app/${CONTRACTS_PATH}

RUN forge install
