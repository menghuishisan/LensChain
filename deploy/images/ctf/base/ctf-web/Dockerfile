FROM node:22-bookworm-slim
WORKDIR /workspace
RUN apt-get update \
  && apt-get install -y curl git build-essential python3 \
  && rm -rf /var/lib/apt/lists/*
RUN npm install -g npm@11.4.1 pnpm@10.8.1 yarn
EXPOSE 8080
ENTRYPOINT ["/bin/sh"]
