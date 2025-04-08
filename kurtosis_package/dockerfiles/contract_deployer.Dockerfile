FROM debian:bookworm-slim as foundry

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

# Only clone if CONTRACTS_REPO is not empty
ARG CONTRACTS_REPO=""
ARG CONTRACTS_REF=""
ARG CONTRACTS_PATH="."

RUN if [ -n "$CONTRACTS_REPO" ]; then \
    git init && \
    git remote add origin "$CONTRACTS_REPO" && \
    git fetch --depth 1 origin "$CONTRACTS_REF" && \
    git checkout FETCH_HEAD && \
    git submodule update --init --recursive --depth 1 --single-branch; \
fi


# Change to contracts path and install dependencies
WORKDIR /app/${CONTRACTS_PATH}
RUN if [ -d ".git" ]; then forge install; else echo "Skipping forge install (no .git)"; fi
