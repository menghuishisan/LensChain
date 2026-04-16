FROM rust:1.87-bookworm
WORKDIR /workspace
RUN apt-get update \
  && apt-get install -y curl git build-essential nodejs npm clang cmake protobuf-compiler \
  && rm -rf /var/lib/apt/lists/*
RUN rustup target add wasm32-unknown-unknown
ENTRYPOINT ["/bin/bash"]
