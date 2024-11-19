FROM debian:bookworm-slim AS foundry

# Install curl & git
RUN apt update -y && \
    apt upgrade -y && \
    apt install -y curl git && \
    apt clean

# Install foundry
RUN curl -L https://foundry.paradigm.xyz | bash
ENV PATH="/root/.foundry/bin:${PATH}"

RUN foundryup

WORKDIR /app/


# The repository's URL.
ARG CONTRACTS_REPO
# The commit hash, tag, or branch to checkout from the repo.
ARG CONTRACTS_REF
# Path to the contracts directory within the repository.
# It must contain a foundry.toml file.
ARG CONTRACTS_PATH="."

FROM foundry AS contract_deployer

WORKDIR /

ARG CONTRACTS_REPO
ARG CONTRACTS_REF
ARG CONTRACTS_PATH

RUN git init app

WORKDIR /app

RUN git remote add origin ${CONTRACTS_REPO} && \
    git fetch --depth 1 origin ${CONTRACTS_REF} && \
    git checkout FETCH_HEAD && \
    git submodule update --init --recursive --depth 1 --single-branch

WORKDIR /app/${CONTRACTS_PATH}

# TODO: we can use a multi-stage build to store artifacts only
RUN forge build
