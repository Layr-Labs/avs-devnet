ARG EIGENLAYER_CONTRACTS_VERSION="v0.4.2-mainnet-pepe"

# Nightly (2024-10-03)
# Foundry doesn't support non-arm64 images, so we need to specify the platform
FROM ghcr.io/foundry-rs/foundry:nightly-471e4ac317858b3419faaee58ade30c0671021e0

WORKDIR /app

ARG EIGENLAYER_CONTRACTS_VERSION="v0.4.2-mainnet-pepe"

RUN git clone https://github.com/Layr-Labs/eigenlayer-contracts.git contracts \
    --single-branch --branch ${EIGENLAYER_CONTRACTS_VERSION} \
    --depth 1 --shallow-submodules --recurse-submodules

WORKDIR /app/contracts

# TODO: we can use a multi-stage build to store artifacts only
RUN forge build
