FROM node:22-bookworm-slim
WORKDIR /app
RUN npm install -g @remix-project/remixd
EXPOSE 8080
ENTRYPOINT ["npx", "http-server", "/app", "-p", "8080"]
