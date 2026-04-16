FROM trufflesuite/ganache:v7.9.2
EXPOSE 8545
ENTRYPOINT ["node", "/app/dist/node/cli.js", "--server.host", "0.0.0.0", "--server.port", "8545", "--wallet.totalAccounts", "20", "--chain.chainId", "31337"]
