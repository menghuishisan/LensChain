FROM ethereum/client-go:v1.14.12

EXPOSE 8545 8546 30303
ENTRYPOINT ["geth", "--dev", "--http", "--http.addr", "0.0.0.0", "--http.api", "eth,net,web3,personal,debug,txpool", "--ws", "--ws.addr", "0.0.0.0", "--ws.api", "eth,net,web3,debug", "--allow-insecure-unlock"]
