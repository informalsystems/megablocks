package main

//
// KVStoreApplication implementing the ABCI interface as described in the user guide
// of CometBFT v0.38
//

import (
	"bytes"
	"context"
	"fmt"
	"log"

	abcitypes "github.com/cometbft/cometbft/abci/types"
	cmtlog "github.com/cometbft/cometbft/libs/log"
	"github.com/dgraph-io/badger/v4"
)

type KVStoreApplication struct {
	db           *badger.DB
	onGoingBlock *badger.Txn
	log          cmtlog.Logger
}

var _ abcitypes.Application = (*KVStoreApplication)(nil)

func NewKVStoreApplication(db *badger.DB, logger cmtlog.Logger) *KVStoreApplication {
	return &KVStoreApplication{
		db:  db,
		log: logger.With("module", "kvstore"),
	}
}

func (app *KVStoreApplication) Info(_ context.Context, info *abcitypes.RequestInfo) (*abcitypes.ResponseInfo, error) {
	return &abcitypes.ResponseInfo{}, nil
}

func (app *KVStoreApplication) Query(_ context.Context, req *abcitypes.RequestQuery) (*abcitypes.ResponseQuery, error) {
	resp := abcitypes.ResponseQuery{Key: req.Data}

	dbErr := app.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(req.Data)
		if err != nil {
			if err != badger.ErrKeyNotFound {
				return err
			}
			resp.Log = "key does not exist"
			return nil
		}

		return item.Value(func(val []byte) error {
			resp.Log = "exists"
			resp.Value = val
			return nil
		})
	})
	if dbErr != nil {
		log.Panicf("Error reading database, unable to execute query: %v", dbErr)
	}
	return &resp, nil
}

func (app *KVStoreApplication) CheckTx(_ context.Context, check *abcitypes.RequestCheckTx) (*abcitypes.ResponseCheckTx, error) {
	code := app.isValid(check.Tx)
	app.log.Info("Check TX called :", check.String())
	return &abcitypes.ResponseCheckTx{Code: code}, nil
}

func (app *KVStoreApplication) InitChain(_ context.Context, chain *abcitypes.RequestInitChain) (*abcitypes.ResponseInitChain, error) {
	app.log.Info("InitChain called :", chain.String())
	return &abcitypes.ResponseInitChain{}, nil
}

// PrepareProposal allows application to reorder,add or remove transactions from the group.
// Here we're returning the unmodified group of transactions
func (app *KVStoreApplication) PrepareProposal(_ context.Context, proposal *abcitypes.RequestPrepareProposal) (*abcitypes.ResponsePrepareProposal, error) {
	app.log.Info("PrepareProposal called :", proposal.String())
	return &abcitypes.ResponsePrepareProposal{Txs: proposal.Txs}, nil
}

// ProcessProposal is used to ask the application to accept the proposal before voting to accept the proposal by the node.
// Here we're simply accept all proposals
func (app *KVStoreApplication) ProcessProposal(_ context.Context, proposal *abcitypes.RequestProcessProposal) (*abcitypes.ResponseProcessProposal, error) {
	app.log.Info("ProcessProposal called :", proposal.String())
	return &abcitypes.ResponseProcessProposal{Status: abcitypes.ResponseProcessProposal_ACCEPT}, nil
}

// FinalizeBlock will add the key and value to the Badger transaction every time
// our application processes a new application transaction from the list received through
// RequestFinalizeBlock
func (app *KVStoreApplication) FinalizeBlock(_ context.Context, req *abcitypes.RequestFinalizeBlock) (*abcitypes.ResponseFinalizeBlock, error) {
	app.log.Info("FinalizedBlock called :", req.String())
	txs := make([]*abcitypes.ExecTxResult, len(req.Txs))
	app.onGoingBlock = app.db.NewTransaction(true)
	for i, tx := range req.Txs {
		// check if tx is valid
		if code := app.isValid(tx); code != 0 {
			app.log.Error("Error: invalid transaction index %v", i)
			txs[i] = &abcitypes.ExecTxResult{Code: code}
		} else {
			parts := bytes.SplitN(tx, []byte("="), 2)
			key, value := parts[0], parts[1]
			app.log.Debug("Adding key %s with value %s", key, value)

			if err := app.onGoingBlock.Set(key, value); err != nil {
				log.Panicf("Error writing to database, unable to execute tx: %v", err)
			}

			app.log.Info("Successfully added key %s with value %s", key, value)

			txs[i] = &abcitypes.ExecTxResult{}
		}
	}

	return &abcitypes.ResponseFinalizeBlock{
		TxResults: txs,
	}, nil
}

func (app KVStoreApplication) Commit(_ context.Context, commit *abcitypes.RequestCommit) (*abcitypes.ResponseCommit, error) {
	// terminate badger transaction and make resulting state persistent
	app.log.Info("Commit called")
	if app.onGoingBlock == nil {
		return nil, fmt.Errorf("application error: no ongoing block from badger")
	}
	return &abcitypes.ResponseCommit{}, app.onGoingBlock.Commit()
}

func (app *KVStoreApplication) ListSnapshots(_ context.Context, snapshots *abcitypes.RequestListSnapshots) (*abcitypes.ResponseListSnapshots, error) {
	app.log.Info("ListSnapshots called: ", snapshots.String())
	return &abcitypes.ResponseListSnapshots{}, nil
}

func (app *KVStoreApplication) OfferSnapshot(_ context.Context, snapshot *abcitypes.RequestOfferSnapshot) (*abcitypes.ResponseOfferSnapshot, error) {
	app.log.Info("OfferSnapshots called: ", snapshot.String())
	return &abcitypes.ResponseOfferSnapshot{}, nil
}

func (app *KVStoreApplication) LoadSnapshotChunk(_ context.Context, chunk *abcitypes.RequestLoadSnapshotChunk) (*abcitypes.ResponseLoadSnapshotChunk, error) {
	app.log.Info("LoadSnapshots called: ", chunk.String())
	return &abcitypes.ResponseLoadSnapshotChunk{}, nil
}

func (app *KVStoreApplication) ApplySnapshotChunk(_ context.Context, chunk *abcitypes.RequestApplySnapshotChunk) (*abcitypes.ResponseApplySnapshotChunk, error) {
	app.log.Info("ApplySnapshots called: ", chunk.String())
	return &abcitypes.ResponseApplySnapshotChunk{Result: abcitypes.ResponseApplySnapshotChunk_ACCEPT}, nil
}

func (app KVStoreApplication) ExtendVote(_ context.Context, extend *abcitypes.RequestExtendVote) (*abcitypes.ResponseExtendVote, error) {
	app.log.Info("ExtendVote called: ", extend.String())
	return &abcitypes.ResponseExtendVote{}, nil
}

func (app *KVStoreApplication) VerifyVoteExtension(_ context.Context, verify *abcitypes.RequestVerifyVoteExtension) (*abcitypes.ResponseVerifyVoteExtension, error) {
	app.log.Info("VerifyVoteExtension called: ", verify.String())
	return &abcitypes.ResponseVerifyVoteExtension{}, nil
}

// isValid is check if a transaction is valid
func (app *KVStoreApplication) isValid(tx []byte) uint32 {
	// very basic check to ensure that the transaction conforms to key=value pattern
	parts := bytes.Split(tx, []byte("="))
	if len(parts) != 2 {
		return 1
	}
	return 0
}
