FROM ruimarinho/bitcoin-core:26
EXPOSE 8332 8333 18332
ENTRYPOINT ["bitcoind", "-printtoconsole", "-regtest=1", "-server=1", "-rpcbind=0.0.0.0", "-rpcallowip=0.0.0.0/0"]
