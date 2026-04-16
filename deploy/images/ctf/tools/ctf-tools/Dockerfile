FROM python:3.12-slim
WORKDIR /workspace
RUN apt-get update \
  && apt-get install -y curl git netcat-openbsd dnsutils iputils-ping jq vim-tiny build-essential \
  && rm -rf /var/lib/apt/lists/*
RUN pip install --no-cache-dir pwntools requests web3 eth-account
ENTRYPOINT ["/bin/sh"]
