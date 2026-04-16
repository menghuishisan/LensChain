FROM rust:1.87-bookworm
WORKDIR /workspace
RUN apt-get update \
  && apt-get install -y curl git build-essential python3 nodejs npm \
  && rm -rf /var/lib/apt/lists/*
RUN cargo install --git https://github.com/aptos-labs/aptos-core.git aptos --locked
ENTRYPOINT ["/bin/bash"]
