package e2e

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

type CosMux struct {
	Home       string
	Binary     string
	LogFile    *os.File
	Command    *exec.Cmd
	Config     string
	ConfigFile string
}

func CreateConfigFromChain(home string, apps []ChainApp) string {
	toml := `
log_level = "debug"
	`
	appTemplate := `
[[apps]]
  Address =        "%s"
  ConnectionType = "%s"
  ChainID = "%s"
  Home = "%s"

`
	for _, app := range apps {
		toml += fmt.Sprintf(appTemplate, app.GetAddress(), app.GetAddressType(), app.GetChainID(), app.GetHome())
	}
	return toml
}

func createCosMux(home string, apps []ChainApp) *CosMux {
	return &CosMux{
		Home:   home,
		Binary: "../../cosmux/cosmux",
		Config: CreateConfigFromChain(home, apps),
	}
}

func (app *CosMux) Init() error {
	if app.Home == "" {
		app.Home = CometHome

		// init CometBFT
		err := initCometBFT(app.Home)
		if err != nil {
			return err
		}

	}

	// write config file
	app.ConfigFile = filepath.Join(app.Home, "cosmux.toml")
	os.WriteFile(app.ConfigFile, []byte(app.Config), 0o644)

	var err error
	if app.LogFile == nil {
		log := filepath.Join(app.Home, "cosmux.log")
		logFile, err := os.Create(log)
		if err != nil {
			return fmt.Errorf("error creating log file for CosMux: %v", err)
		}
		app.LogFile = logFile
	}

	return err
}

// Start starts the Multiplexer shim
func (app *CosMux) Start() error {
	cmd := exec.Command(app.Binary, "-v", "--cmt-home", app.Home, "-c", app.ConfigFile)
	cmd.Stdout = app.LogFile
	// request to create process group to terminate created childrens as well
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	app.Command = cmd

	err := cmd.Start()
	if err != nil {
		return fmt.Errorf("error starting CosMux: %v", err)
	}

	fmt.Printf("Started CosMux. PID: %d, Logs: %s\n", cmd.Process.Pid, app.LogFile.Name())

	err = waitCometBFT()
	if err != nil {
		cmd.Process.Kill()
		log.Println("error running cosmux: ", cmd)
		DumpLog(app.LogFile.Name())
		return fmt.Errorf("starting CosMux failed: not reachable")
	}
	return err
}

func (app *CosMux) Stop() error {
	if app.Command == nil {
		return nil
	}

	fmt.Println("Terminating CosMux")
	if err := app.Command.Process.Kill(); err != nil {
		log.Println("error terminating cosmux ", err)
		return err
	}
	return nil
}
