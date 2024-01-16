package main

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

var homeDir string
var socketAddr string

func init() {
	flag.StringVar(&homeDir, "kv-home", "", "Path to the kvstore directory (if empty, uses $HOME/.kvstore)")
	flag.StringVar(&socketAddr, "socket-addr", "unix://example.sock", "Unix domain socket address (if empty, uses \"unix://example.sock\"")
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
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("Closing database: %v", err)
		}
	}()

	app := NewKVStoreApplication(db)

	logger := cmtlog.NewTMLogger(cmtlog.NewSyncWriter(os.Stdout))

	server := abciserver.NewSocketServer(socketAddr, app)
	server.SetLogger(logger)

	if err := server.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "error starting socket server: %v", err)

		os.Exit(1)
	}
	defer server.Stop()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
}
