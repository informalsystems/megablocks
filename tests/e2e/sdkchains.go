package e2e

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type ChainApp interface {
	Init() error
	Start() error
	Stop() error
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
}

func createSdkApp() *SdkApp {
	return &SdkApp{
		ChainID:     "sdk-app-2",
		Home:        filepath.Join(os.TempDir(), "sdk-app-2"),
		Binary:      "../../app/sdk-chain-a/cmd/minid/minid",
		Address:     "/tmp/mind.sock",
		AddressType: "socket",
		Command:     nil,
		Moniker:     "minid",
		NodeKey:     "alice",
	}
}

func (app *SdkApp) UpdateGenesis() error {

	// Update Genesis
	// 	${APP_BIN} genesis add-genesis-account ${NODE_KEY} 10000000stake --keyring-backend test --home ${NODE_DIR}
	stake := "10000000stake"
	out, err := exec.Command(app.Binary, "genesis", "add-genesis-account", app.NodeKey, stake,
		"--keyring-backend", "test", "--home", app.Home).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed adding genesis account: %v, %s", err, string(out))
	}

	// create default validator
	// ${APP_BIN} genesis gentx ${NODE_KEY} 1000000stake --chain-id ${CHAIN_ID} --home ${NODE_DIR}
	genTxDir := filepath.Join(app.Home, "/config/gentx")
	cmd := exec.Command(app.Binary, "genesis", "gentx", app.NodeKey, stake,
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
		"genesis", "collect-gentxs", "--home", app.Home, "--gentx-dir", genTxDir).CombinedOutput()
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
	out, err = exec.Command(app.Binary, "keys", "add", app.NodeKey, "--keyring-backend", "test",
		"--home", app.Home).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed adding keys '%v': %s", err, string(out))
	}
	return nil
}

func (app *SdkApp) Init() error {
	if app.Home == "" {
		app.Home = filepath.Join(os.TempDir(), "sdk-chain-a")
	}

	if _, err := CreateHomeDirectory(app.Home); err != nil {
		return err
	}

	if app.LogFile == nil {
		logFile, err := os.CreateTemp(app.Home, "sdk-chain-a.log")
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
	//
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

	fmt.Printf("Started sdk-chain-a. PID:%d, Logs=%s\n", app.Command.Process.Pid, app.LogFile.Name())
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
