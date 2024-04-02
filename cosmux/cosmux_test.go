package main

import (
	"context"
	"crypto/sha1"
	"fmt"
	"reflect"
	"testing"

	abcitypes "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/proto/tendermint/crypto"
	"github.com/cometbft/cometbft/proto/tendermint/types"
	comettypes "github.com/cometbft/cometbft/types"
	gomock "github.com/golang/mock/gomock"
	"github.com/informalsystems/megablocks/testutil/mocks"
)

func generateTxApp(chainID string, data []byte) []byte {
	tx := comettypes.Tx{0x23, 0x6d, 0x75, 0x78} // Megablocks MAGIC
	checksum := sha1.Sum([]byte(chainID))
	megablocks_id := checksum[:4]
	return append(tx, megablocks_id...)
}

func getChainAppIdentifier(chainID string) ChainAppIdentifier {
	sha1Sum := sha1.Sum([]byte(chainID))
	return ChainAppIdentifier(sha1Sum[:5])
}

func createHeader(chainId string) []byte {
	identifier := getChainAppIdentifier(chainId)
	return append(MAGIC[:], identifier[:]...)
}

func TestCheckHeader(t *testing.T) {

	type HdrCheck struct {
		Name            string
		Header          []byte
		ExpectedFailure bool
	}

	badHeader := []byte{0x23, 0xdd, 0x75, 0x78}

	checks := []HdrCheck{
		{
			Name:            "GoodTest",
			Header:          append(MAGIC[:], 0x01, 0x02, 0x03, 0x04),
			ExpectedFailure: false,
		},
		{
			Name:            "HeaderTooShort",
			Header:          append(MAGIC[:], 0x01, 0x02, 0x03),
			ExpectedFailure: true,
		},
		{
			Name:            "WrongMagic",
			Header:          append(badHeader, 0x01, 0x02, 0x03, 0x04),
			ExpectedFailure: true,
		},
		{
			Name:            "EmptyHeader",
			Header:          []byte{},
			ExpectedFailure: true,
		},
	}

	for _, check := range checks {
		err := CheckHeader(check.Header)
		if err == nil && check.ExpectedFailure {
			t.Errorf("CheckHeader on test %s passed where it is expected to fail", check.Name)
			return
		}
		if err != nil && !check.ExpectedFailure {
			t.Errorf("CheckHeader on test %s failed with (%v) where it is expected to pass",
				check.Name, err)
			return
		}
	}
}

func TestGetHandler(t *testing.T) {
	type HdrCheck struct {
		Name            string
		Header          []byte
		ChainId         string
		NumHandlers     int
		ExpectedFailure bool
	}

	cosmux := NewMultiplexer(
		&CosmuxConfig{LogLevel: "debug"},
	)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	checks := []HdrCheck{
		{
			Name:            "Positive Test",
			ChainId:         "myChain",
			Header:          createHeader("myChain"),
			ExpectedFailure: false,
			NumHandlers:     1,
		},
		{
			Name:            "No matching handler",
			ChainId:         "myChain",
			Header:          createHeader("otherChain"),
			ExpectedFailure: true,
			NumHandlers:     1,
		},
		{
			Name:            "Invalid header",
			ChainId:         "myChain",
			Header:          append(MAGIC[:], 0x12),
			ExpectedFailure: false,
			NumHandlers:     1,
		},
		{
			Name:            "Empty handlers",
			ChainId:         "myChain",
			Header:          append(MAGIC[:], 0x12),
			ExpectedFailure: false,
			NumHandlers:     0,
		},
	}

	for _, check := range checks {
		if check.NumHandlers > 0 {
			// Create Abci handler with mocked ABCIclient
			mockclient := mocks.NewMockClient(mockCtrl)
			//response := abcitypes.ResponseFinalizeBlock{}
			//mockclient.EXPECT().FinalizeBlock(gomock.Any, gomock.Any).Return(&response, nil).AnyTimes()

			cosmux.clients = map[ChainAppIdentifier]*AbciHandler{
				getChainAppIdentifier(check.ChainId): &AbciHandler{
					ChainID: check.ChainId,
					ID:      getChainAppIdentifier(check.ChainId),
					client:  mockclient,
				},
			}
		}
		hdlr, err := cosmux.getHandler(check.Header)

		// Fail on unexpected pass
		if err == nil && check.ExpectedFailure {
			t.Errorf("CheckHeader on test '%s' passed where it is expected to fail. Got handler=%+v", check.Name, hdlr)
			return
		}

		// Fail on unexpected errors
		if err != nil && !check.ExpectedFailure {
			t.Errorf("CheckHeader on test '%s' failed with (%v) where it is expected to pass",
				check.Name, err)
			return
		}

		// no further checks on expected failures
		if err != nil {
			return
		}

		if hdlr.ChainID != check.ChainId {
			t.Errorf("Check %s returned wrong handler: expected=%s, got=%s",
				check.Name, check.ChainId, hdlr.ChainID)
		}
	}
}

func TestInitChain(t *testing.T) {
	cosmux := NewMultiplexer(
		&CosmuxConfig{LogLevel: "debug"},
	)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	type AppResponse struct {
		Response *abcitypes.ResponseInitChain
		Error    error
	}
	type InitChainCheck struct {
		Test             string
		ChainIDs         []string
		Request          abcitypes.RequestInitChain
		AppResponses     map[string]AppResponse
		ExpectedResponse *abcitypes.ResponseInitChain
		ExpectedFailure  bool
	}

	checks := []InitChainCheck{
		{
			Test: "InitChain for 2 chain apps",
			Request: abcitypes.RequestInitChain{
				ChainId:         "initialChain",
				ConsensusParams: &types.ConsensusParams{},
				Validators:      []abcitypes.ValidatorUpdate{},
				AppStateBytes:   []byte{},
			},
			AppResponses: map[string]AppResponse{
				"chain1": {
					Response: &abcitypes.ResponseInitChain{
						ConsensusParams: &types.ConsensusParams{
							Block: &types.BlockParams{
								MaxBytes: 1,
								MaxGas:   4},
						},
						Validators: []abcitypes.ValidatorUpdate{},
						AppHash:    []byte{0xde, 0xa, 0xd, 0xbe, 0xef},
					},
					Error: nil,
				},
				"chain2": {
					Response: &abcitypes.ResponseInitChain{
						ConsensusParams: &types.ConsensusParams{
							Block: &types.BlockParams{
								MaxBytes: 1,
								MaxGas:   4},
						},
						Validators: []abcitypes.ValidatorUpdate{
							{PubKey: crypto.PublicKey{Sum: &crypto.PublicKey_Ed25519{Ed25519: []byte{1, 2, 3}}}},
							{PubKey: crypto.PublicKey{Sum: &crypto.PublicKey_Ed25519{Ed25519: []byte{3, 4, 5}}}},
						},
						AppHash: []byte{},
					},
				},
			},
			ExpectedResponse: &abcitypes.ResponseInitChain{
				ConsensusParams: &types.ConsensusParams{
					Block: &types.BlockParams{
						MaxBytes: 1,
						MaxGas:   4},
				},
				Validators: []abcitypes.ValidatorUpdate{
					{PubKey: crypto.PublicKey{Sum: &crypto.PublicKey_Ed25519{Ed25519: []byte{1, 2, 3}}}},
					{PubKey: crypto.PublicKey{Sum: &crypto.PublicKey_Ed25519{Ed25519: []byte{3, 4, 5}}}},
				},
				AppHash: []byte{0xde, 0xa, 0xd, 0xbe, 0xef},
			},
			ExpectedFailure: false,
		},
		{
			Test: "InitChain for 2 chain apps with one returning error",
			Request: abcitypes.RequestInitChain{
				ChainId:         "initialChain",
				ConsensusParams: &types.ConsensusParams{},
				Validators:      []abcitypes.ValidatorUpdate{},
				AppStateBytes:   []byte{},
			},
			AppResponses: map[string]AppResponse{
				"chain1": {
					Response: &abcitypes.ResponseInitChain{
						ConsensusParams: &types.ConsensusParams{
							Block: &types.BlockParams{
								MaxBytes: 1,
								MaxGas:   4},
						},
						Validators: []abcitypes.ValidatorUpdate{
							{PubKey: crypto.PublicKey{Sum: &crypto.PublicKey_Ed25519{Ed25519: []byte{1, 2, 3}}}},
							{PubKey: crypto.PublicKey{Sum: &crypto.PublicKey_Ed25519{Ed25519: []byte{3, 4, 5}}}},
						},
						AppHash: []byte{},
					},
					Error: nil,
				},
				"chain2": {
					Response: nil,
					Error:    fmt.Errorf("error returned from chain 2")},
			},
			ExpectedResponse: &abcitypes.ResponseInitChain{
				ConsensusParams: &types.ConsensusParams{
					Block: &types.BlockParams{
						MaxBytes: 1,
						MaxGas:   4},
				},
				Validators: []abcitypes.ValidatorUpdate{
					{PubKey: crypto.PublicKey{Sum: &crypto.PublicKey_Ed25519{Ed25519: []byte{1, 2, 3}}}},
					{PubKey: crypto.PublicKey{Sum: &crypto.PublicKey_Ed25519{Ed25519: []byte{3, 4, 5}}}},
				},
				AppHash: []byte{},
			},
			ExpectedFailure: true,
		},
		{
			Test: "InitChain for 2 chain apps returning errors",
			Request: abcitypes.RequestInitChain{
				ChainId:         "initialChain",
				ConsensusParams: &types.ConsensusParams{},
				Validators:      []abcitypes.ValidatorUpdate{},
				AppStateBytes:   []byte{},
			},
			AppResponses: map[string]AppResponse{
				"chain1": {
					Response: nil,
					Error:    fmt.Errorf("error returned from chain 1")},
				"chain2": {
					Response: nil,
					Error:    fmt.Errorf("error returned from chain 2")},
			},
			ExpectedResponse: nil,
			ExpectedFailure:  true,
		},
	}

	ctx := context.Background()
	for _, check := range checks {
		// Setup handler for chain applications
		cosmux.clients = map[ChainAppIdentifier]*AbciHandler{}
		for chainId, response := range check.AppResponses {
			mockclient := mocks.NewMockClient(mockCtrl)

			mockclient.EXPECT().InitChain(gomock.Any(), gomock.Any()).Return(response.Response, response.Error).AnyTimes()
			cosmux.clients[getChainAppIdentifier(chainId)] = &AbciHandler{
				ChainID: chainId,
				ID:      getChainAppIdentifier(chainId),
				client:  mockclient,
			}
		}

		// Call InitChain on the multiplexer
		resp, err := cosmux.InitChain(ctx, &check.Request)
		if check.ExpectedFailure && err == nil {
			t.Errorf("InitChain did not return an error")
			return
		}

		// Check results
		if !check.ExpectedFailure && err != nil {
			t.Error("InitChain failed with error:", err)
			return
		}

		if check.ExpectedFailure {
			return
		}

		if !reflect.DeepEqual(resp, check.ExpectedResponse) {
			t.Errorf("Test '%s' failed:\nExpected:\n%v\n\nGot:\n%v\n", check.Test,
				check.ExpectedResponse, resp)
		}
	}
}

func TestFinalizeBock(t *testing.T) {

	cosmux := NewMultiplexer(
		&CosmuxConfig{LogLevel: "debug"},
	)

	type FinalizeCheck struct {
		Name             string
		Header           []byte
		ChainIDs         []string
		NumHandlers      int
		ClientResponse   map[string]abcitypes.ResponseFinalizeBlock
		Request          abcitypes.RequestFinalizeBlock
		ExpectedResponse abcitypes.ResponseFinalizeBlock
		ExpectedFailure  bool
	}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	checks := []FinalizeCheck{
		{
			Name:            "Positive Test",
			ChainIDs:        []string{"myChain", "anotherChain"},
			Header:          createHeader("myChain"),
			ExpectedFailure: false,
			ClientResponse: map[string]abcitypes.ResponseFinalizeBlock{
				"myChain": {
					Events: []abcitypes.Event{
						{
							Type:       "myChain-Attr1",
							Attributes: []abcitypes.EventAttribute{{Key: "k1", Value: "val1"}},
						},
						{
							Type:       "myChain-Attr2",
							Attributes: []abcitypes.EventAttribute{{Key: "k2", Value: "val2"}},
						},
					},
					TxResults: []*abcitypes.ExecTxResult{
						{Info: "myChain", GasWanted: 11, GasUsed: 11},
						{Info: "myChain", GasWanted: 12, GasUsed: 12},
						{Info: "myChain", GasWanted: 13, GasUsed: 13},
						{Info: "myChain", GasWanted: 14, GasUsed: 14},
						{Info: "myChain", GasWanted: 15, GasUsed: 15},
					},
					ValidatorUpdates: []abcitypes.ValidatorUpdate{
						{PubKey: crypto.PublicKey{Sum: &crypto.PublicKey_Ed25519{Ed25519: []byte{1, 2, 3}}}, Power: 50},
						{PubKey: crypto.PublicKey{Sum: &crypto.PublicKey_Ed25519{Ed25519: []byte{3, 4, 5}}}, Power: 60},
					},
					ConsensusParamUpdates: &types.ConsensusParams{
						Block: &types.BlockParams{MaxBytes: 1024, MaxGas: 4000},
					},
					AppHash: []byte{0xff, 0xf1, 0x02, 0x01},
				},
				"anotherChain": {
					Events: []abcitypes.Event{
						{
							Type:       "anotherChain-Attr1",
							Attributes: []abcitypes.EventAttribute{{Key: "x1", Value: "y1"}},
						},
					},
					TxResults: []*abcitypes.ExecTxResult{
						{Info: "anotherChain", GasWanted: 21, GasUsed: 21},
						{Info: "anotherChain", GasWanted: 22, GasUsed: 22},
						{Info: "anotherChain", GasWanted: 23, GasUsed: 23},
						{Info: "anotherChain", GasWanted: 24, GasUsed: 24},
						{Info: "anotherChain", GasWanted: 25, GasUsed: 25},
					},
					ValidatorUpdates: []abcitypes.ValidatorUpdate{
						{PubKey: crypto.PublicKey{Sum: &crypto.PublicKey_Ed25519{Ed25519: []byte{6, 7, 8}}}, Power: 70},
					},
					ConsensusParamUpdates: &types.ConsensusParams{
						Block: &types.BlockParams{MaxBytes: 1024, MaxGas: 4000},
					},
					AppHash: []byte{0xa1, 0xb1, 0xc1, 0xd1},
				},
			},
			Request: abcitypes.RequestFinalizeBlock{
				Txs: [][]byte{
					append(createHeader("myChain"), 0x01, 0x00, 0xff),
					append(createHeader("anotherChain"), 0x02, 0x00, 0xff),
					append(createHeader("anotherChain"), 0x02, 0x01, 0xff),
					append(createHeader("myChain"), 0x01, 0x01, 0xff),
					append(createHeader("myChain"), 0x01, 0x03, 0xff),
					append(createHeader("myChain"), 0x01, 0x04, 0xff),
					append(createHeader("anotherChain"), 0x02, 0x03, 0xff),
					append(createHeader("myChain"), 0x01, 0x05, 0xff),
					append(createHeader("anotherChain"), 0x02, 0x04, 0xff),
					append(createHeader("anotherChain"), 0x02, 0x05, 0xff),
				},
			},
			ExpectedResponse: abcitypes.ResponseFinalizeBlock{
				Events: []abcitypes.Event{
					{
						Type:       "anotherChain-Attr1",
						Attributes: []abcitypes.EventAttribute{{Key: "x1", Value: "y1"}},
					},
					{
						Type:       "myChain-Attr1",
						Attributes: []abcitypes.EventAttribute{{Key: "k1", Value: "val1"}},
					},
					{
						Type:       "myChain-Attr2",
						Attributes: []abcitypes.EventAttribute{{Key: "k2", Value: "val2"}},
					},
				},
				TxResults: []*abcitypes.ExecTxResult{
					{Info: "myChain", GasWanted: 11, GasUsed: 11},
					{Info: "anotherChain", GasWanted: 21, GasUsed: 21},
					{Info: "anotherChain", GasWanted: 22, GasUsed: 22},
					{Info: "myChain", GasWanted: 12, GasUsed: 12},
					{Info: "myChain", GasWanted: 13, GasUsed: 13},
					{Info: "myChain", GasWanted: 14, GasUsed: 14},
					{Info: "anotherChain", GasWanted: 23, GasUsed: 23},
					{Info: "myChain", GasWanted: 15, GasUsed: 15},
					{Info: "anotherChain", GasWanted: 24, GasUsed: 24},
					{Info: "anotherChain", GasWanted: 25, GasUsed: 25},
				},
				ValidatorUpdates: []abcitypes.ValidatorUpdate{
					{PubKey: crypto.PublicKey{Sum: &crypto.PublicKey_Ed25519{Ed25519: []byte{6, 7, 8}}}, Power: 70},
					{PubKey: crypto.PublicKey{Sum: &crypto.PublicKey_Ed25519{Ed25519: []byte{1, 2, 3}}}, Power: 50},
					{PubKey: crypto.PublicKey{Sum: &crypto.PublicKey_Ed25519{Ed25519: []byte{3, 4, 5}}}, Power: 60},
				},
				ConsensusParamUpdates: &types.ConsensusParams{
					Block: &types.BlockParams{MaxBytes: 1024, MaxGas: 4000},
				},
				AppHash: []byte{0xa1, 0xb1, 0xc1, 0xd1, 0xff, 0xf1, 0x02, 0x01},
			},
		},

		{
			Name:            "Positive Test with one app having empty TxResults",
			ChainIDs:        []string{"myChain", "anotherChain"},
			Header:          createHeader("myChain"),
			ExpectedFailure: false,
			ClientResponse: map[string]abcitypes.ResponseFinalizeBlock{
				"myChain": {
					Events: []abcitypes.Event{
						{
							Type:       "myChain-Attr1",
							Attributes: []abcitypes.EventAttribute{{Key: "k1", Value: "val1"}},
						},
						{
							Type:       "myChain-Attr2",
							Attributes: []abcitypes.EventAttribute{{Key: "k2", Value: "val2"}},
						},
					},
					TxResults: []*abcitypes.ExecTxResult{
						{Info: "myChain", GasWanted: 11, GasUsed: 11},
						{Info: "myChain", GasWanted: 12, GasUsed: 12},
						{Info: "myChain", GasWanted: 13, GasUsed: 13},
						{Info: "myChain", GasWanted: 14, GasUsed: 14},
						{Info: "myChain", GasWanted: 15, GasUsed: 15},
					},
					ValidatorUpdates: []abcitypes.ValidatorUpdate{
						{PubKey: crypto.PublicKey{Sum: &crypto.PublicKey_Ed25519{Ed25519: []byte{1, 2, 3}}}, Power: 50},
						{PubKey: crypto.PublicKey{Sum: &crypto.PublicKey_Ed25519{Ed25519: []byte{3, 4, 5}}}, Power: 60},
					},
					ConsensusParamUpdates: &types.ConsensusParams{
						Block: &types.BlockParams{MaxBytes: 1024, MaxGas: 4000},
					},
					AppHash: []byte{0xff, 0xf1, 0x02, 0x01},
				},
				"anotherChain": {
					Events: []abcitypes.Event{
						{
							Type:       "anotherChain-Attr1",
							Attributes: []abcitypes.EventAttribute{{Key: "x1", Value: "y1"}},
						},
					},
					TxResults: []*abcitypes.ExecTxResult{
						{Info: "anotherChain", GasWanted: 21, GasUsed: 21},
						{Info: "anotherChain", GasWanted: 22, GasUsed: 22},
						{Info: "anotherChain", GasWanted: 23, GasUsed: 23},
						{Info: "anotherChain", GasWanted: 24, GasUsed: 24},
						{Info: "anotherChain", GasWanted: 25, GasUsed: 25},
					},
					ValidatorUpdates: []abcitypes.ValidatorUpdate{
						{PubKey: crypto.PublicKey{Sum: &crypto.PublicKey_Ed25519{Ed25519: []byte{6, 7, 8}}}, Power: 70},
					},
					ConsensusParamUpdates: &types.ConsensusParams{
						Block: &types.BlockParams{MaxBytes: 1024, MaxGas: 4000},
					},
					AppHash: []byte{0xa1, 0xb1, 0xc1, 0xd1},
				},
			},
			Request: abcitypes.RequestFinalizeBlock{
				Txs: [][]byte{
					append(createHeader("myChain"), 0x01, 0x00, 0xff),
					append(createHeader("anotherChain"), 0x02, 0x00, 0xff),
					append(createHeader("anotherChain"), 0x02, 0x01, 0xff),
					append(createHeader("myChain"), 0x01, 0x01, 0xff),
					append(createHeader("myChain"), 0x01, 0x03, 0xff),
					append(createHeader("myChain"), 0x01, 0x04, 0xff),
					append(createHeader("anotherChain"), 0x02, 0x03, 0xff),
					append(createHeader("myChain"), 0x01, 0x05, 0xff),
					append(createHeader("anotherChain"), 0x02, 0x04, 0xff),
					append(createHeader("anotherChain"), 0x02, 0x05, 0xff),
				},
			},
			ExpectedResponse: abcitypes.ResponseFinalizeBlock{
				Events: []abcitypes.Event{
					{
						Type:       "anotherChain-Attr1",
						Attributes: []abcitypes.EventAttribute{{Key: "x1", Value: "y1"}},
					},
					{
						Type:       "myChain-Attr1",
						Attributes: []abcitypes.EventAttribute{{Key: "k1", Value: "val1"}},
					},
					{
						Type:       "myChain-Attr2",
						Attributes: []abcitypes.EventAttribute{{Key: "k2", Value: "val2"}},
					},
				},
				TxResults: []*abcitypes.ExecTxResult{
					{Info: "myChain", GasWanted: 11, GasUsed: 11},
					{Info: "anotherChain", GasWanted: 21, GasUsed: 21},
					{Info: "anotherChain", GasWanted: 22, GasUsed: 22},
					{Info: "myChain", GasWanted: 12, GasUsed: 12},
					{Info: "myChain", GasWanted: 13, GasUsed: 13},
					{Info: "myChain", GasWanted: 14, GasUsed: 14},
					{Info: "anotherChain", GasWanted: 23, GasUsed: 23},
					{Info: "myChain", GasWanted: 15, GasUsed: 15},
					{Info: "anotherChain", GasWanted: 24, GasUsed: 24},
					{Info: "anotherChain", GasWanted: 25, GasUsed: 25},
				},
				ValidatorUpdates: []abcitypes.ValidatorUpdate{
					{PubKey: crypto.PublicKey{Sum: &crypto.PublicKey_Ed25519{Ed25519: []byte{6, 7, 8}}}, Power: 70},
					{PubKey: crypto.PublicKey{Sum: &crypto.PublicKey_Ed25519{Ed25519: []byte{1, 2, 3}}}, Power: 50},
					{PubKey: crypto.PublicKey{Sum: &crypto.PublicKey_Ed25519{Ed25519: []byte{3, 4, 5}}}, Power: 60},
				},
				ConsensusParamUpdates: &types.ConsensusParams{
					Block: &types.BlockParams{MaxBytes: 1024, MaxGas: 4000},
				},
				AppHash: []byte{0xa1, 0xb1, 0xc1, 0xd1, 0xff, 0xf1, 0x02, 0x01},
			},
		},
	}

	ctx := context.Background()
	for _, check := range checks {
		cosmux.clients = map[ChainAppIdentifier]*AbciHandler{}
		for _, chainId := range check.ChainIDs {
			mockclient := mocks.NewMockClient(mockCtrl)
			chainResponse := check.ClientResponse[chainId]
			mockclient.EXPECT().FinalizeBlock(gomock.Any(), gomock.Any()).Return(&chainResponse, nil).AnyTimes()
			cosmux.clients[getChainAppIdentifier(chainId)] = &AbciHandler{
				ChainID: chainId,
				ID:      getChainAppIdentifier(chainId),
				client:  mockclient,
			}
		}
		response, err := cosmux.FinalizeBlock(ctx, &check.Request)

		// Fail on unexpected pass
		if err == nil && check.ExpectedFailure {
			t.Errorf("TestFinalizeBloc on check '%s' passed where it is expected to fail. Got response=%+v",
				check.Name, response)
			return
		}

		// Fail on unexpected errors
		if err != nil && !check.ExpectedFailure {
			t.Errorf("TestFinalizeBloc on check '%s' failed with (%v) where it is expected to pass",
				check.Name, err)
			return
		}

		// no further checks on expected failures
		if err != nil {
			return
		}

		// Check TxResults
		if !reflect.DeepEqual(check.ExpectedResponse.TxResults, response.TxResults) {
			t.Errorf("TxResults mismatch: Got=%+v, Want=%+v",
				response.TxResults, check.ExpectedResponse.TxResults)
			return
		}

		// Check AppHash
		if !reflect.DeepEqual(check.ExpectedResponse.AppHash, response.AppHash) {
			t.Errorf("AppHash mismatch: Got=%+v, Want=%+v",
				response.AppHash, check.ExpectedResponse.AppHash)
			return
		}

		// Check ConsensusParamUpdates
		if !reflect.DeepEqual(check.ExpectedResponse.ConsensusParamUpdates, response.ConsensusParamUpdates) {
			t.Errorf("ConsensusParamUpdates mismatch '%s': Got=%+v, Want=%+v", check.Name,
				response.ConsensusParamUpdates, check.ExpectedResponse.ConsensusParamUpdates)
			return
		}

		// Check Events
		if !reflect.DeepEqual(check.ExpectedResponse.Events, response.Events) {
			t.Errorf("Events mismatch in '%s': Got=%+v, Want=%+v", check.Name,
				response.Events, check.ExpectedResponse.Events)
			return
		}

		// Check Response
		if !reflect.DeepEqual(check.ExpectedResponse.ValidatorUpdates, response.ValidatorUpdates) {
			t.Errorf("ValidatorsUpdates mismatch '%s': Got=%+v, Want=%+v", check.Name,
				response.ValidatorUpdates, check.ExpectedResponse.ValidatorUpdates)
			return
		}

	}
}
