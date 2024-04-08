package main

import (
	"os"
	"path/filepath"

	comettypes "github.com/cometbft/cometbft/types"
)

func GetInitialAppState(home string) ([]byte, error) {
	genesisFile := filepath.Join(home, "config", "genesis.json")
	if _, err := os.Stat(genesisFile); err != nil {
		// no genesis file found
		return []byte{}, nil
	}

	genesis, err := comettypes.GenesisDocFromFile(genesisFile)
	if err != nil {
		return nil, err
	}
	return genesis.AppState, nil
}
