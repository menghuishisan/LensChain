FROM golang:1.24-bookworm
WORKDIR /workspace
RUN apt-get update \
  && apt-get install -y git build-essential curl \
  && rm -rf /var/lib/apt/lists/*
ENTRYPOINT ["/bin/bash"]
