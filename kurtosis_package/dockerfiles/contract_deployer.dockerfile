# The repository's URL.
ARG CONTRACTS_REPO
# The commit hash, tag, or branch to checkout from the repo.
ARG CONTRACTS_REF
# Path to the contracts directory within the repository.
# It must contain a foundry.toml file.
ARG CONTRACTS_PATH="."

# Nightly (2024-10-03)
# Foundry doesn't support non-arm64 images, so we need to specify the platform
FROM --platform=amd64 ghcr.io/foundry-rs/foundry:nightly-471e4ac317858b3419faaee58ade30c0671021e0

WORKDIR /app

ARG CONTRACTS_REPO
ARG CONTRACTS_REF
ARG CONTRACTS_PATH

RUN git init contracts

WORKDIR /app/contracts

RUN git remote add origin ${CONTRACTS_REPO} && \
    git fetch --depth 1 origin ${CONTRACTS_REF} && \
    git checkout FETCH_HEAD && \
    git submodule update --init --recursive --depth 1 --single-branch --recursive

WORKDIR /app/contracts/${CONTRACTS_PATH}

# TODO: we can use a multi-stage build to store artifacts only
RUN forge build
