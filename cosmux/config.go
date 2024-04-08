package main

import (
	"fmt"
	"log"

	cfg "github.com/cometbft/cometbft/config"
	"github.com/spf13/viper"
)

type CosmuxConfig struct {
	Apps     []MegaBlockApp `mapstructure:"apps"`
	LogLevel string         `mapstructure:"log_level"`
}

func (cfg *CosmuxConfig) ValidateBasic() error {
	// TODO: add some config validation here
	return nil
}

func DefaultApps() []MegaBlockApp {
	return []MegaBlockApp{
		// KV-Store-Chain
		{
			Address:        "unix:///tmp/kvapp.sock",
			ConnectionType: "socket",
			ChainID:        "KVStore",
			Home:           "/tmp/kvstore",
		},

		// SDK-App-1
		{
			//Address:        "127.0.0.1:26658",
			Address:        "unix:///tmp/mind.sock",
			ConnectionType: "socket",
			ChainID:        "sdk-app-2",
			Home:           "/tmp/sdk-app-2",
		},
	}
}

// DefaultConfig returns a default configuration for a CometBFT node
func DefaultConfig() *CosmuxConfig {
	return &CosmuxConfig{
		LogLevel: "info",
		Apps:     DefaultApps(),
	}
}

func ConfigureCometMultiplexer(cfgFile string) *CosmuxConfig {
	config := DefaultConfig()

	if cfgFile == "" {
		fmt.Println("Using default Cosmux configuration")
		return config
	}

	viper.SetConfigFile(cfgFile)
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

// ConfigureCometBFT creates default config in specified cometBFT home directory
func ConfigureCometBFT(homeDir string) *cfg.Config {
	config := cfg.DefaultConfig()
	config.SetRoot(homeDir)
	viper.SetConfigFile(fmt.Sprintf("%s/%s", homeDir, "config/config.toml"))
	viper.SetConfigType("toml")
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
