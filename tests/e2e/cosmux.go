package e2e

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"syscall"
)

type CosMux struct {
	Home    string
	Binary  string
	LogFile *os.File
	Command *exec.Cmd
}

func createCosMux(home string) *CosMux {
	return &CosMux{
		Home:   home,
		Binary: "../../cosmux/cosmux",
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

	logFile, err := os.CreateTemp(app.Home, "cosmux")
	if err != nil {
		return fmt.Errorf("error creating log file for CosMux: %v", err)
	}
	app.LogFile = logFile
	return err
}

// Start starts the Multiplexer shim
func (app *CosMux) Start() error {
	cmd := exec.Command(app.Binary, "-v", "--cmt-home", app.Home)
	cmd.Stdout = app.LogFile
	// request to create process group to terminate created childrens as well
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	app.Command = cmd

	err := cmd.Start()
	if err != nil {
		return fmt.Errorf("error starting CosMux: %v", err)
	}

	err = waitCometBFT()
	if err != nil {
		cmd.Process.Kill()
		log.Println("error running cosmux: ", cmd)
		DumpLog(app.LogFile.Name())
		return fmt.Errorf("starting CosMux failed: not reachable")
	}
	fmt.Printf("Started CosMux. PID=%d, Logs=%s\n", cmd.Process.Pid, app.LogFile.Name())
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
