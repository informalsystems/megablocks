package e2e

import (
	"context"
	"crypto/sha1"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/cometbft/cometbft/rpc/client/http"
	rpchttp "github.com/cometbft/cometbft/rpc/client/http"
	comettypes "github.com/cometbft/cometbft/types"
)

var (
	BlockTime time.Duration = time.Second * 3
)

// createMegablocksHeader creates the header needed for all transactions of megablocks
// applications.
func CreateMegablocksHeader(chainID string) []byte {
	// The Megablocks header is based on a 4-bytes Magic + 4-bytes truncated sha1 of the chain ID
	// of the application.
	tx := comettypes.Tx{0x23, 0x6d, 0x75, 0x78} // Megablocks MAGIC

	// create
	checksum := sha1.Sum([]byte(chainID))
	megablocks_id := checksum[:4]
	return append(tx, megablocks_id...)
}

func Client(ip, proxyPort string) (*rpchttp.HTTP, error) {
	return rpchttp.New(fmt.Sprintf("http://%s:%v", ip, proxyPort), "/websocket")
}

type KeyValEntry struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type RpcResult struct {
	Response KeyValEntry `json: "response"`
}
type RpcResponse struct {
	Result RpcResult `json: "result"`
}

// SendTx broadcast a transaction on a client connection
func SendTx(client *http.HTTP, tx comettypes.Tx) error {
	ctx, loadCancel := context.WithCancel(context.Background())
	defer loadCancel()
	fmt.Println("Sending transaction", tx)
	if _, err := client.BroadcastTxSync(ctx, tx); err != nil {
		return fmt.Errorf("error sending Tx to %v: %s", client, err.Error())
	}
	fmt.Println("Sent transaction", tx)
	return nil
}

// startApplications starts all chain applications
// returns list of cmd pointers chain app processes
func startApplications(apps ...ChainApp) ([]ChainApp, error) {
	if len(apps) == 0 {
		apps = append(apps, createSdkApp(), createKVStore())
	}
	started := []ChainApp{}

	for idx, _ := range apps {
		err := apps[idx].Init()
		if err != nil {
			stopApplications(apps)
			return nil, err
		}
		if err = apps[idx].Start(); err != nil {
			stopApplications(apps)
			return nil, err
		}
		started = append(started, apps[idx])
	}
	return started, nil
}

// stopApplications stops all provided apps
func stopApplications(apps []ChainApp) error {
	fmt.Println("stopping applications:", apps)
	var err error

	for _, app := range apps {
		rc := app.Stop()
		if rc != nil {
			err = rc
		}
	}
	return err
}

// buildApplications triggers build of all executables
// controlled by Makefile in project root
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

// DumpLog prints tail of a given logfile to stdout
func DumpLog(logfile string) {
	file, err := os.Open(logfile)
	if err != nil {
		log.Printf("error dumping logs when reading logfile %s: %v\n", logfile, err)
		return
	}
	defer file.Close()
	fileInfo, err := file.Stat()
	if err != nil {
		log.Printf("error dumping logs when getting stats for logfile %s: %v\n", logfile, err)
		return
	}

	buf := make([]byte, 800)
	offset := int64(0)
	if fileInfo.Size() > int64(len(buf)) {
		offset = fileInfo.Size() - int64(len(buf))
	}
	_, err = file.ReadAt(buf, offset)
	if err == nil || err == io.EOF {
		header := fmt.Sprintf("\n\n========= Tail of LogFile %s ========\n", logfile)
		footer := "\n=======================================\n\n"
		log.Println(header, string(buf), footer)
	} else {
		log.Printf("error dumping logs %s: %v", logfile, err)
	}
}

func CreateHomeDirectory(dirPath string) (string, error) {
	var err error = nil
	if dirPath == "" {
		if dirPath, err = os.MkdirTemp(os.TempDir(), "mega-blocks"); err != nil {
			return "", fmt.Errorf("error creating home directory: %v", err)
		}
	}

	// delete existing home
	_, err = os.Stat(dirPath)
	if err == nil {
		fmt.Println("Deleting existing home :", dirPath)
		if err = os.RemoveAll(dirPath); err != nil {
			return "", fmt.Errorf("error deleting home directory %s: %v", dirPath, err)
		}
	}

	if err = os.MkdirAll(dirPath, 0770); err != nil {
		return "", fmt.Errorf("error creating home directory %s: %v", dirPath, err)
	}

	return dirPath, err
}
