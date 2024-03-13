package main

import (
	"fmt"
	"os"

	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"

	"github.com/informalsystems/megablocks/app/sdk-chain-a/app"
	"github.com/informalsystems/megablocks/app/sdk-chain-a/app/params"
	"github.com/informalsystems/megablocks/app/sdk-chain-a/cmd/minid/cmd"
)

func main() {
	params.SetAddressPrefixes()

	rootCmd := cmd.NewRootCmd()
	if err := svrcmd.Execute(rootCmd, "", app.DefaultNodeHome); err != nil {
		fmt.Fprintln(rootCmd.OutOrStderr(), err)
		os.Exit(1)
	}
}
