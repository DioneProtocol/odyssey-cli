// Copyright (C) 2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package configcmd

import (
	"fmt"

	"github.com/DioneProtocol/odyssey-cli/pkg/application"

	"github.com/spf13/cobra"
)

var app *application.Odyssey

func NewCmd(injectedApp *application.Odyssey) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Modify configuration for Odyssey-CLI",
		Long:  `Customize configuration for Odyssey-CLI`,
		Run: func(cmd *cobra.Command, args []string) {
			err := cmd.Help()
			if err != nil {
				fmt.Println(err)
			}
		},
	}
	app = injectedApp
	// set user metrics collection preferences cmd
	cmd.AddCommand(newMetricsCmd())
	cmd.AddCommand(newMigrateCmd())
	cmd.AddCommand(newSingleNodeCmd())
	cmd.AddCommand(newAutorizeCloudAccessCmd())
	return cmd
}
