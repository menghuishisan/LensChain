FROM ubuntu:24.04
WORKDIR /workspace
RUN apt-get update \
  && apt-get install -y curl wget git python3 python3-pip unzip file jq netcat-openbsd \
  && rm -rf /var/lib/apt/lists/*
ENTRYPOINT ["/bin/bash"]
