package main

import (
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

// CometMux is an ABCI++ block multiplexer
type CometMux struct {
	log     cmtlog.Logger
	clients map[string]*AbciHandler
}

type AbciHandler struct {
	ID     string
	client abcicli.Client
}

// Connect creates the client and connects to the chain application
func (hdl *AbciHandler) Connect() error {
	logger := cmtlog.NewTMLogger(cmtlog.NewSyncWriter(os.Stdout)).With("module",
		fmt.Sprintf("app-%s", hdl.ID))
	hdl.client.SetLogger(logger)

	// Start client
	if hdl.client.IsRunning() {
		logger.Info("Client already running")
		return nil
	}

	if err := hdl.client.Start(); err != nil {
		return fmt.Errorf("error starting client %s: %v", hdl.ID, err.Error())
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
		clients: map[string]*AbciHandler{},
	}
	return &m
}

// AddApplication adds a chain application to the multiplexer
func (app *CometMux) AddApplication(addr, transport string) error {
	id := addr + transport
	_, exists := app.clients[id]
	if exists {
		log.Fatal("handler exists already with ID", id)
	}
	client, err := proxy.NewRemoteClientCreator(addr, transport, true).NewABCIClient()
	if err != nil {
		return err
	}
	app.clients[id] = &AbciHandler{
		ID:     id,
		client: client,
	}
	return nil
}

// Start connects to all registered applications
func (app *CometMux) Start() error {
	for _, client := range app.clients {
		if err := client.Connect(); err != nil {
			return fmt.Errorf("error connecting to chain app %s: %v", client.ID, err)
		}
	}
	return nil
}

func (app *CometMux) getHandler() *AbciHandler {
	for idx := range app.clients {
		return app.clients[idx]
	}
	return nil
}

//
// ABCI++ Implementation of CometMux follows here
//

func (app *CometMux) Info(ctx context.Context, info *abcitypes.RequestInfo) (*abcitypes.ResponseInfo, error) {
	app.log.Info("@@@ Info called: ", info)
	hdlr := app.getHandler()
	if hdlr == nil {
		app.log.Error("call to Info failed: no handler found to forward call")
		return nil, fmt.Errorf("no handler found for call")
	}
	cl := hdlr.client
	response, err := cl.Info(ctx, info)
	if err != nil {
		app.log.Error("error forwarding Info: ", err)
		return nil, err
	}
	return response, nil
}

func (app *CometMux) Query(ctx context.Context, req *abcitypes.RequestQuery) (*abcitypes.ResponseQuery, error) {
	app.log.Info("@@@ Query called: ", req.String())
	hdlr := app.getHandler()
	if hdlr == nil {
		app.log.Error("call to Query failed: no handler found to forward call")
		return nil, fmt.Errorf("no handler found for call")
	}
	cl := hdlr.client
	response, err := cl.Query(ctx, req)
	if err != nil {
		app.log.Error("error forwarding Query: ", err)
		return nil, err
	}
	return response, err
}

func (app *CometMux) CheckTx(ctx context.Context, check *abcitypes.RequestCheckTx) (*abcitypes.ResponseCheckTx, error) {
	app.log.Info("@@@ CheckTx called: ", check)
	hdlr := app.getHandler()
	if hdlr == nil {
		app.log.Error("call to CheckTx failed: no handler found to forward call")
		return nil, fmt.Errorf("no handler found for call")
	}
	cl := hdlr.client
	response, err := cl.CheckTx(ctx, check)
	if err != nil {
		app.log.Error("error forwarding CheckTx: ", err)
		return nil, err
	}
	return response, err
}

func (app *CometMux) InitChain(ctx context.Context, chain *abcitypes.RequestInitChain) (*abcitypes.ResponseInitChain, error) {
	app.log.Info("@@@ InitChain called: ", chain.String())
	hdlr := app.getHandler()
	if hdlr == nil {
		app.log.Error("call to InitChain failed: no handler found to forward call")
		return nil, fmt.Errorf("no handler found for call")
	}
	cl := hdlr.client
	response, err := cl.InitChain(ctx, chain)
	if err != nil {
		app.log.Error("error forwarding InitChain: ", err)
		return nil, err
	}
	return response, err
}

func (app *CometMux) PrepareProposal(_ context.Context, proposal *abcitypes.RequestPrepareProposal) (*abcitypes.ResponsePrepareProposal, error) {
	// TODO: to be decided if app should get the possibility to regroup this
	response := abcitypes.ResponsePrepareProposal{Txs: proposal.Txs}
	app.log.Info("@@@@ PrepareProposal called #Txs: ", len(response.Txs))
	return &response, nil
}

// ProcessProposal allows applications to check if proposed block is valid
func (app *CometMux) ProcessProposal(_ context.Context, proposal *abcitypes.RequestProcessProposal) (*abcitypes.ResponseProcessProposal, error) {
	// TODO: to be decided if app should get the ability to check that and outcome of 'Atomic IBC'
	response := abcitypes.ResponseProcessProposal{Status: abcitypes.ResponseProcessProposal_ACCEPT}
	app.log.Info("@@@@ ProcessProposal called. response: ", response.String())
	return &response, nil
}

// Finalize Block: combination of BeginBlock, DeliverTx and EndBlock
//
// Note: FinalizeBlock only prepares the update to be made and does not change the state of the application.
// The state change is actually committed in a later stage i.e. in commit phase.
func (app *CometMux) FinalizeBlock(ctx context.Context, req *abcitypes.RequestFinalizeBlock) (*abcitypes.ResponseFinalizeBlock, error) {
	app.log.Info("@@@ Finalize Block called: ", req.String())
	hdlr := app.getHandler()
	if hdlr == nil {
		app.log.Error("call to FinalizeBlock failed: no handler found to forward call")
		return nil, fmt.Errorf("no handler found for call")
	}
	cl := hdlr.client
	response, err := cl.FinalizeBlock(ctx, req)
	if err != nil {
		app.log.Error("error forwarding FinalizeBlock: ", err)
		return nil, err
	}
	return response, err
}

func (app CometMux) Commit(ctx context.Context, commit *abcitypes.RequestCommit) (*abcitypes.ResponseCommit, error) {
	app.log.Info("@@@ Commit called: ")
	hdlr := app.getHandler()
	if hdlr == nil {
		app.log.Error("call to Commit failed: no handler found to forward call")
		return nil, fmt.Errorf("no handler found for call")
	}
	cl := hdlr.client
	response, err := cl.Commit(ctx, commit)
	if err != nil {
		app.log.Error("error forwarding Commit: ", err)
		return nil, err
	}
	return response, err
}

func (app *CometMux) ListSnapshots(_ context.Context, snapshots *abcitypes.RequestListSnapshots) (*abcitypes.ResponseListSnapshots, error) {
	app.log.Info("@@@ ListSnapshots called: ", snapshots.String())
	return &abcitypes.ResponseListSnapshots{}, nil
}

func (app *CometMux) OfferSnapshot(_ context.Context, snapshot *abcitypes.RequestOfferSnapshot) (*abcitypes.ResponseOfferSnapshot, error) {
	app.log.Info("@@@ OfferSnapshots called: ", snapshot.String())
	return &abcitypes.ResponseOfferSnapshot{}, nil
}

func (app *CometMux) LoadSnapshotChunk(_ context.Context, chunk *abcitypes.RequestLoadSnapshotChunk) (*abcitypes.ResponseLoadSnapshotChunk, error) {
	app.log.Info("@@@ LoadSnapshots called: ", chunk.String())
	return &abcitypes.ResponseLoadSnapshotChunk{}, nil
}

func (app *CometMux) ApplySnapshotChunk(_ context.Context, chunk *abcitypes.RequestApplySnapshotChunk) (*abcitypes.ResponseApplySnapshotChunk, error) {
	app.log.Info("@@@ ApplySnapshots called: ", chunk.String())
	return &abcitypes.ResponseApplySnapshotChunk{Result: abcitypes.ResponseApplySnapshotChunk_ACCEPT}, nil
}

func (app CometMux) ExtendVote(_ context.Context, extend *abcitypes.RequestExtendVote) (*abcitypes.ResponseExtendVote, error) {
	app.log.Info("@@@ ExtendVote called: ", extend.String())
	return &abcitypes.ResponseExtendVote{}, nil
}

func (app *CometMux) VerifyVoteExtension(_ context.Context, verify *abcitypes.RequestVerifyVoteExtension) (*abcitypes.ResponseVerifyVoteExtension, error) {
	app.log.Info("@@@ VerifyVoteExtension called: ", verify.String())
	return &abcitypes.ResponseVerifyVoteExtension{}, nil
}
