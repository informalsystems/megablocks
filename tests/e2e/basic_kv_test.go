package e2e

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"testing"
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

// Set a key/value on the store
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
		err = fmt.Errorf("failed starting cometBFT: %v", err)
	}
	defer terminateCosMux(cosmux)

	// Set an entry in the KVStore
	key := "cosmux"
	value := "muxTestEntry"
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
	t.Log("resulting value is:", result)

}
