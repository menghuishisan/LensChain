FROM node:22-bookworm-slim
WORKDIR /workspace
RUN npm install -g npm@11.4.1 pnpm@10.8.1
RUN npm install -g create-next-app@15.3.0
RUN npm install -g yarn
RUN mkdir -p /workspace/app
WORKDIR /workspace/app
ENTRYPOINT ["/bin/sh"]
