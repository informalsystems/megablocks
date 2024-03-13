package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	cfg "github.com/cometbft/cometbft/config"
	cmtflags "github.com/cometbft/cometbft/libs/cli/flags"
	cmtlog "github.com/cometbft/cometbft/libs/log"
	"github.com/cometbft/cometbft/node"
	"github.com/cometbft/cometbft/p2p"
	"github.com/cometbft/cometbft/privval"
	"github.com/cometbft/cometbft/proxy"
	"github.com/spf13/viper"
)

var (
	homeDir string
	verbose bool
)

type MegaBlockApp struct {
	ID             uint8 // app identifier used to route tx
	Address        string
	ConnectionType string
	ChainID        string
}

// ChainApps is a list of applications handled by Multiplexer
// TODO: get this from a config file
var ChainApps []MegaBlockApp = []MegaBlockApp{
	// KV-Store-Chain
	{
		ID:             1,
		Address:        "unix:///tmp/kvapp.sock",
		ConnectionType: "socket",
		ChainID:        "KVStore",
	},

	// SDK-App-1
	{
		ID: 2,
		//Address:        "127.0.0.1:26658",
		Address:        "unix:///tmp/mind.sock",
		ConnectionType: "socket",
		ChainID:        "sdk-app-2",
	},
}

func init() {
	flag.StringVar(&homeDir, "cmt-home", "", "Path to the CometBFT config directory (if empty, uses $HOME/.cometbft)")
	flag.BoolVar(&verbose, "v", false, "verbose")
}

// configureCometBFT creates default config in specified cometBFT home directory
func configureCometBFT(homeDir string) *cfg.Config {
	config := cfg.DefaultConfig()
	config.SetRoot(homeDir)
	viper.SetConfigFile(fmt.Sprintf("%s/%s", homeDir, "config/config.toml"))

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Reading config: %v", err)
	}
	if err := viper.Unmarshal(config); err != nil {
		log.Fatalf("Decoding config: %v", err)
	}
	if err := config.ValidateBasic(); err != nil {
		log.Fatalf("Invalid configuration data: %v", err)
	}

	return config
}

func main() {
	flag.Parse()

	if homeDir == "" {
		homeDir = os.ExpandEnv("$HOME/.cosmux")
	}
	config := configureCometBFT(homeDir)

	// override loglevel config if requested
	if verbose {
		config.LogLevel = "debug" //"*:error,p2p:info,state:info"
	}

	// Create Multiplexer Shim
	muxCfg := CosmuxConfig{LogLevel: config.LogLevel}
	cosmux := NewMultiplexer(muxCfg)

	// Register applications
	for _, app := range ChainApps {
		if err := cosmux.AddApplication(app); err != nil {
			log.Fatalf("error registering chain application: %v", err)
		}
	}

	if err := cosmux.Start(); err != nil {
		log.Fatalf("error starting cosmux; %v", err)
	}

	// use private validator to sign consensus messages
	pv := privval.LoadFilePV(
		config.PrivValidatorKeyFile(),
		config.PrivValidatorStateFile(),
	)

	// nodeKey is needed to identify the node in a p2p network
	nodeKey, err := p2p.LoadNodeKey(config.NodeKeyFile())
	if err != nil {
		log.Fatalf("failed to load node's key: %v", err)
	}

	logger := cmtlog.NewTMLogger(cmtlog.NewSyncWriter(os.Stdout))
	logger, err = cmtflags.ParseLogLevel(config.LogLevel, logger, cfg.DefaultLogLevel)
	if err != nil {
		log.Fatalf("failed to parse log level: %v", err)
	}

	clientCreator := proxy.NewConnSyncLocalClientCreator(cosmux)
	node, err := node.NewNode(
		config,
		pv,
		nodeKey,
		clientCreator,
		node.DefaultGenesisDocProviderFunc(config),
		cfg.DefaultDBProvider,
		node.DefaultMetricsProvider(config.Instrumentation),
		logger,
	)
	if err != nil {
		log.Fatalf("error creating node: %v", err)
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
