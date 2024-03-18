package e2e

import (
	"fmt"
	"testing"
	"time"
)

// This tests expects as a pre-requisite to have cosmux and clients up and running
// It will send transactions to client applications and check that the expected transactions
// were processed as expected
func TestManualClient(t *testing.T) {
	t.Skip()
	kvs := map[string]string{}

	client, err := Client(HOST, fmt.Sprint(CometGrpcPort))
	if err != nil {
		t.Errorf("error creating client: %v", err)
		return

	}

	for i := 1; i < 3; i++ {
		kvs[fmt.Sprintf("key_%d", i)] = fmt.Sprintf("value_%d", i)
	}
	// Send KVStore transactions
	kvApp := createKVStore()
	for key, value := range kvs {
		fmt.Printf("Sending transaction for chain '%s': %s=%s\n", kvApp.ChainID, key, value)
		sendKvMBtransaction(client, kvApp.ChainID, key, value)
	}

	// Check transaction was successful
	start := time.Now()
	timeout := time.Second * 11
	for key, value := range kvs {
		fmt.Println("Checking for key", key, " matching value", value)
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
				t.Log("resulting value is: ", key, "=", result)
				t.Errorf("timed out checking KV result")
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
	}

}
