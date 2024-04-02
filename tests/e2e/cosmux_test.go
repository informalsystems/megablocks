package e2e

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func getKeyVals(count int) map[string]string {
	kv := map[string]string{}
	for i := 0; i < count; i++ {
		kv[fmt.Sprintf("key_%d", i)] = fmt.Sprintf("val_%d", i)
	}
	return kv
}

// Test simultaneous request for the registered applications
func TestParallelRequests(t *testing.T) {

	// start applications
	kvApp := createKVStore()
	sdkApp := createSdkApp()
	apps, err := startApplications(kvApp, sdkApp)
	if err != nil {
		t.Errorf("error starting apps: %v", err)
		return
	}
	defer stopApplications(apps)

	// start multiplexer
	cosmux := createCosMux(sdkApp.Home, apps)
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

	// wait for blockheight > 1 to enforce all TXs in the same blocka
	if err := sdkApp.waitForBlockHeight(1, 50*BlockTime); err != nil {
		t.Errorf("error waiting for blockheight > 1: %v", err)
		return
	}

	var aliceAddr, bobAddr string //, carolAddr string
	if aliceAddr, err = sdkApp.GetUserAddress(sdkApp.NodeKey); err != nil {
		t.Errorf("Failed getting alice' address: %v", err)
		return
	}

	bob := "bob"
	if bobAddr, err = sdkApp.GetUserAddress(bob); err != nil {
		t.Errorf("Failed getting bobs address: %v", err)
		return
	}

	var balance int
	if balance, err = sdkApp.GetBalance(bobAddr, "stake"); err != nil {
		t.Errorf("Failed getting bob's balance: %v", err)
		return
	} else {
		t.Log("bob's initial balance is", balance)
	}

	/// Send transactions to SDK-APP and KV Store
	// Issue bank transaction
	amount := 10
	if err = sdkApp.SendBankTransaction(aliceAddr, bobAddr, fmt.Sprintf("%dstake", amount)); err != nil {
		t.Errorf("Failed sending transaction: %v", err)
		return
	} else {
		t.Log("SDK bank transaction sent")
	}

	// Send key/value transactions for KV-Store app
	waitGroup := sync.WaitGroup{}
	keyVals := getKeyVals(2)
	for k, v := range keyVals {
		waitGroup.Add(1)
		go func(key, value string) {
			defer waitGroup.Done()
			client, err := Client(HOST, fmt.Sprint(CometGrpcPort))
			if err != nil {
				t.Errorf("error creating client: %v", err)
				return

			}
			// Set an entry in the KVStore
			err = sendKvMBtransaction(client, kvApp.ChainID, key, value)
			if err != nil {
				t.Errorf("Send transaction failed: %v", err)
				return
			}
		}(k, v)
	}
	waitGroup.Wait()

	currentHeight, err := sdkApp.GetLatestBlockHeight()
	if err != nil {
		t.Errorf("error getting latest block height: %v", err)
		return
	} else {
		t.Log("current height is", currentHeight)
	}

	// wait for next block and check balance for bob
	sdkApp.waitForBlockHeight(currentHeight+5, BlockTime)
	if balance, err = sdkApp.GetBalance(bobAddr, "stake"); err != nil {
		t.Errorf("Failed getting bob's balance: %v", err)
		return
	}

	if balance != amount {
		t.Errorf("Unexpected balance for %s: got=%dstake, expected=%s",
			bob, balance, "1000stake")
	} else {
		t.Log("bob's balance is ", balance)
	}

	// Check key/value entries in KV-Store application
	client, err := Client(HOST, fmt.Sprint(CometGrpcPort))
	if err != nil {
		t.Errorf("error creating client: %v", err)
		return

	}

	for key, value := range keyVals {
		start := time.Now()
		timeout := time.Second * 11
		for {
			result, err := queryMbKVStore(client, kvApp.ChainID, key)
			if err != nil {
				t.Errorf("query failed with %s", err.Error())
				return
			}
			if result == value {
				break
			} else if result != "" {
				t.Errorf("Invalid value for %s: expected=%s, got=%s.", key, value, result)
				return
			}

			if time.Since(start) > timeout {
				t.Log("KV query timed out. result value is: ", key, "=", result)
				t.Errorf("timed out checking KV result")
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}
