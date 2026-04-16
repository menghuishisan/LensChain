FROM rust:1.87-bookworm
WORKDIR /workspace
RUN apt-get update \
  && apt-get install -y curl git build-essential nodejs npm python3 \
  && rm -rf /var/lib/apt/lists/*
RUN cargo install --git https://github.com/coral-xyz/anchor avm --locked
ENTRYPOINT ["/bin/bash"]
