FROM debian:bookworm-slim

# Install required tools
RUN apt update && \
    apt install -y curl git && \
    apt clean

# Install Foundry
RUN curl -L https://foundry.paradigm.xyz | bash && \
    /root/.foundry/bin/foundryup

ENV PATH="/root/.foundry/bin:$PATH"

# Create /app and make it writable
RUN mkdir -p /app && chmod -R 777 /app

# Use /app as working directory
WORKDIR /app

# The repository's URL.
ARG CONTRACTS_REPO
# The commit hash, tag, or branch to checkout from the repo.
ARG CONTRACTS_REF
# Path to the contracts directory within the repository.
# It must contain a foundry.toml file.
ARG CONTRACTS_PATH="."

# Clone contracts
RUN git init && \
    git remote add origin ${CONTRACTS_REPO} && \
    git fetch --depth 1 origin ${CONTRACTS_REF} && \
    git checkout FETCH_HEAD && \
    git submodule update --init --recursive --depth 1 --single-branch

# Change to contracts path and install dependencies
WORKDIR /app/${CONTRACTS_PATH}
RUN forge install
