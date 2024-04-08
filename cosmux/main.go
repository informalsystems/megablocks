package main

import (
	"flag"
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
)

var (
	homeDir    string
	verbose    bool
	configFile string
	logLevel   string
)

type MegaBlockApp struct {
	//ID             uint8  // app identifier used to route tx
	Address        string //`mapstructure:"address"`
	ConnectionType string //`mapstructure:"connection_type"`
	ChainID        string //`mapstructure:"chain_id"`
	Home           string //`mapstructure:"home"`
}

// ChainApps is a list of applications handled by Multiplexer
// TODO: get this from a config file
var ChainApps []MegaBlockApp = []MegaBlockApp{
	// KV-Store-Chain
	{
		//ID:             1,
		Address:        "unix:///tmp/kvapp.sock",
		ConnectionType: "socket",
		ChainID:        "KVStore",
	},

	// SDK-App-1
	{
		//ID: 2,
		//Address:        "127.0.0.1:26658",
		Address:        "unix:///tmp/mind.sock",
		ConnectionType: "socket",
		ChainID:        "sdk-app-2",
	},
}

func init() {
	flag.StringVar(&homeDir, "cmt-home", "", "Path to the CometBFT config directory (if empty, uses $HOME/.cometbft)")
	flag.BoolVar(&verbose, "v", false, "verbose")
	flag.StringVar(&logLevel, "log_level", "comet-mux:debug", "Log level")
	flag.StringVar(&configFile, "c", "", "multiplexer config file")
}

func main() {
	flag.Parse()

	if homeDir == "" {
		homeDir = os.ExpandEnv("$HOME/.cosmux")
	}

	cometCfg := ConfigureCometBFT(homeDir)
	muxCfg := ConfigureCometMultiplexer(configFile)

	// override loglevel config if requested
	if verbose {
		cometCfg.LogLevel = "debug" //"*:error,p2p:info,state:info"
		muxCfg.LogLevel = "debug"
	} else {
		cometCfg.LogLevel = logLevel
		muxCfg.LogLevel = "debug"
	}

	// Create Multiplexer Shim
	cosmux := NewMultiplexer(muxCfg)
	if err := cosmux.Start(); err != nil {
		log.Fatalf("error starting cosmux; %v", err)
	}

	// use private validator to sign consensus messages
	pv := privval.LoadFilePV(
		cometCfg.PrivValidatorKeyFile(),
		cometCfg.PrivValidatorStateFile(),
	)

	// nodeKey is needed to identify the node in a p2p network
	nodeKey, err := p2p.LoadNodeKey(cometCfg.NodeKeyFile())
	if err != nil {
		log.Fatalf("failed to load node's key: %v", err)
	}

	logger := cmtlog.NewTMLogger(cmtlog.NewSyncWriter(os.Stdout))
	logger, err = cmtflags.ParseLogLevel(cometCfg.LogLevel, logger, cfg.DefaultLogLevel)
	if err != nil {
		log.Fatalf("failed to parse log level: %v", err)
	}

	clientCreator := proxy.NewConnSyncLocalClientCreator(cosmux)
	node, err := node.NewNode(
		cometCfg,
		pv,
		nodeKey,
		clientCreator,
		node.DefaultGenesisDocProviderFunc(cometCfg),
		cfg.DefaultDBProvider,
		node.DefaultMetricsProvider(cometCfg.Instrumentation),
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
