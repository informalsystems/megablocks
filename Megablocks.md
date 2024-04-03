# Megablocks Spike-Implementation

This is the spike implementation of Megablocks. In the following chapters we give a brief overview
of Megablocks approach, the design decision taken and discuss the limitations and known issues of the prototype.

## Architecture
This spike implements the Megablocks approach as described in [this](https://forum.cosmos.network/t/chips-discussion-phase-atomic-ibc-megablocks/11767) CHIPs discussion.

The multiplexer shim is implemented as a built-in application of CometBFT serving CometBFT on the 'Southbound' interface. On the 'Northbound' interface the multiplexer connects to the registered chain applications.
North and Southbound interface are implementing ABCI++.

## Transactions & Queries

Transactions and queries need to be marked in order to dispatch the request to the correct chain application by the multiplexer. A Megablocks-header needs to be prepended to each transaction. This Megablocks-header consists of a Magic value and a chain-app identifier and needs to be provided by the application sending the transaction to CometBFT broadcast interface. The multiplexer strips the Megablocks-header before sending the transaction to the correct chain application.

For queries a new ABCI Query Option 'chain-id' was introduced to tag the target chain application the query should be forwarded to by the multiplexer.

For this spike implementation the needed changes on CometBFT and Cosmos-SDK side were implemented on a fork of these repositories. The modified implementations of CometBFT and Cosmos-SDK are staged in the ./cosmos directory of Megablocks implementation.

## Known Limitations

1) ABCI++: Not all parts of the ABCI++ interface are supported, e.g. PrepareProposal and Snapshot-related APIs are not supported at the current version of the multiplexer implementation
2) App-hash errors and other erroneous behavior of registered chain-apps needs special handling by the multiplexer which is not supported at the moment.
3) Support of Validators in Megablocks environment needs to be defined and implemented
4) Handling of consensus parameters across different chain apps need to be defined and implemented
5) Current implementation was tested with 2 chain applications (sdk and non-sdk based) simultaneously