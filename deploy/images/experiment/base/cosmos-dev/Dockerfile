FROM golang:1.24-bookworm
WORKDIR /workspace
RUN apt-get update \
  && apt-get install -y curl git build-essential nodejs npm \
  && rm -rf /var/lib/apt/lists/*
RUN curl https://get.ignite.com/cli! | bash
ENTRYPOINT ["/bin/bash"]
