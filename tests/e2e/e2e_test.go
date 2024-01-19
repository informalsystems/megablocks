package e2e

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

var (
	ChainApps        []*exec.Cmd
	CometBFT         *exec.Cmd = nil
	CometURL         string    = "github.com/cometbft/cometbft/cmd/cometbft@v0.38.0"
	CometGrpcPort    int       = 26657
	CometGrpcAddress string    = fmt.Sprintf("127.0.0.1:%d", CometGrpcPort)
	CometGrpcURL     string    = fmt.Sprintf("tcp://%s", CometGrpcAddress)
	CometHome                  = filepath.Join(os.TempDir(), "cometbft-home")
)

// Basic test to check that the transaction was performed successfully on KVStore application
func TestBasicKVoperation(t *testing.T) {
	// Set a key/value on the store
	host := "localhost"
	key := "megablock"
	value := "rocks"
	query := fmt.Sprintf(`%s:%d/broadcast_tx_commit?tx="%s=%s"`, host, CometGrpcPort, key, value)

	cmd := exec.Command("curl", "-s", query)
	t.Log("running tx: ", cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Error("error sending query to kv store", err, string(out))
		time.Sleep(time.Second * 60)
		return
	}

	// Check transaction was successful
	query = fmt.Sprintf(`%s:%d/abci_query?data="%s"`, host, CometGrpcPort, key)
	cmd = exec.Command("curl", "-s", query)
	t.Log("running query: ", cmd)
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Error("error checking transaction:", err, string(out))
		return
	}

	var result map[string]interface{}
	json.Unmarshal(out, &result)
	response := result["result"].(map[string]interface{})["response"].(map[string]interface{})
	resultKey := response["key"].(string)
	resultValue := response["value"].(string)

	// decode key
	dst := make([]byte, base64.StdEncoding.DecodedLen(len(resultKey)))
	n, err := base64.StdEncoding.Decode(dst, []byte(resultKey))
	if err != nil {
		t.Error("decode error:", err)
		return
	}
	dst = dst[:n]
	if string(dst) != key {
		t.Errorf("Unexpected result for key: Expected %s, Got: %s", key, dst)
		return
	}
	t.Log("resulting key is:", string(dst))

	dst = make([]byte, base64.StdEncoding.DecodedLen(len(resultValue)))
	n, err = base64.StdEncoding.Decode(dst, []byte(resultValue))
	if err != nil {
		t.Error("decode error:", err)
		return
	}
	dst = dst[:n]
	if string(dst) != value {
		t.Errorf("Unexpected result for value: Expected %s, Got: %s", key, dst)
		return
	}
	t.Log("resulting value is:", string(dst))
}

func parseArguments() error {
	var err error
	flag.Parse()
	return err
}

func startKVStore() (*exec.Cmd, error) {
	kvHome := filepath.Join(os.TempDir(), "kvHome")
	logFile, err := os.CreateTemp(os.TempDir(), "kvstorelog")
	if err != nil {
		return nil, fmt.Errorf("error creating logfile for kvstore: %v", err)
	}
	cmd := exec.Command("../../app/kvstore/kvstore", "-kv-home", kvHome)
	cmd.Stdout = logFile
	err = cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("error starting kvStore application %v: %v", cmd, err)
	}
	fmt.Printf("Started KVStore. PID:%d, Logs=%s\n", cmd.Process.Pid, logFile.Name())
	return cmd, nil
}

func startApplications() ([]*exec.Cmd, error) {
	apps := []*exec.Cmd{}
	cmd, err := startKVStore()
	if err == nil {
		apps = append(ChainApps, cmd)
	}
	return apps, err
}

func initCometBFT(cometHome string) error {
	fmt.Println("Initializing CometBFT")
	cmd := exec.Command("go", "run", CometURL, "init", "--home", cometHome)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error initializing cometBFT: %v, %s", err, string(out))
	}
	return err
}

func waitCometBFT() error {
	var err error
	// check if it's reachable
	startTime := time.Now()
	for {
		connection, rc := net.Dial("tcp", CometGrpcAddress)
		if rc == nil {
			fmt.Println("CometBFT is up!")
			connection.Close()
			return rc
		}
		elapsed := time.Since(startTime)
		if elapsed > time.Second*20 {
			err = fmt.Errorf("CometBFT not reachable on %s", CometGrpcAddress)
			break
		}
		time.Sleep(time.Millisecond * 200)
	}
	return err
}

func startCometBFT() (*exec.Cmd, error) {
	logFile, err := os.CreateTemp(os.TempDir(), "cometBFT")
	if err != nil {
		return nil, fmt.Errorf("error creating log file for cometBFT: %v", err)
	}

	// init CometBFT
	err = initCometBFT(CometHome)
	if err != nil {
		return nil, err
	}

	cmd := exec.Command("go", "run", CometURL, "node", "--home", CometHome, "--rpc.laddr", CometGrpcURL, "--proxy_app", "unix://example.sock")
	cmd.Stdout = logFile
	// request to create process group to terminate created childrens as well
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	err = cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("error starting CometBFT: %v", err)
	}

	err = waitCometBFT()
	if err != nil {
		fmt.Println("Terminating CometBFT")
		if rc := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); rc != nil {
			log.Println("error terminating cometBFT ", rc)
		}
		return nil, err
	}
	fmt.Printf("Started CometBFT. PID=%d, Logs=%s\n", cmd.Process.Pid, logFile.Name())
	return cmd, err
}

func terminateCometBFT(comet *exec.Cmd) error {
	fmt.Println("Terminating CometBFT")
	pid := fmt.Sprintf("%d", comet.Process.Pid)
	cmd := exec.Command("go", "run", CometURL, "debug", "kill", pid, "/dev/null", "--home", CometHome, "--rpc-laddr", CometGrpcURL)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Println("Failed terminating CometBFT running:", cmd)
		return fmt.Errorf("error terminating CometBFT: %v, %s", err, string(out))
	}
	return nil
}

func buildApplications() error {
	fmt.Println("building applications")
	cmd := exec.Command("make", "build")
	cmd.Dir = "../../"
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error building applications: %s", string(out))
	}
	return err
}

func setup() error {
	fmt.Println("Setting up environment")

	// build applications
	err := buildApplications()
	if err != nil {
		return err
	}

	// start applications
	ChainApps, err = startApplications()
	if err != nil {
		return fmt.Errorf("error starting apps: %v", err)
	}
	// start cometBFT
	CometBFT, err = startCometBFT()
	if err != nil {
		err = fmt.Errorf("failed starting cometBFT: %v", err)
	}
	return err
}

func teardown() {
	fmt.Println("Tearing down environment")
	apps := ChainApps
	if CometBFT != nil {
		if err := terminateCometBFT(CometBFT); err != nil {
			log.Println("Error terminating CometBFT: ", err)
		}
	}
	for _, app := range apps {
		app.Process.Signal(os.Interrupt)
		err := app.Wait()
		if err != nil {
			fmt.Println("error stopping ", app)
			app.Process.Kill()
		}
	}
}

// runs E2E tests
func TestMain(m *testing.M) {

	if err := parseArguments(); err != nil {
		flag.Usage()
		log.Fatalf("Error parsing command arguments %s\n", err)
	}

	err := setup()
	if err != nil {
		fmt.Println("Failed setting up environment: ", err)
		teardown()
		os.Exit(-1)
	}

	rc := m.Run()
	teardown()
	os.Exit(rc)
}
