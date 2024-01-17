# Megablocks - Example app KV store
This is an example application for the Megablocks project implementing a key value store application following the CometBFT guide example for application and CometBFT as separate processes.

## Build the application
To build the application run the following commands in the `app/kvstore`` directory
```
go get
go build
```
## Run the application
To run the applicaition
- Initialize CometBFT by running

    `go run github.com/cometbft/cometbft/cmd/cometbft@v0.38.0 init --home /tmp/cometbft-home`

- Run the application

    `./kvstore -kv-home /tmp/badger-home`

- In a separate terminal run CometBFT
    `go run github.com/cometbft/cometbft/cmd/cometbft@v0.38.0 node --home /tmp/cometbft-home --proxy_app=unix://example.sock`

- To submit a transaction run

    `curl -s 'localhost:26657/broadcast_tx_commit?tx="megablock=rocks"'`
- To check if the transaction was successful run

    `curl -s 'localhost:26657/abci_query?data="megablock"'`

    Returned key/value results are base64 encoded and needs to be translated using `base64 -d` to get expected results