package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type KvApp struct {
	ChainID     string
	Home        string
	Binary      string
	Address     string
	AddressType string
	LogFile     *os.File
	Command     *exec.Cmd
}

func createKVStore() *KvApp {

	app := KvApp{
		ChainID:     "KVStore",
		Home:        filepath.Join(os.TempDir(), "kvHome"),
		Binary:      "../../app/kvstore/kvstore",
		Address:     "/tmp/kvapp.sock",
		AddressType: "socket",
	}

	return &app
}

func (app *KvApp) GetAddress() string {
	switch app.AddressType {
	case "socket":
		return "unix://" + app.Address
	default:
		panic(fmt.Sprintf("Unsupported address type %s", app.AddressType))
	}
}

func (app *KvApp) Init() error {
	if app.Home == "" {
		app.Home = filepath.Join(os.TempDir(), "kvStore")
	}

	if _, err := CreateHomeDirectory(app.Home); err != nil {
		return err
	}

	if app.LogFile == nil {
		logFile, err := os.CreateTemp(app.Home, "kvstore.log")
		if err != nil {
			return fmt.Errorf("error creating logfile for kvstore: %v", err)
		}
		app.LogFile = logFile
	}
	return nil
}

func (app *KvApp) Start() error {
	cmd := exec.Command(app.Binary,
		"-kv-home", app.Home,
		"-v=3",
		"-socket-addr", app.GetAddress())
	cmd.Stdout = app.LogFile
	cmd.Stderr = app.LogFile
	app.Command = cmd
	err := cmd.Start()
	if err != nil {
		return fmt.Errorf("error starting kvStore application %v: %v", cmd, err)
	}

	fmt.Printf("Started KVStore. PID:%d, Logs=%s\n", cmd.Process.Pid, app.LogFile.Name())
	return nil
}

func (app *KvApp) Stop() error {
	// try a graceful termination of the process
	if app.Command == nil {
		return nil
	}

	cmd := app.Command
	cmd.Process.Signal(os.Interrupt)
	err := cmd.Wait()
	if err != nil {
		DumpLog(app.LogFile.Name())
		fmt.Println("error stopping ", app, err)
		cmd.Process.Kill()
	}
	return err
}
