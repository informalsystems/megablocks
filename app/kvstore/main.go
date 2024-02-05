package main

//
// Example application for MegaBlocks implementing a KV store following the user guide
// of CometBFT (https://docs.cometbft.com/v0.38/guides/go) for applications running as
// separate process from CometBFT
//

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	abciserver "github.com/cometbft/cometbft/abci/server"
	cmtlog "github.com/cometbft/cometbft/libs/log"
	"github.com/dgraph-io/badger/v3"
)

var (
	homeDir    string
	socketAddr string
)

func init() {
	flag.StringVar(&homeDir, "kv-home", "", "Path to the kvstore directory (if empty, uses $HOME/.kvstore)")
	flag.StringVar(&socketAddr, "socket-addr", "unix://example.sock", "Unix domain socket address (if empty, uses \"unix://example.sock\"")
}

func closeDB(db *badger.DB) {
	if err := db.Close(); err != nil {
		log.Printf("Closing database: %v", err)
	}
}

func main() {
	flag.Parse()
	if homeDir == "" {
		homeDir = os.ExpandEnv("$HOME/.cometbft")
	}

	// initialize badger
	dbPath := filepath.Join(homeDir, "badger")
	db, err := badger.Open(badger.DefaultOptions(dbPath))
	if err != nil {
		log.Fatalf("Opening database: %v", err)
	}
	defer closeDB(db)

	app := NewKVStoreApplication(db)

	logger := cmtlog.NewTMLogger(cmtlog.NewSyncWriter(os.Stdout))

	server := abciserver.NewSocketServer(socketAddr, app)
	server.SetLogger(logger)

	if err := server.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "error starting socket server: %v", err)
		closeDB(db)
		//nolint  // ignore exitAfterDefer as closeDB called explicitly
		os.Exit(1)
	}
	defer server.Stop()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
}
