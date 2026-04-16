FROM node:22-bookworm-slim
WORKDIR /workspace
RUN apt-get update \
  && apt-get install -y curl git build-essential python3 python3-pip \
  && rm -rf /var/lib/apt/lists/*
RUN npm install -g hardhat@2.24.1 truffle@5.11.5 ganache@7.9.2 solc@0.8.30
RUN curl -L https://foundry.paradigm.xyz | bash && /root/.foundry/bin/foundryup
ENV PATH="/root/.foundry/bin:${PATH}"
ENTRYPOINT ["/bin/sh"]
