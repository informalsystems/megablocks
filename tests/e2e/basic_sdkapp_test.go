package e2e

import (
	"fmt"
	"strings"
	"testing"
)

// This test checks that a megablocks transaction and queries are
// work for the sdk-applicatin running on the cosmux mutliplexer
//
// The test triggers a send transaction of bank module and verifies
// that the transaction was successful by querying the related balance.
func TestBasicSendTransactionAndQuery(t *testing.T) {
	// start applications
	kvApp := createKVStore()
	sdkApp := createSdkApp()
	apps, err := startApplications(kvApp, sdkApp)
	if err != nil {
		t.Errorf("error starting apps: %v", err)
		return
	}
	defer stopApplications(apps)

	//
	// start multiplexer
	cosmux := createCosMux(sdkApp.Home)
	err = cosmux.Init()
	if err != nil {
		t.Errorf("failed initializing multiplexer: %v", err)
		return
	}
	err = cosmux.Start()
	if err != nil {
		t.Errorf("failed starting multiplexer: %v", err)
		return
	}
	defer cosmux.Stop()

	// Issue bank transaction
	var aliceAddr, bobAddr string
	if aliceAddr, err = sdkApp.GetUserAddress(sdkApp.NodeKey); err != nil {
		t.Errorf("Failed getting carol's address: %v", err)
		return
	}

	bob := "bob"
	if bobAddr, err = sdkApp.GetUserAddress(bob); err != nil {
		t.Errorf("Failed getting bob's address: %v", err)
		return
	}

	var balance int
	if balance, err = sdkApp.GetBalance(aliceAddr, "stake"); err != nil {
		t.Errorf("Failed getting bob's balance: %v", err)
		return
	}
	if balance <= 1000 {
		t.Errorf("error: not enough balance to send stakes: current balance=%dstake, needed=%dstake", balance, 1000)
		return
	}
	fmt.Println("alice's balance is: ", balance)

	if err = sdkApp.SendBankTransaction(aliceAddr, bobAddr, "1000stake"); err != nil {
		t.Errorf("Failed sending transaction: %v", err)
		return
	}

	currentHeight, err := sdkApp.GetLatestBlockHeight()
	if err != nil {
		t.Errorf("error getting latest block height: %v", err)
		return
	}

	sdkApp.waitForBlockHeight(currentHeight+1, BlockTime)
	if balance, err = sdkApp.GetBalance(bobAddr, "stake"); err != nil {
		t.Errorf("Failed getting bob's balance: %v", err)
		return
	}

	if balance != 1000 {
		t.Errorf("Unexpected balance for %s: got=%dstake, expected=%s",
			bob, balance, "1000stake")
	}

}

func TestBasicSdkQueryWithWrongChain(t *testing.T) {

	// start applications
	kvApp := createKVStore()
	sdkApp := createSdkApp()
	apps, err := startApplications(kvApp, sdkApp)
	if err != nil {
		t.Errorf("error starting apps: %v", err)
		return
	}
	defer stopApplications(apps)

	//
	// start multiplexer
	cosmux := createCosMux(sdkApp.Home)
	err = cosmux.Init()
	if err != nil {
		t.Errorf("failed initializing multiplexer: %v", err)
		return
	}
	err = cosmux.Start()
	if err != nil {
		t.Errorf("failed starting multiplexer: %v", err)
		return
	}
	defer cosmux.Stop()

	// Issue bank transaction
	var aliceAddr string
	if aliceAddr, err = sdkApp.GetUserAddress(sdkApp.NodeKey); err != nil {
		t.Errorf("Failed getting carol's address: %v", err)
		return
	}

	_, err = sdkApp.GetBalanceWithChainID(aliceAddr, "stake", "invalid-chain-id")
	if err == nil {
		t.Errorf("Query for balance with invalid chain did not fail as expected: %v", err)
		return
	}
	if !strings.Contains(err.Error(), "no application handler found") {
		t.Errorf("Unexpected error returned on query. 'no application handler found' not in error: '%s'", err.Error())
		return
	}
}
