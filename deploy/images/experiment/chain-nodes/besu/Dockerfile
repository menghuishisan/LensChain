FROM hyperledger/besu:24.9.1
EXPOSE 8545 8546 30303
ENTRYPOINT ["besu", "--network=dev", "--rpc-http-enabled", "--rpc-http-host=0.0.0.0", "--rpc-ws-enabled", "--rpc-ws-host=0.0.0.0", "--host-allowlist=*"]
