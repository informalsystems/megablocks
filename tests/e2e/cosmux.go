package e2e

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"syscall"
)

// startCosMux starts the Multiplexer shim
func startCosMux() (*exec.Cmd, error) {
	logFile, err := os.CreateTemp(os.TempDir(), "CosMux")
	if err != nil {
		return nil, fmt.Errorf("error creating log file for CosMux: %v", err)
	}

	// init CometBFT
	err = initCometBFT(CometHome)
	if err != nil {
		return nil, err
	}

	cmd := exec.Command("../../cosmux/cosmux", "--cmt-home", CometHome)
	cmd.Stdout = logFile
	// request to create process group to terminate created childrens as well
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	err = cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("error starting CosMux: %v", err)
	}

	err = waitCometBFT()
	if err != nil {
		cmd.Process.Kill()
		log.Println("error running cosmux: ", cmd)
		return nil, fmt.Errorf("starting CosMux failed: not reachable")
	}
	fmt.Printf("Started CosMux. PID=%d, Logs=%s\n", cmd.Process.Pid, logFile.Name())
	return cmd, err
}

func terminateCosMux(cmd *exec.Cmd) error {
	if cmd == nil {
		return nil
	}

	fmt.Println("Terminating CosMux")
	if err := cmd.Process.Kill(); err != nil {
		log.Println("error terminating cosmux ", err)
		return err
	}
	return nil
}
