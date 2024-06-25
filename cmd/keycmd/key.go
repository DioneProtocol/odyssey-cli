// Copyright (C) 2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package keycmd

import (
	"fmt"

	"github.com/DioneProtocol/odyssey-cli/pkg/application"
	"github.com/spf13/cobra"
)

var app *application.Odyssey

func NewCmd(injectedApp *application.Odyssey) *cobra.Command {
	app = injectedApp

	cmd := &cobra.Command{
		Use:   "key",
		Short: "Create and manage testnet signing keys",
		Long: `The key command suite provides a collection of tools for creating and managing
signing keys. You can use these keys to deploy Subnets to the Testnet,
but these keys are NOT suitable to use in production environments. DO NOT use
these keys on Mainnet.

To get started, use the key create command.`,
		Run: func(cmd *cobra.Command, args []string) {
			err := cmd.Help()
			if err != nil {
				fmt.Println(err)
			}
		},
	}

	// odyssey key create
	cmd.AddCommand(newCreateCmd())

	// odyssey key list
	cmd.AddCommand(newListCmd())

	// odyssey key delete
	cmd.AddCommand(newDeleteCmd())

	// odyssey key export
	cmd.AddCommand(newExportCmd())

	// odyssey key transfer
	cmd.AddCommand(newTransferCmd())

	return cmd
}
