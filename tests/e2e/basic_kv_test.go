package e2e

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"testing"
	"time"

	"github.com/cometbft/cometbft/rpc/client/http"
	comettypes "github.com/cometbft/cometbft/types"
)

var (
	HOST string = "localhost"
)

// Set a key/value on the store
func sendKvMBtransaction(client *http.HTTP, id uint8, key, value string) error {
	tx := comettypes.Tx{0x23, 0x6d, 0x75, 0x78} //MAGIC
	tx = append(tx, id)
	tx = append(tx, []byte(fmt.Sprintf("%s=%s", key, value))...)
	return SendTx(client, tx)
}

// queryMbKVStore sends a query to get the value for a given key
// by using megablocks header on the query data
func queryMbKVStore(client *http.HTTP, id uint8, key string) (string, error) {
	ctx, loadCancel := context.WithCancel(context.Background())
	mbKey := []byte{0x23, 0x6d, 0x75, 0x78} //MAGIC
	mbKey = append(mbKey, id)
	mbKey = append(mbKey, []byte(key)...)

	response, err := client.ABCIQuery(ctx, "", mbKey)
	defer loadCancel()
	if err != nil {
		return "", fmt.Errorf("query failed with: %s", err.Error())
	}

	resultKey := response.Response.Key
	resultValue := response.Response.Value

	if string(resultKey) != key {
		return "", fmt.Errorf("Unexpected key in response: Expected %s, Got: %s",
			key, resultKey)
	}

	// decode value
	return string(resultValue), nil
}

func sendKVtransaction(key, value string) error {

	query := fmt.Sprintf(`%s:%d/broadcast_tx_commit?tx="%s=%s"`,
		HOST, CometGrpcPort, key, value)

	cmd := exec.Command("curl", query)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Println("Command failed: ", cmd)
		return fmt.Errorf("error sending query to kv store: %v, '%s'", err, string(out))
	}
	return err

}

// Query KV store to get the value for a given key
func queryKVStore(key string) (string, error) {
	query := fmt.Sprintf(`%s:%d/abci_query?data="%s"`, HOST, CometGrpcPort, key)
	cmd := exec.Command("curl", "-s", query)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("error checking transaction: %v, %s", err, string(out))
	}

	response := RpcResponse{}
	err = json.Unmarshal(out, &response)
	if err != nil {
		panic(fmt.Sprintf("error unmarshalling query response %+v", err))
	}
	resultKey := response.Result.Response.Key
	resultValue := response.Result.Response.Value

	// Key and value in response are base64 encoded.
	// decode key
	dst := make([]byte, base64.StdEncoding.DecodedLen(len(resultKey)))
	n, err := base64.StdEncoding.Decode(dst, []byte(resultKey))
	if err != nil {
		return "", fmt.Errorf("error decoding key: %v", err)
	}
	dst = dst[:n]

	if string(dst) != key {
		return "", fmt.Errorf("Unexpected key in response: Expected %s, Got: %s",
			key, string(dst))
	}

	// decode value
	dst = make([]byte, base64.StdEncoding.DecodedLen(len(resultValue)))
	n, err = base64.StdEncoding.Decode(dst, []byte(resultValue))
	if err != nil {
		return "", fmt.Errorf("error decoding value: %v", err)
	}
	dst = dst[:n]

	return string(dst), nil
}

// Basic test to check that the transaction was performed successfully on KVStore application
func TestBasicKVwithCometBFT(t *testing.T) {
	// start applications
	app := createKVStore()
	app.Init()
	err := app.Start()
	defer stopApplications([]ChainApp{app})
	if err != nil {
		t.Errorf("error starting KV Store: %v", err)
		return
	}

	// start cometBFT
	CometBFT, err := startCometBFT(app.GetAddress())
	if err != nil {
		t.Errorf("failed starting cometBFT: %v", err)
		return
	}
	defer terminateCometBFT(CometBFT)

	// Set an entry in the KVStore
	key := "myKey"
	value := "someValue"
	err = sendKVtransaction(key, value)
	if err != nil {
		t.Errorf("Send transaction failed: %v", err)
		return
	}

	// Check transaction was successful
	result, err := queryKVStore(key)
	if err != nil {
		t.Errorf("error querying KV store: %v", err)
		return
	}
	if result != value {
		t.Errorf("Unexpected result for value: Expected %s, Got: %s", value, result)
		return
	}
}

func TestBasicKVwithCosMux(t *testing.T) {

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

	client, err := Client(HOST, fmt.Sprint(CometGrpcPort))
	if err != nil {
		t.Errorf("error creating client: %v", err)
		return

	}

	// Set an entry in the KVStore
	key := "cosmux"
	value := "muxTestEntry"
	appID := uint8(1)
	err = sendKvMBtransaction(client, appID, key, value)
	if err != nil {
		t.Errorf("Send transaction failed: %v", err)
		return
	}

	// Check transaction was successful
	start := time.Now()
	timeout := time.Second * 11
	for {
		result, err := queryMbKVStore(client, appID, key)
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
