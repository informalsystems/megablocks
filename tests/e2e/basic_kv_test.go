package e2e

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"testing"

	"github.com/cometbft/cometbft/rpc/client/http"
	comettypes "github.com/cometbft/cometbft/types"
)

var (
	HOST string = "localhost"
)

type kvEntry struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type RpcResult struct {
	Response kvEntry `json: "response"`
}
type RpcResponse struct {
	Result RpcResult `json: "result"`
}

func sendTx(client *http.HTTP, tx comettypes.Tx) error {
	ctx, loadCancel := context.WithCancel(context.Background())
	defer loadCancel()
	fmt.Println("Sending transaction")
	if _, err := client.BroadcastTxSync(ctx, tx); err != nil {
		return fmt.Errorf("error sending Tx to %v: %s", client, err.Error())
	}
	fmt.Println("Sent transaction")
	return nil
}

// Set a key/value on the store
func sendKvMBtransaction(client *http.HTTP, id uint8, key, value string) error {
	tx := comettypes.Tx{0x23, 0x6d, 0x75, 0x78} //MAGIC
	tx = append(tx, id)
	tx = append(tx, []byte(fmt.Sprintf("%s=%s", key, value))...)
	return sendTx(client, tx)
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
	kvStore, err := startKVStore()
	if err != nil {
		t.Errorf("error starting apps: %v", err)
		return
	}
	defer stopApplications([]*exec.Cmd{kvStore})

	// start cometBFT
	CometBFT, err = startCometBFT()
	if err != nil {
		err = fmt.Errorf("failed starting cometBFT: %v", err)
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

	if result != value {
		t.Errorf("Unexpected result for value: Expected %s, Got: %s", value, result)
		return
	}
}

func TestBasicKVwithCosMux(t *testing.T) {
	// start applications

	kvStore, err := startKVStore()
	if err != nil {
		t.Errorf("error starting apps: %v", err)
		return
	}
	defer stopApplications([]*exec.Cmd{kvStore})

	// start multiplexer
	cosmux, err := startCosMux()
	if err != nil {
		t.Errorf("failed starting cometBFT: %v", err)
		return
	}
	defer terminateCosMux(cosmux)

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
	result, err := queryMbKVStore(client, appID, key)
	if err != nil {
		t.Errorf("query failed with %s", err.Error())
		return
	}

	if result != value {
		t.Errorf("Unexpected result for value: Expected %s, Got: %s", value, result)
		return
	}
	t.Log("resulting value is:", result)

}
