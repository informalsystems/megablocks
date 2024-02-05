package e2e

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os/exec"
	"testing"
)

var (
	HOST string = "localhost"
)

// Set a key/value on the store
func sendKVtransaction(key, value string) error {
	query := fmt.Sprintf(`%s:%d/broadcast_tx_commit?tx="%s=%s"`,
		HOST, CometGrpcPort, key, value)

	cmd := exec.Command("curl", "-s", query)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error sending query to kv store: %v, %s", err, string(out))
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

	var result map[string]interface{}
	json.Unmarshal(out, &result)
	response := result["result"].(map[string]interface{})["response"].(map[string]interface{})
	resultKey := response["key"].(string)
	resultValue := response["value"].(string)

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
