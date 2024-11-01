FROM debian:bookworm-slim

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
