package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
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
	logger  cmtlog.Logger
	clients map[string]AbciHandler
	mux     http.ServeMux
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
		logger:  logger,
		clients: map[string]AbciHandler{},
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
	app.clients[id] = AbciHandler{
		ID:     id,
		client: client,
	}
	return nil
}

// Start connects to all registered applications
func (app *CometMux) Start() error {
	for _, client := range app.clients {
		if err := client.Connect(); err != nil {
			return fmt.Errorf("Error connecting to chain app %s: %v", client.ID, err)
		}
	}
	return nil
}

//
// ABCI++ Implementation of CometMux follows here
//

func (app *CometMux) Info(_ context.Context, info *abcitypes.RequestInfo) (*abcitypes.ResponseInfo, error) {
	return &abcitypes.ResponseInfo{}, nil
}

func (app *CometMux) Query(_ context.Context, req *abcitypes.RequestQuery) (*abcitypes.ResponseQuery, error) {
	return &abcitypes.ResponseQuery{}, nil
}

func (app *CometMux) CheckTx(ctx context.Context, check *abcitypes.RequestCheckTx) (*abcitypes.ResponseCheckTx, error) {

	response := &abcitypes.ResponseCheckTx{}
	for _, client := range app.clients {
		cl := client.client
		app.logger.Debug(" @@@@ client is runnint :", cl.IsRunning())
		//appctx := context.Background()
		appctx := ctx
		response, err := cl.CheckTx(appctx, check)
		if err != nil {
			app.logger.Error("error forwarding CheckTx: ", err)
			return nil, err
		}
		return response, err
	}
	app.logger.Debug("no app registered to receive CheckTx")
	return response, nil
}

func (app *CometMux) InitChain(_ context.Context, chain *abcitypes.RequestInitChain) (*abcitypes.ResponseInitChain, error) {
	return &abcitypes.ResponseInitChain{}, nil
}

func (app *CometMux) PrepareProposal(_ context.Context, proposal *abcitypes.RequestPrepareProposal) (*abcitypes.ResponsePrepareProposal, error) {
	//TODO forward this to app
	response := abcitypes.ResponsePrepareProposal{Txs: proposal.Txs}
	app.logger.Info("@@@@ prepare proposal called. #Txs: ", len(response.Txs))
	return &response, nil
}

func (app *CometMux) ProcessProposal(_ context.Context, proposal *abcitypes.RequestProcessProposal) (*abcitypes.ResponseProcessProposal, error) {
	response := abcitypes.ResponseProcessProposal{Status: abcitypes.ResponseProcessProposal_ACCEPT}
	app.logger.Info("@@@@ process proposal called. response: ", response.String())
	return &response, nil
}

// Finalize Block: combination of BeginBlock, DeliverTx and EndBlock
func (app *CometMux) FinalizeBlock(_ context.Context, req *abcitypes.RequestFinalizeBlock) (*abcitypes.ResponseFinalizeBlock, error) {
	app.logger.Info("@@@ Finalize Block called: ", req.String())
	return &abcitypes.ResponseFinalizeBlock{}, nil
}

func (app CometMux) Commit(_ context.Context, commit *abcitypes.RequestCommit) (*abcitypes.ResponseCommit, error) {
	return &abcitypes.ResponseCommit{}, nil
}

func (app *CometMux) ListSnapshots(_ context.Context, snapshots *abcitypes.RequestListSnapshots) (*abcitypes.ResponseListSnapshots, error) {
	return &abcitypes.ResponseListSnapshots{}, nil
}

func (app *CometMux) OfferSnapshot(_ context.Context, snapshot *abcitypes.RequestOfferSnapshot) (*abcitypes.ResponseOfferSnapshot, error) {
	return &abcitypes.ResponseOfferSnapshot{}, nil
}

func (app *CometMux) LoadSnapshotChunk(_ context.Context, chunk *abcitypes.RequestLoadSnapshotChunk) (*abcitypes.ResponseLoadSnapshotChunk, error) {
	return &abcitypes.ResponseLoadSnapshotChunk{}, nil
}

func (app *CometMux) ApplySnapshotChunk(_ context.Context, chunk *abcitypes.RequestApplySnapshotChunk) (*abcitypes.ResponseApplySnapshotChunk, error) {
	return &abcitypes.ResponseApplySnapshotChunk{Result: abcitypes.ResponseApplySnapshotChunk_ACCEPT}, nil
}

func (app CometMux) ExtendVote(_ context.Context, extend *abcitypes.RequestExtendVote) (*abcitypes.ResponseExtendVote, error) {
	return &abcitypes.ResponseExtendVote{}, nil
}

func (app *CometMux) VerifyVoteExtension(_ context.Context, verify *abcitypes.RequestVerifyVoteExtension) (*abcitypes.ResponseVerifyVoteExtension, error) {
	return &abcitypes.ResponseVerifyVoteExtension{}, nil
}
