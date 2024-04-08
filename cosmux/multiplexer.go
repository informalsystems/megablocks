package main

import (
	"bytes"
	"context"
	"crypto/sha1"
	"fmt"
	"log"
	"os"
	"sort"
	"sync"

	"cosmossdk.io/api/tendermint/abci"
	abcicli "github.com/cometbft/cometbft/abci/client"
	abcitypes "github.com/cometbft/cometbft/abci/types"
	cfg "github.com/cometbft/cometbft/config"
	cmtflags "github.com/cometbft/cometbft/libs/cli/flags"
	cmtlog "github.com/cometbft/cometbft/libs/log"
	"github.com/cometbft/cometbft/proxy"
)

var (
	// MAGIC used in the header of each valid Megablocks transaction
	MAGIC = [...]byte{0x23, 0x6d, 0x75, 0x78}
	// MAGIC + 4byte-hash of ChainID
	ChainAppIdLen     = 4
	MbHeaderLen   int = len(MAGIC) + ChainAppIdLen
)

type ChainAppIdentifier [4]byte

type appIdStorter struct {
	identifiers []ChainAppIdentifier
}

// SortChainAppIDs sorts a list of ChainAppIdentifier
func SortChainAppIDs(ids []ChainAppIdentifier) {
	x := &appIdStorter{
		identifiers: ids,
	}
	sort.Sort(x)
}

// Sorting interface 'Less()' for appIdStorter
func (ci appIdStorter) Less(l, k int) bool {
	for i := 0; i < ChainAppIdLen; i++ {
		if ci.identifiers[l][i] == ci.identifiers[k][i] {
			continue
		}
		return ci.identifiers[l][i] < ci.identifiers[k][i]
	}
	return false
}

// Sorting interface 'Len()' for appIdStorter
func (ci appIdStorter) Len() int {
	return len(ci.identifiers)
}

// Sorting interface 'Swap' for appIdStorter
func (ci appIdStorter) Swap(i, j int) {
	ci.identifiers[i], ci.identifiers[j] = ci.identifiers[j], ci.identifiers[i]
}

// CometMux is an ABCI++ block multiplexer
type CometMux struct {
	log     cmtlog.Logger
	clients map[ChainAppIdentifier]*AbciHandler
	cfg     *CosmuxConfig
}

type AbciHandler struct {
	ID                ChainAppIdentifier // unique application identifier
	ChainID           string
	client            abcicli.Client
	logLevel          string
	InitAppStateBytes []byte
	InitValidators    []byte
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

func (hdl *AbciHandler) InitChain(ctx context.Context, chain *abcitypes.RequestInitChain) (*abcitypes.ResponseInitChain, error) {
	req := *chain
	req.ChainId = hdl.ChainID
	req.AppStateBytes = hdl.InitAppStateBytes
	// TBD: Decide validator setup for multi-chain.
	//      In this spike it's not an app specific setting but a multiplexer
	return hdl.client.InitChain(ctx, &req)
}

// Check API compliance
var _ abcitypes.Application = (*CometMux)(nil)

// Create Multiplexer and register chains
func NewMultiplexer(config *CosmuxConfig) *CometMux {
	logger := cmtlog.NewTMLogger(cmtlog.NewSyncWriter(os.Stdout)).With("module", "comet-mux")
	logger, err := cmtflags.ParseLogLevel(config.LogLevel, logger, cfg.DefaultLogLevel)
	if err != nil {
		log.Fatalf("failed to parse log level: %v", err)
	}
	m := CometMux{
		log:     logger,
		clients: map[ChainAppIdentifier]*AbciHandler{},
		cfg:     config,
	}

	// Register applications
	for _, app := range config.Apps {
		if err := m.AddApplication(app); err != nil {
			log.Fatalf("error registering chain application: %v", err)
		}
	}

	return &m
}

// AddApplication adds a chain application to the multiplexer
func (mux *CometMux) AddApplication(app MegaBlockApp) error {
	sha1Sum := sha1.Sum([]byte(app.ChainID))
	appId := ChainAppIdentifier(sha1Sum[:5])

	_, exists := mux.clients[appId]
	if exists {
		log.Fatal("handler exists already with ID", appId)
	} else {
		mux.log.Info(fmt.Sprintf("Adding handler for %s= %v", app.ChainID, appId))
	}
	client, err := proxy.NewRemoteClientCreator(app.Address, app.ConnectionType, true).NewABCIClient()
	if err != nil {
		return err
	}

	var appState = []byte{}
	if appState, err = GetInitialAppState(app.Home); err != nil {
		log.Fatalf("Error reading app state for '%s': %v", app.ChainID, err)
	}

	mux.clients[appId] = &AbciHandler{
		ID:                appId,
		ChainID:           app.ChainID,
		client:            client,
		logLevel:          mux.cfg.LogLevel,
		InitAppStateBytes: appState,
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
	// Check if tx has a valid megablocks header
	if err := CheckHeader(header); err != nil {
		mux.log.Error(err.Error())
		return nil, err
	}

	appId := ChainAppIdentifier(header[len(MAGIC):MbHeaderLen])
	if _, exists := mux.clients[appId]; !exists {
		return nil, fmt.Errorf("invalid chain reference in MB header: len=%d, %v",
			len(header), header)
	}
	return mux.clients[appId], nil
}

func (mux *CometMux) getHandlerFromChainId(chainID string) (*AbciHandler, error) {
	for _, client := range mux.clients {
		if client.ChainID == chainID {
			return client, nil
		}
	}
	return nil, fmt.Errorf("no application handler found for chain-id '%s'", chainID)
}

// CheckHeader verifies if the tx contains a valid Megablocks header
func CheckHeader(tx []byte) error {
	if len(tx) < MbHeaderLen {
		return fmt.Errorf("invalid tx header length: %d", len(tx))
	}
	// check Magic
	if !bytes.Equal(tx[:len(MAGIC)], MAGIC[:]) {
		return fmt.Errorf("invalid Megablocks tx header: %v", tx[:len(MAGIC)])
	}
	return nil
}

//
// ABCI++ Implementation of CometMux follows here
//

// Info calls are not forwarded to chain apps
func (mux *CometMux) Info(ctx context.Context, info *abcitypes.RequestInfo) (*abcitypes.ResponseInfo, error) {
	mux.log.Debug("Info called: ", "info", info)
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
	mux.log.Debug("Query called for: ", "chain-id", req.ChainId, "request", req)

	hdlr, err := mux.getHandlerFromChainId(req.ChainId)
	if err != nil {
		mux.log.Error("call to Query failed: no handler found to forward call", "error", err)
		return nil, fmt.Errorf("query failed: %v", err)
	}
	//req.Path = path[1]
	cl := hdlr.client
	response, err := cl.Query(ctx, req)
	if err != nil {
		mux.log.Error("error forwarding Query", "error", err)
		return nil, err
	}
	mux.log.Debug("Query result:", "chain-id", req.ChainId, "response", response)
	return response, err
}

// CheckTx will identify the target app based on the megablocks header and forward it to the app
func (mux *CometMux) CheckTx(ctx context.Context, check *abcitypes.RequestCheckTx) (*abcitypes.ResponseCheckTx, error) {
	mux.log.Info("CheckTx called: ", "type", check.Type, "length", len(check.Tx), "Tx", check.Tx)
	hdlr, err := mux.getHandler(check.Tx[:MbHeaderLen])
	if err != nil {
		mux.log.Error("call to CheckTx failed:", "error", err)
		return nil, fmt.Errorf("CheckTx failed: %s", err.Error())
	}

	// Strip MB header
	check.Tx = check.Tx[MbHeaderLen:]
	cl := hdlr.client
	response, err := cl.CheckTx(ctx, check)
	if err != nil {
		mux.log.Error("error forwarding CheckTx", "error", err)
		return nil, err
	}
	return response, err
}

// InitChain
func (mux *CometMux) InitChain(ctx context.Context, chain *abcitypes.RequestInitChain) (*abcitypes.ResponseInitChain, error) {
	mux.log.Debug("InitChain called", "chain-id", chain.ChainId, "request", chain)
	var response *abcitypes.ResponseInitChain
	var err error = nil

	type InitResponse struct {
		Response *abcitypes.ResponseInitChain
		Error    error
	}
	chResp := make(chan InitResponse, len(mux.clients))
	wg := sync.WaitGroup{}
	wg.Add(len(mux.clients))

	for _, client := range mux.clients {
		client := client
		go func() {
			defer wg.Done()
			resp, rc := client.InitChain(ctx, chain)
			chResp <- InitResponse{Response: resp, Error: rc}
		}()
	}

	go func() {
		wg.Wait()
		close(chResp)
	}()

	// loop on the channel until it's closed
	for resp := range chResp {
		if resp.Error != nil {
			mux.log.Error("Error on response from InitChain")
			return nil, resp.Error
		}
		mux.log.Debug("Response received", "resp", resp.Response)
		if response == nil {
			response = resp.Response
		} else {
			response.AppHash = append(response.AppHash, resp.Response.AppHash...)
		}

		// TBD: consensus parameters should be the same across chain apps
		if resp.Response.ConsensusParams != nil {
			response.ConsensusParams = resp.Response.ConsensusParams
		}
		response.Validators = append(response.Validators, resp.Response.Validators...)
	}

	return response, err
}

func (mux *CometMux) PrepareProposal(_ context.Context, proposal *abcitypes.RequestPrepareProposal) (*abcitypes.ResponsePrepareProposal, error) {
	// TODO: to be decided if app should get the possibility to regroup this
	response := abcitypes.ResponsePrepareProposal{Txs: proposal.Txs}
	mux.log.Debug("PrepareProposal called ", "#Txs", len(response.Txs), "proposal", proposal)
	return &response, nil
}

// ProcessProposal allows applications to check if proposed block is valid
func (mux *CometMux) ProcessProposal(ctx context.Context, proposal *abcitypes.RequestProcessProposal) (*abcitypes.ResponseProcessProposal, error) {
	mux.log.Debug("ProcessProposal called ", "#Txs", len(proposal.Txs), "proposal", proposal)

	handlerTxs := map[ChainAppIdentifier]([][]byte){}

	for idx := range proposal.Txs {
		hdlr, err := mux.getHandler(proposal.Txs[idx])
		if err != nil {
			mux.log.Error("call to ProcessProposal failed", "error", err)
			return nil, fmt.Errorf("no handler found for call")
		}
		// Add stripped transaction to handlers Tx set
		handlerTxs[hdlr.ID] = append(handlerTxs[hdlr.ID], proposal.Txs[idx][MbHeaderLen:])
	}

	type ProposalResponse struct {
		Response  *abcitypes.ResponseProcessProposal
		HandlerID ChainAppIdentifier
		Error     error
	}

	chanResp := make(chan ProposalResponse, len(mux.clients))
	wg := sync.WaitGroup{}

	for hdlrID, txs := range handlerTxs {
		wg.Add(1)
		hdlrID := hdlrID
		txs := txs
		newReq := *proposal
		newReq.Txs = txs
		chainID := mux.clients[hdlrID].ChainID
		mux.log.Debug("Forwarding ProcessProposal", "#TXs", len(newReq.Txs), "hdlr-id", hdlrID, "chain-id", chainID)
		go func() {
			defer wg.Done()
			appResp, err := mux.clients[hdlrID].client.ProcessProposal(ctx, &newReq)
			chanResp <- ProposalResponse{
				Response:  appResp,
				HandlerID: hdlrID,
				Error:     err}

		}()
	}

	// wait until all routines are done
	go func() {
		wg.Wait()
		close(chanResp)
	}()

	// loop until all response are received
	response := abcitypes.ResponseProcessProposal{Status: abcitypes.ResponseProcessProposal_ACCEPT}

	for resp := range chanResp {
		chainID := mux.clients[resp.HandlerID].ChainID
		mux.log.Debug("Response received on ProcessProposal", "chain-id", chainID, "response", resp.Response)

		if resp.Error != nil {
			mux.log.Error("call to ProcessProposal failed", "error",
				resp.Error, "chain-id", chainID)
			return nil, resp.Error
		}
		if response.Status == abcitypes.ResponseProcessProposal_ProposalStatus(abci.ResponseProcessProposal_UNKNOWN) {
			response.Status = resp.Response.Status
		}
		if resp.Response.Status != abcitypes.ResponseProcessProposal_ACCEPT {
			response.Status = resp.Response.Status
		}

	}
	// TODO: to be decided if app should get the ability to check that and outcome of 'Atomic IBC'
	mux.log.Debug("Overall Response on ProcessProposal", "response", response)
	return &response, nil
}

// Finalize Block: combination of BeginBlock, DeliverTx and EndBlock
//
// Note: FinalizeBlock only prepares the update to be made and does not change the state of the application.
// The state change is actually committed in a later stage i.e. in commit phase.
func (mux *CometMux) FinalizeBlock(ctx context.Context, req *abcitypes.RequestFinalizeBlock) (*abcitypes.ResponseFinalizeBlock, error) {
	mux.log.Debug("FinalizeBlock called", "#Txs", len(req.Txs), "req", req)

	responseSlots := map[ChainAppIdentifier][]int{}
	handlerTxs := map[ChainAppIdentifier]([][]byte){}

	for _, hdlr := range mux.clients {
		//hdlr.HandleFinalizeBlock(req, &txResults)
		handlerTxs[hdlr.ID] = [][]byte{}
	}

	for idx := range req.Txs {
		hdlr, err := mux.getHandler(req.Txs[idx])
		if err != nil {
			mux.log.Error("call to FinalizeBlock failed", "error", err)
			return nil, fmt.Errorf("no handler found for call")
		}
		// Add stripped transaction to handlers Tx set
		handlerTxs[hdlr.ID] = append(handlerTxs[hdlr.ID], req.Txs[idx][MbHeaderLen:])
		responseSlots[hdlr.ID] = append(responseSlots[hdlr.ID], idx)
	}

	// Send transactions to dedicated application
	appHashes := map[ChainAppIdentifier][]byte{}
	validatorUpdates := map[ChainAppIdentifier][]abcitypes.ValidatorUpdate{}
	events := map[ChainAppIdentifier][]abcitypes.Event{}
	response := abcitypes.ResponseFinalizeBlock{
		TxResults: make([]*abcitypes.ExecTxResult, len(req.Txs)),
	}

	type FinalizeResponse struct {
		Response  *abcitypes.ResponseFinalizeBlock
		HandlerID ChainAppIdentifier
		Slots     []int
		Error     error
	}

	chanResp := make(chan FinalizeResponse, len(mux.clients))
	wg := sync.WaitGroup{}
	wg.Add(len(mux.clients))

	for hdlrID, txs := range handlerTxs {
		hdlrID := hdlrID
		txs := txs
		newReq := *req
		newReq.Txs = txs
		chainID := mux.clients[hdlrID].ChainID
		mux.log.Debug("Forwarding FinalizeBlock", "#TXs", len(newReq.Txs), "hdlr-id", hdlrID, "chain-id", chainID)
		go func() {
			defer wg.Done()
			appResp, err := mux.clients[hdlrID].client.FinalizeBlock(ctx, &newReq)
			chanResp <- FinalizeResponse{
				Response:  appResp,
				HandlerID: hdlrID,
				Slots:     responseSlots[hdlrID],
				Error:     err}

		}()
	}

	// wait until all routines are done
	go func() {
		wg.Wait()
		close(chanResp)
	}()

	// loop until all response are received
	for resp := range chanResp {
		chainID := mux.clients[resp.HandlerID].ChainID

		if resp.Error != nil {
			mux.log.Error("call to FinalizeBlock failed", "error",
				resp.Error, "chain-id", chainID)
			return nil, resp.Error
		}
		mux.log.Debug("Response received on FinalizeBlock", "chain-id", chainID, "response", resp.Response)

		slots := resp.Slots
		chainResponse := resp.Response
		for idx, resp := range chainResponse.TxResults {
			if idx < len(slots) {
				mux.log.Debug("Adding result TxResult entry", "chain-id", chainID, "slots", slots, "idx", idx, "txResults", chainResponse.TxResults)
				response.TxResults[slots[idx]] = resp
			} else {
				mux.log.Debug("result index mismatch no matching slot for response")
			}
		}

		// TBD: handling of consensus parameters from different chain apps
		//      It is assumed that this must be equal for all chain apps
		if response.ConsensusParamUpdates == nil {
			response.ConsensusParamUpdates = resp.Response.ConsensusParamUpdates
		}

		// store in a map as we need ordered result on the following
		appHashes[resp.HandlerID] = chainResponse.AppHash
		validatorUpdates[resp.HandlerID] = chainResponse.ValidatorUpdates
		events[resp.HandlerID] = chainResponse.Events
	}

	// sort hash results by ChainAppID and append them
	keys := []ChainAppIdentifier{}
	for k := range appHashes {
		keys = append(keys, k)
	}
	SortChainAppIDs(keys)
	for _, k := range keys {
		response.AppHash = append(response.AppHash, appHashes[k]...)
		response.ValidatorUpdates = append(response.ValidatorUpdates, validatorUpdates[k]...)
		response.Events = append(response.Events, events[k]...)
	}

	mux.log.Debug("Overall FinalizeBlock response is", "response", response)
	return &response, nil
}

// Commit sends commit to all apps
func (mux CometMux) Commit(ctx context.Context, commit *abcitypes.RequestCommit) (*abcitypes.ResponseCommit, error) {
	mux.log.Debug("Commit called", "commit", commit)
	var response *abcitypes.ResponseCommit
	for _, hdlr := range mux.clients {
		cl := hdlr.client
		resp, err := cl.Commit(ctx, commit)
		if err != nil {
			mux.log.Error("error forwarding Commit: ", err)
			return nil, err
		}
		if response != nil && resp.RetainHeight != response.RetainHeight {
			mux.log.Info("Unexpected retain height diverge", "chain-id", hdlr.ChainID,
				"this height", resp.RetainHeight, "prev-height", resp.RetainHeight)
		}
		response = resp
	}

	return response, nil
}

func (mux *CometMux) ListSnapshots(_ context.Context, snapshots *abcitypes.RequestListSnapshots) (*abcitypes.ResponseListSnapshots, error) {
	mux.log.Debug("ListSnapshots called", "snapshot", snapshots)
	return &abcitypes.ResponseListSnapshots{}, nil
}

func (mux *CometMux) OfferSnapshot(_ context.Context, snapshot *abcitypes.RequestOfferSnapshot) (*abcitypes.ResponseOfferSnapshot, error) {
	mux.log.Debug("OfferSnapshots called", "snapshot", snapshot)
	return &abcitypes.ResponseOfferSnapshot{}, nil
}

func (mux *CometMux) LoadSnapshotChunk(_ context.Context, chunk *abcitypes.RequestLoadSnapshotChunk) (*abcitypes.ResponseLoadSnapshotChunk, error) {
	mux.log.Debug("LoadSnapshots called", "chunk", chunk)
	return &abcitypes.ResponseLoadSnapshotChunk{}, nil
}

func (mux *CometMux) ApplySnapshotChunk(_ context.Context, chunk *abcitypes.RequestApplySnapshotChunk) (*abcitypes.ResponseApplySnapshotChunk, error) {
	mux.log.Debug("ApplySnapshots called", "chunk", chunk)
	return &abcitypes.ResponseApplySnapshotChunk{Result: abcitypes.ResponseApplySnapshotChunk_ACCEPT}, nil
}

func (mux CometMux) ExtendVote(_ context.Context, extend *abcitypes.RequestExtendVote) (*abcitypes.ResponseExtendVote, error) {
	mux.log.Debug("ExtendVote called", "request", extend)
	return &abcitypes.ResponseExtendVote{}, nil
}

func (mux *CometMux) VerifyVoteExtension(_ context.Context, verify *abcitypes.RequestVerifyVoteExtension) (*abcitypes.ResponseVerifyVoteExtension, error) {
	mux.log.Debug("VerifyVoteExtension called", "request", verify)
	return &abcitypes.ResponseVerifyVoteExtension{}, nil
}
