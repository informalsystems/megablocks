package e2e

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

var (
	CometHome                  = filepath.Join(os.TempDir(), "cometbft-home")
	CometBFT         *exec.Cmd = nil
	CometURL         string    = "github.com/cometbft/cometbft/cmd/cometbft@v0.38.0"
	CometGrpcPort    int       = 26657
	CometGrpcAddress string    = fmt.Sprintf("127.0.0.1:%d", CometGrpcPort)
	CometGrpcURL     string    = fmt.Sprintf("tcp://%s", CometGrpcAddress)
)

func initCometBFT(cometHome string) error {
	fmt.Println("Initializing CometBFT")
	_, err := os.Stat(CometHome)
	if err == nil {
		fmt.Println("Deleting existing comet home :", CometHome)
		err = os.RemoveAll(CometHome)
		if err != nil {
			return fmt.Errorf("error initializing cometBFT: %s", err.Error())
		}
	}
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

	cmd := exec.Command("go", "run", CometURL, "node", "--home", CometHome, "--rpc.laddr", CometGrpcURL, "--proxy_app", KVSocket)
	cmd.Stdout = logFile
	// request to create process group to terminate created childrens as well
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	err = cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("error starting CometBFT: %v", err)
	}

	err = waitCometBFT()
	if err != nil {
		_ = terminateCometBFT(cmd)
		return nil, err
	}
	fmt.Printf("Started CometBFT. PID=%d, Logs=%s\n", cmd.Process.Pid, logFile.Name())
	return cmd, err
}

// Kill cometBFT
func terminateCometBFT(cmd *exec.Cmd) error {
	if cmd == nil {
		return nil
	}

	fmt.Println("Terminating CometBFT")
	// CometBFT is started as process group so we need to send the signal
	// to each process in the group  (prefix  `-`)
	if rc := syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM); rc != nil {
		log.Println("error terminating cometBFT ", rc)
		return rc
	}
	return nil
}
