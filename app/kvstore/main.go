package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/cometbft/cometbft/p2p"
	"github.com/cometbft/cometbft/privval"
	"github.com/cometbft/cometbft/proxy"

	cfg "github.com/cometbft/cometbft/config"
	cmtflags "github.com/cometbft/cometbft/libs/cli/flags"
	cmtlog "github.com/cometbft/cometbft/libs/log"
	nm "github.com/cometbft/cometbft/node"
	"github.com/dgraph-io/badger/v3"
	"github.com/spf13/viper"
)

var homeDir string

func init() {
	flag.StringVar(&homeDir, "cmt-home", "", "Path to the CometBFT config directory (if empty, uses $HOME/.cometbft)")
}

// load config form specified directory
func configure(cfgDir string) (*cfg.Config, error) {
	config := cfg.DefaultConfig()
	config.SetRoot(cfgDir)
	viper.SetConfigFile(fmt.Sprintf("%s/%s", cfgDir, "config/config.toml"))

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("Reading config: %v", err)
	}
	if err := viper.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("Decoding config: %v", err)
	}
	if err := config.ValidateBasic(); err != nil {
		return nil, fmt.Errorf("Invalid configuration data: %v", err)
	}
	return config, nil
}

func main() {
	flag.Parse()
	if homeDir == "" {
		homeDir = os.ExpandEnv("$HOME/.cometbft")
	}

	config, err := configure(homeDir)
	if err != nil {
		log.Fatalf("Failed getting configuration: %v", err)
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

	pv := privval.LoadFilePV(
		config.PrivValidatorKeyFile(),
		config.PrivValidatorStateFile(),
	)

	nodeKey, err := p2p.LoadNodeKey(config.NodeKeyFile())
	if err != nil {
		log.Fatalf("failed to load node's key: %v", err)
	}

	logger := cmtlog.NewTMLogger(cmtlog.NewSyncWriter(os.Stdout))
	logger, err = cmtflags.ParseLogLevel(config.LogLevel, logger, cfg.DefaultLogLevel)

	if err != nil {
		log.Fatalf("failed to parse log level: %v", err)
	}

	node, err := nm.NewNode(
		config,
		pv,
		nodeKey,
		proxy.NewLocalClientCreator(app),
		nm.DefaultGenesisDocProviderFunc(config),
		cfg.DefaultDBProvider, //fixed
		nm.DefaultMetricsProvider(config.Instrumentation),
		logger,
	)

	if err != nil {
		log.Fatalf("Creating node: %v", err)
	}

	node.Start()
	defer func() {
		node.Stop()
		node.Wait()
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
}
