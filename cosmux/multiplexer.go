package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"

	abcicli "github.com/cometbft/cometbft/abci/client"
	abcitypes "github.com/cometbft/cometbft/abci/types"
	cfg "github.com/cometbft/cometbft/config"
	cmtflags "github.com/cometbft/cometbft/libs/cli/flags"
	cmtlog "github.com/cometbft/cometbft/libs/log"
	"github.com/cometbft/cometbft/proxy"
)

// MAGIC used in the header of each valid Megablocks transaction
var (
	MAGIC           = [...]byte{0x23, 0x6d, 0x75, 0x78}
	MbHeaderLen int = len(MAGIC) + 1 // MAGIC + AppID (1byte)
)

// CometMux is an ABCI++ block multiplexer
type CometMux struct {
	log     cmtlog.Logger
	clients map[uint]*AbciHandler
	cfg     CosmuxConfig
}

type AbciHandler struct {
	ID       uint8 // unique application identifier
	ChainID  string
	client   abcicli.Client
	logLevel string
}

// Connect creates the client and connects to the chain application
func (hdl *AbciHandler) Connect() error {
	logger := cmtlog.NewTMLogger(cmtlog.NewSyncWriter(os.Stdout)).With("module",
		fmt.Sprintf("app-%d", hdl.ID))
	logger, err := cmtflags.ParseLogLevel(hdl.logLevel, logger, cfg.DefaultLogLevel)
	if err != nil {
		return err
	}
	hdl.client.SetLogger(logger)

	// Start client
	if hdl.client.IsRunning() {
		logger.Info("Client already running")
		return nil
	}

	if err := hdl.client.Start(); err != nil {
		return fmt.Errorf("error starting client %d: %v", hdl.ID, err.Error())
	}

	logger.Info("Connected")
	return nil
}

type CosmuxConfig struct {
	LogLevel string
}

// Check API compliance
var _ abcitypes.Application = (*CometMux)(nil)

func NewMultiplexer(config CosmuxConfig) *CometMux {
	logger := cmtlog.NewTMLogger(cmtlog.NewSyncWriter(os.Stdout))
	logger, err := cmtflags.ParseLogLevel(config.LogLevel, logger, cfg.DefaultLogLevel)
	if err != nil {
		log.Fatalf("failed to parse log level: %v", err)
	}
	m := CometMux{
		log:     logger,
		clients: map[uint]*AbciHandler{},
		cfg:     config,
	}
	return &m
}

// AddApplication adds a chain application to the multiplexer
func (mux *CometMux) AddApplication(app MegaBlockApp) error {
	_, exists := mux.clients[uint(app.ID)]
	if exists {
		log.Fatal("handler exists already with ID", app.ID)
	}
	client, err := proxy.NewRemoteClientCreator(app.Address, app.ConnectionType, true).NewABCIClient()
	if err != nil {
		return err
	}
	mux.clients[uint(app.ID)] = &AbciHandler{
		ID:       app.ID,
		ChainID:  app.ChainID,
		client:   client,
		logLevel: mux.cfg.LogLevel,
	}
	return nil
}

// Start connects to all registered applications
func (mux *CometMux) Start() error {
	for _, client := range mux.clients {
		if err := client.Connect(); err != nil {
			return fmt.Errorf("error connecting to chain app %d: %v", client.ID, err)
		}
	}
	return nil
}

func (mux *CometMux) getHandler(header []byte) (*AbciHandler, error) {
	if err := CheckHeader(header); err != nil {
		mux.log.Error(err.Error())
		return nil, err
	}
	id := uint(header[MbHeaderLen-1])
	if _, exists := mux.clients[id]; !exists {
		// Header references invalid application which should never happen
		return nil, fmt.Errorf("no application handler found in megablocks header: len=%d, %v",
			len(header), header)
	}
	return mux.clients[id], nil
}

// CheckHeader verifies if the tx contains a valid Megablocks header
func CheckHeader(tx []byte) error {
	if len(tx) < MbHeaderLen {
		return fmt.Errorf("invalid tx header length: %d", len(tx))
	}
	// check Magic
	if !bytes.Equal(tx[:MbHeaderLen-1], MAGIC[:]) {
		return fmt.Errorf("invalid Megablocks tx header: %v", tx[:MbHeaderLen-1])
	}
	return nil
}

//
// ABCI++ Implementation of CometMux follows here
//

// Info calls are not forwarded to chain apps
func (mux *CometMux) Info(ctx context.Context, info *abcitypes.RequestInfo) (*abcitypes.ResponseInfo, error) {
	mux.log.Debug("Info called: ", info)
	response := abcitypes.ResponseInfo{}
	var err error = nil
	for _, clt := range mux.clients {
		resp, rc := clt.client.Info(ctx, info)
		if rc != nil {
			err = rc
		} else {
			// TODO: LastBlock Apphash for multi-apps
			response = *resp
		}
	}
	return &response, err
}

// Query relays a query to the corresponding application
func (mux *CometMux) Query(ctx context.Context, req *abcitypes.RequestQuery) (*abcitypes.ResponseQuery, error) {
	mux.log.Debug("Query called: ", req.String())

	// TODO : to be defined how to identify the target application on queries
	// use hardcoded handler for now
	hdlr, err := mux.getHandler(req.Data[:MbHeaderLen])
	if err != nil {
		mux.log.Error("call to Query failed: no handler found to forward call: %v", err)
		return nil, fmt.Errorf("query failed: %v", err)
	}

	// Strip Megablocks header
	req.Data = req.Data[MbHeaderLen:]
	cl := hdlr.client
	response, err := cl.Query(ctx, req)
	if err != nil {
		mux.log.Error("error forwarding Query: ", err)
		return nil, err
	}
	return response, err
}

// CheckTx will identify the target app based on the megablocks header and forward it to the app
func (mux *CometMux) CheckTx(ctx context.Context, check *abcitypes.RequestCheckTx) (*abcitypes.ResponseCheckTx, error) {
	mux.log.Info("CheckTx called: ", check)
	hdlr, err := mux.getHandler(check.Tx[:MbHeaderLen])
	if err != nil {
		mux.log.Error("call to CheckTx failed: %v", err)
		return nil, fmt.Errorf("CheckTx failed: %s", err.Error())
	}

	// Strip MB header
	check.Tx = check.Tx[MbHeaderLen:]
	cl := hdlr.client
	response, err := cl.CheckTx(ctx, check)
	if err != nil {
		mux.log.Error("error forwarding CheckTx: ", err)
		return nil, err
	}
	return response, err
}

// InitChain
func (mux *CometMux) InitChain(ctx context.Context, chain *abcitypes.RequestInitChain) (*abcitypes.ResponseInitChain, error) {
	mux.log.Debug("InitChain called: ", chain.String())
	// TODO: dispatching logic to be decided as this one contains validator set. for now it's a noop\
	response := &abcitypes.ResponseInitChain{}
	var err error = nil
	//chainID := chain.ChainId
	for _, client := range mux.clients {
		/* 		chainReq := *chain
		   		chainReq.ChainId = client.ChainID
		*/
		chain.ChainId = client.ChainID
		resp, rc := client.client.InitChain(ctx, chain)
		if rc != nil {
			mux.log.Error("error on InitChain from %s: %s", client.ChainID, rc.Error())
			err = rc
		} else {
			mux.log.Debug("InitResponse from chain %s: %+v", client.ChainID, resp)
			response = resp //TODO: needs to be adapted to multiple chains as APPHASH is part of this response
		}
	}
	return response, err
}

func (mux *CometMux) PrepareProposal(_ context.Context, proposal *abcitypes.RequestPrepareProposal) (*abcitypes.ResponsePrepareProposal, error) {
	// TODO: to be decided if app should get the possibility to regroup this
	response := abcitypes.ResponsePrepareProposal{Txs: proposal.Txs}
	mux.log.Debug("PrepareProposal called #Txs: ", len(response.Txs))
	return &response, nil
}

// ProcessProposal allows applications to check if proposed block is valid
func (mux *CometMux) ProcessProposal(_ context.Context, proposal *abcitypes.RequestProcessProposal) (*abcitypes.ResponseProcessProposal, error) {
	// TODO: to be decided if app should get the ability to check that and outcome of 'Atomic IBC'
	response := abcitypes.ResponseProcessProposal{Status: abcitypes.ResponseProcessProposal_ACCEPT}
	mux.log.Debug("ProcessProposal called. response: ", response.String())
	return &response, nil
}

// Finalize Block: combination of BeginBlock, DeliverTx and EndBlock
//
// Note: FinalizeBlock only prepares the update to be made and does not change the state of the application.
// The state change is actually committed in a later stage i.e. in commit phase.
func (mux *CometMux) FinalizeBlock(ctx context.Context, req *abcitypes.RequestFinalizeBlock) (*abcitypes.ResponseFinalizeBlock, error) {
	mux.log.Debug("Finalize Block called: #Txs=", len(req.Txs), "req=", req.String())

	// TODO: triage txs per app and forward them
	txResults := []*abcitypes.ExecTxResult{}
	handlerTxs := map[uint8]([][]byte){}
	for _, hdlr := range mux.clients {
		handlerTxs[hdlr.ID] = [][]byte{}
	}

	for idx := range req.Txs {
		hdlr, err := mux.getHandler(req.Txs[idx])
		if err != nil {
			mux.log.Error("call to FinalizeBlock failed:%v", err)
			return nil, fmt.Errorf("no handler found for call")
		}
		// Add stripped transaction to handlers Tx set
		handlerTxs[hdlr.ID] = append(handlerTxs[hdlr.ID], req.Txs[idx][MbHeaderLen:])
	}

	// Send transactions to dedicated application
	// TODO: figure out apphash, ordered responses, validator updates,...
	appHashes := []byte{}
	response := abcitypes.ResponseFinalizeBlock{}
	for hdlrID, txs := range handlerTxs {
		newReq := *req
		newReq.Txs = txs
		mux.log.Debug("Forwarding FinalizeBlock to ", hdlrID)
		appResp, err := mux.clients[uint(hdlrID)].client.FinalizeBlock(ctx, &newReq)
		if err != nil {
			mux.log.Error("call to FinalizeBlock failed: %v", err)
			return nil, err
		}
		txResults = append(txResults, appResp.GetTxResults()...)
		appHashes = appResp.AppHash
	}

	// add aggregated txResuls
	// TODO: This needs to be ordered according to the txs of the original request
	response.TxResults = txResults
	response.AppHash = appHashes
	return &response, nil
}

// Commit sends commit to all apps
func (mux CometMux) Commit(ctx context.Context, commit *abcitypes.RequestCommit) (*abcitypes.ResponseCommit, error) {
	mux.log.Debug("Commit called: ")
	for _, hdlr := range mux.clients {
		cl := hdlr.client
		_, err := cl.Commit(ctx, commit)
		if err != nil {
			mux.log.Error("error forwarding Commit: ", err)
			return nil, err
		}
	}
	return &abcitypes.ResponseCommit{}, nil
}

func (mux *CometMux) ListSnapshots(_ context.Context, snapshots *abcitypes.RequestListSnapshots) (*abcitypes.ResponseListSnapshots, error) {
	mux.log.Debug("ListSnapshots called: ", snapshots.String())
	return &abcitypes.ResponseListSnapshots{}, nil
}

func (mux *CometMux) OfferSnapshot(_ context.Context, snapshot *abcitypes.RequestOfferSnapshot) (*abcitypes.ResponseOfferSnapshot, error) {
	mux.log.Debug("OfferSnapshots called: ", snapshot.String())
	return &abcitypes.ResponseOfferSnapshot{}, nil
}

func (mux *CometMux) LoadSnapshotChunk(_ context.Context, chunk *abcitypes.RequestLoadSnapshotChunk) (*abcitypes.ResponseLoadSnapshotChunk, error) {
	mux.log.Debug("LoadSnapshots called: ", chunk.String())
	return &abcitypes.ResponseLoadSnapshotChunk{}, nil
}

func (mux *CometMux) ApplySnapshotChunk(_ context.Context, chunk *abcitypes.RequestApplySnapshotChunk) (*abcitypes.ResponseApplySnapshotChunk, error) {
	mux.log.Debug("ApplySnapshots called: ", chunk.String())
	return &abcitypes.ResponseApplySnapshotChunk{Result: abcitypes.ResponseApplySnapshotChunk_ACCEPT}, nil
}

func (mux CometMux) ExtendVote(_ context.Context, extend *abcitypes.RequestExtendVote) (*abcitypes.ResponseExtendVote, error) {
	mux.log.Debug("ExtendVote called: ", extend.String())
	return &abcitypes.ResponseExtendVote{}, nil
}

func (mux *CometMux) VerifyVoteExtension(_ context.Context, verify *abcitypes.RequestVerifyVoteExtension) (*abcitypes.ResponseVerifyVoteExtension, error) {
	mux.log.Debug("VerifyVoteExtension called: ", verify.String())
	return &abcitypes.ResponseVerifyVoteExtension{}, nil
}
