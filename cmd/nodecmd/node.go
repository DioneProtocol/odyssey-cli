// Copyright (C) 2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package nodecmd

import (
	"fmt"

	"github.com/DioneProtocol/odyssey-cli/pkg/application"
	"github.com/spf13/cobra"
)

var app *application.Odyssey

// odyssey subnet
func NewCmd(injectedApp *application.Odyssey) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "node",
		Short: "Set up testnet and mainnet validator on cloud service",
		Long: `The node command suite provides a collection of tools for creating and maintaining
validators on Odyssey Network.

To get started, use the node create command wizard to walk through the
configuration to make your node a primary validator on Odyssey public network. You can use the
rest of the commands to maintain your node and make your node a Subnet Validator.`,
		Run: func(cmd *cobra.Command, args []string) {
			err := cmd.Help()
			if err != nil {
				fmt.Println(err)
			}
		},
	}
	app = injectedApp
	// node create
	cmd.AddCommand(newCreateCmd())
	// node validate
	cmd.AddCommand(NewValidateCmd())
	// node sync cluster --subnet subnetName
	cmd.AddCommand(newSyncCmd())
	// node stop
	cmd.AddCommand(newStopCmd())
	// node status cluster
	cmd.AddCommand(newStatusCmd())
	// node list
	cmd.AddCommand(newListCmd())
	// node update
	cmd.AddCommand(newUpdateCmd())
	// node devnet
	cmd.AddCommand(newDevnetCmd())
	// node upgrade
	cmd.AddCommand(newUpgradeCmd())
	// node ssh
	cmd.AddCommand(newSSHCmd())
	// node refresh-ips
	cmd.AddCommand(newRefreshIPsCmd())
	return cmd
}
