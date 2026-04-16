FROM python:3.12-slim
WORKDIR /workspace
RUN apt-get update \
  && apt-get install -y curl wget git jq netcat-openbsd python3-pip \
  && rm -rf /var/lib/apt/lists/*
RUN pip install --no-cache-dir requests web3 eth-account pwntools
ENTRYPOINT ["/bin/sh"]
