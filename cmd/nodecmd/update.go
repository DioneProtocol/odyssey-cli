// Copyright (C) 2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package nodecmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "(ALPHA Warning) Update odysseygo or VM config for all node in a cluster",
		Long: `(ALPHA Warning) This command is currently in experimental mode.

The node update command suite provides a collection of commands for nodes to update
their odysseygo or VM config.

You can check the status after update by calling odyssey node status`,
		Run: func(cmd *cobra.Command, args []string) {
			err := cmd.Help()
			if err != nil {
				fmt.Println(err)
			}
		},
	}
	// node update subnet
	cmd.AddCommand(newUpdateSubnetCmd())
	return cmd
}
