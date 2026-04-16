FROM node:22-bookworm-slim
WORKDIR /app
RUN npm install -g hardhat@2.24.1
EXPOSE 8545
ENTRYPOINT ["npx", "hardhat", "node", "--hostname", "0.0.0.0", "--port", "8545"]
