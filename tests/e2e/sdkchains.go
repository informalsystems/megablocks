package e2e

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Balance struct {
	Balance struct {
		Amount string `json:"amount"`
		Denom  string `json:"denom"`
	} `json:"balance"`
}

type ChainApp interface {
	Init() error
	Start() error
	Stop() error
	GetChainID() string
	GetAddress() string
	GetAddressType() string
	GetHome() string
}

type SdkApp struct {
	ChainID     string
	Home        string
	Binary      string
	Address     string
	AddressType string
	LogFile     *os.File
	Command     *exec.Cmd
	Moniker     string
	NodeKey     string
	Keys        []string
}

func createSdkApp() *SdkApp {
	return &SdkApp{
		ChainID:     "sdk-app-2",
		Home:        "/tmp/sdk-app-2", //TODO: Move back to : filepath.Join(os.TempDir(), "sdk-app-2"),
		Binary:      "../../app/sdk-chain-a/cmd/minid/minid",
		Address:     "/tmp/mind.sock",
		AddressType: "socket",
		Command:     nil,
		Moniker:     "minid",
		NodeKey:     "alice",
		Keys:        []string{"bob", "alice", "carol"},
	}
}

func (app *SdkApp) UpdateGenesis() error {

	// Update Genesis
	// 	${APP_BIN} genesis add-genesis-account ${NODE_KEY} 10000000stake --keyring-backend test --home ${NODE_DIR}
	stake := "100000000stake"
	out, err := exec.Command(app.Binary,
		"genesis", "add-genesis-account", app.NodeKey, stake,
		"--chain-id", app.ChainID,
		"--keyring-backend", "test",
		"--home", app.Home).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed adding genesis account: %v, %s", err, string(out))
	}

	// create default validator
	// ${APP_BIN} genesis gentx ${NODE_KEY} 1000000stake --chain-id ${CHAIN_ID} --home ${NODE_DIR}
	genTxDir := filepath.Join(app.Home, "/config/gentx")
	cmd := exec.Command(app.Binary,
		"genesis", "gentx", app.NodeKey, "70000000stake",
		"--keyring-backend", "test",
		"--home", app.Home,
		"--chain-id", app.ChainID,
	)
	out, err = cmd.CombinedOutput()
	if err != nil {
		log.Println("Command failed:", cmd)
		return fmt.Errorf("failed generating a genesis tx '%v': %s", err, string(out))
	}

	// ${APP_BIN} genesis collect-gentxs --home ${NODE_DIR} --gentx-dir ${NODE_DIR}/config/gentx/
	out, err = exec.Command(app.Binary,
		"genesis", "collect-gentxs",
		"--home", app.Home,
		"--gentx-dir", genTxDir).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed creating default validator '%v': %s", err, string(out))
	}

	// Fix SDK issue generating initial height with wrong json data-type
	//	sed -i -e 's/"initial_height": 1/"initial_height": "1"/g' ${NODE_DIR}/config/genesis.json
	genesisFile := filepath.Join(app.Home, "config/genesis.json")
	out, err = exec.Command("sed", "-i", "-e", `s/"initial_height": 1/"initial_height": "1"/g`,
		genesisFile).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed fixing genesis file '%v': %s", err, string(out))
	}
	return err
}

func (app *SdkApp) Configure() error {

	// 	${APP_BIN}  config set client chain-id ${CHAIN_ID} --home ${NODE_DIR}
	out, err := exec.Command(app.Binary, "config", "set", "client", "chain-id", app.ChainID,
		"--home", app.Home).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed setting config on chain '%v': %s", err, string(out))
	}

	//	${APP_BIN}  config set client keyring-backend test --home ${NODE_DIR}
	out, err = exec.Command(app.Binary, "config", "set", "client", "keyring-backend", "test",
		"--home", app.Home).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed setting config for keyring-backend '%v': %s", err, string(out))
	}

	// ${APP_BIN} keys add ${NODE_KEY} --home ${NODE_DIR}
	for _, key := range app.Keys {
		if err = app.AddKey(key); err != nil {
			return err
		}
	}
	return err
}

// AddKey adds a key to the keyring
func (app *SdkApp) AddKey(name string) error {
	cmd := exec.Command(app.Binary, "keys", "add", name,
		"--keyring-backend", "test",
		"--home", app.Home)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed adding key '%v': %s", err, string(out))
	}
	return nil

}

func (app *SdkApp) Init() error {
	if app.Home == "" {
		app.Home = filepath.Join(os.TempDir(), "sdk-chain-2")
	}

	if _, err := CreateHomeDirectory(app.Home); err != nil {
		return err
	}

	if app.LogFile == nil {
		log := filepath.Join(app.Home, "sdk-chain-a.log")
		logFile, err := os.Create(log)
		if err != nil {
			return fmt.Errorf("error creating log file for sdk-chain-a: %v", err)
		}
		app.LogFile = logFile
	}

	// configure chain
	if err := app.Configure(); err != nil {
		return err
	}

	// init chain
	out, err := exec.Command(app.Binary, "init", app.Moniker, "--chain-id", app.ChainID, "--home", app.Home).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to initialize chain '%v': %s", err, string(out))
	}

	return app.UpdateGenesis()

}

func (app *SdkApp) GetAddress() string {
	sdkAddress := ""
	switch app.AddressType {
	case "socket":
		sdkAddress = "unix://" + app.Address
	default:
		// for now we just support socket connections to chain apps
		panic("address unknown. add support in e2e tests")
	}
	return sdkAddress
}

func (app *SdkApp) Start() error {

	app.Command = exec.Command(app.Binary, "start",
		"--address", app.GetAddress(),
		"--rpc.laddr", "tcp://127.0.0.1:26657",
		"--grpc.address", "127.0.0.1:9091",
		"--p2p.laddr", "tcp://127.0.0.1:26656",
		"--grpc-web.enable=false",
		"--log_level=trace",
		"--trace",
		"--log_no_color",
		"--with-comet=false",
		"--home", app.Home,
	)

	app.Command.Stdout = app.LogFile
	err := app.Command.Start()
	if err != nil {
		return fmt.Errorf("error starting sdk-chain application %v: %v", app.Command, err)
	}

	// wait until chain its up
	err = app.waitForChain()
	if err != nil {
		app.Stop()
		return err
	}

	fmt.Printf("Started sdk-chain-a. PID: %d, Logs: %s\n", app.Command.Process.Pid, app.LogFile.Name())
	return nil
}

func (app *SdkApp) Stop() error {
	if app.Command == nil {
		return nil
	}
	cmd := app.Command
	cmd.Process.Signal(os.Interrupt)
	err := cmd.Wait()
	if err != nil {
		fmt.Println("error stopping ", app, err)
		cmd.Process.Kill()
	}

	// cleanup socket in case it's still there
	if app.AddressType == "socket" {
		_, err := os.Stat(app.Address)
		if err != nil {
			err = os.Remove(app.Address)
			return fmt.Errorf("error removing unix socket %s: %v", app.Address, err)
		}
	}
	return nil
}

func (app *SdkApp) waitForChain() error {
	var err error
	// check if it's reachable
	startTime := time.Now()
	for {
		connection, rc := net.Dial("unix", app.Address)
		if rc == nil {
			fmt.Println("sdk-chain-a is up!")
			connection.Close()
			return rc
		}
		elapsed := time.Since(startTime)
		if elapsed > time.Second*20 {
			err = fmt.Errorf("chain %s not reachable on %s", app.ChainID, app.Address)
			break
		}
		time.Sleep(time.Millisecond * 200)
	}
	return err
}

func (app *SdkApp) GetLatestBlockHeight() (int, error) {
	var (
		status struct {
			SyncInfo struct {
				LatestBlockHeight string `json:"latest_block_height"`
			} `json:"sync_info"`
		}
		err           error
		currentHeight int
	)

	// Get latest_block_height from node status
	out, err := exec.Command(app.Binary, "--home", app.Home, "status").CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("failed getting node status (%v): %s", err, string(out))
	}
	if err = json.Unmarshal(out, &status); err != nil {
		return 0, fmt.Errorf("error unmarshalling node status: %v", err)
	}
	if currentHeight, err = strconv.Atoi(status.SyncInfo.LatestBlockHeight); err != nil {
		return 0, fmt.Errorf("error converting latest_block_height to integer: %v", err)
	}
	return currentHeight, nil
}

// Wait until a minimal block height is reached
func (app *SdkApp) waitForBlockHeight(height int, timeout time.Duration) error {
	startTime := time.Now()
	currentHeight := 0
	for {
		elapsed := time.Since(startTime)
		if elapsed > timeout {
			return fmt.Errorf("timed out at height=%d while waiting for height %d", currentHeight, height)
		}

		currentHeight, err := app.GetLatestBlockHeight()
		if err != nil {
			log.Println("error getting latest block...", err, "retrying")
			continue
		}
		time.Sleep(time.Millisecond * 100)
		if currentHeight >= height {
			return nil
		}

	}
}

func (app *SdkApp) GetBalanceWithChainID(address, denom, chainID string) (int, error) {
	app.waitForBlockHeight(1, BlockTime)
	out, err := exec.Command(app.Binary, "query", "bank", "balance", address, denom,
		"--home", app.Home,
		"--chain-id", chainID,
		"--log_level=trace",
		"--output", "json",
	).CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("error getting balances for '%s' (%v): %s",
			address, err, string(out))
	}

	balance := Balance{}
	if err = json.Unmarshal(out, &balance); err != nil {
		return 0, fmt.Errorf("error unmarshalling balance (%v): %s", err, out)
	}

	amount, err := strconv.Atoi(balance.Balance.Amount)
	if err != nil {
		return 0, fmt.Errorf("error converting balance %s: %v", balance.Balance.Amount, err)
	}
	return amount, nil
}

// GetBalance returns bank balance for a given address
func (app *SdkApp) GetBalance(address, denom string) (int, error) {
	return app.GetBalanceWithChainID(address, denom, app.ChainID)
}

func (app *SdkApp) SendBankTransaction(fromAddress, toAddress, amount string) error {
	// wait for  blockheight >= 1 before sending transaction
	err := app.waitForBlockHeight(1, BlockTime)
	if err != nil {
		return err
	}

	cmd := exec.Command(app.Binary,
		"--home", app.Home,
		"--chain-id", app.ChainID,
		"--gas", "auto",
		"--keyring-backend", "test",
		"--log_level=trace",
		"tx", "bank", "send", fromAddress, toAddress, amount,
		"--yes",
	)

	log.Println("Running command:", cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Println("failed running command:", cmd)
		return fmt.Errorf("failed sending bank transaction '%v': %s",
			err, string(out))
	} else {
		log.Println("Sent transaction", string(out))
	}
	return err
}

// GetUserAddress returns the addrs for given user
func (app *SdkApp) GetUserAddress(user string) (string, error) {
	out, err := exec.Command(app.Binary,
		"keys", "show", user,
		"--address",
		"--keyring-backend", "test",
		"--home", app.Home).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed getting keys for '%s' (%v): %s",
			user, err, string(out))
	}
	return strings.TrimSpace(string(out)), err
}

func (app *SdkApp) GetChainID() string {
	return app.ChainID
}

func (app *SdkApp) GetAddressType() string {
	return app.AddressType
}

func (app *SdkApp) GetHome() string {
	return app.Home
}
