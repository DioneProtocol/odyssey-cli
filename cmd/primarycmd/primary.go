// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package primarycmd

import (
	"fmt"

	"github.com/DioneProtocol/odyssey-cli/pkg/application"
	"github.com/spf13/cobra"
)

var app *application.Odyssey

// odyssey primary
func NewCmd(injectedApp *application.Odyssey) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "primary",
		Short: "Interact with the Primary Network",
		Long: `The primary command suite provides a collection of tools for interacting with the
Primary Network`,
		Run: func(cmd *cobra.Command, args []string) {
			err := cmd.Help()
			if err != nil {
				fmt.Println(err)
			}
		},
	}
	app = injectedApp
	// primary addValidator
	cmd.AddCommand(newAddValidatorCmd())
	return cmd
}
