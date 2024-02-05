package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

var (
	KVSocket string = "unix:///tmp/kvapp.sock"
)

func startKVStore() (*exec.Cmd, error) {
	kvHome := filepath.Join(os.TempDir(), "kvHome")
	logFile, err := os.CreateTemp(os.TempDir(), "kvstorelog")
	if err != nil {
		return nil, fmt.Errorf("error creating logfile for kvstore: %v", err)
	}
	cmd := exec.Command("../../app/kvstore/kvstore",
		"-kv-home", kvHome,
		"-socket-addr", KVSocket)
	cmd.Stdout = logFile
	err = cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("error starting kvStore application %v: %v", cmd, err)
	}

	fmt.Printf("Started KVStore. PID:%d, Logs=%s\n", cmd.Process.Pid, logFile.Name())
	return cmd, nil
}
