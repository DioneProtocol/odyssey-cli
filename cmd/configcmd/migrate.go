// Copyright (C) 2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package configcmd

import (
	"fmt"
	"os"

	"github.com/DioneProtocol/odyssey-cli/pkg/constants"
	"github.com/DioneProtocol/odyssey-cli/pkg/utils"
	"github.com/DioneProtocol/odyssey-cli/pkg/ux"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var MigrateOutput string

// odyssey config metrics migrate
func newMigrateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "migrate",
		Short:        "migrate ~/.odyssey-cli.json and ~/.odyssey-cli/config to new configuration location ~/.odyssey-cli/config.json",
		Long:         `migrate command migrates old ~/.odyssey-cli.json and ~/.odyssey-cli/config to /.odyssey-cli/config.json..`,
		RunE:         migrateConfig,
		SilenceUsage: true,
	}
	return cmd
}

func migrateConfig(_ *cobra.Command, _ []string) error {
	oldConfigFilename := utils.UserHomePath(constants.OldConfigFileName)
	oldMetricsConfigFilename := utils.UserHomePath(constants.OldMetricsConfigFileName)
	configFileName := app.Conf.GetConfigPath()
	if utils.FileExists(configFileName) {
		ux.Logger.PrintToUser("Configuration file %s already exists. Configuration migration is not required.", configFileName)
		return nil
	}
	if !utils.FileExists(oldConfigFilename) && !utils.FileExists(oldMetricsConfigFilename) {
		ux.Logger.PrintToUser("Old configuration file %s or %s not found. Configuration migration is not required.", oldConfigFilename, oldMetricsConfigFilename)
		return nil
	} else {
		// load old config
		if utils.FileExists(oldConfigFilename) {
			viper.SetConfigFile(oldConfigFilename)
			if err := viper.MergeInConfig(); err != nil {
				return err
			}
		}
		if utils.FileExists(oldMetricsConfigFilename) {
			viper.SetConfigFile(oldMetricsConfigFilename)
			if err := viper.MergeInConfig(); err != nil {
				return err
			}
		}
		viper.SetConfigFile(configFileName)
		if err := viper.WriteConfig(); err != nil {
			return err
		}
		ux.Logger.PrintToUser("Configuration migrated to %s", configFileName)
		// remove old configuration file
		if utils.FileExists(oldConfigFilename) {
			if err := os.Remove(oldConfigFilename); err != nil {
				return fmt.Errorf("failed to remove old configuration file %s", oldConfigFilename)
			}
			ux.Logger.PrintToUser("Old configuration file %s removed", oldConfigFilename)
		}
		if utils.FileExists(oldMetricsConfigFilename) {
			if err := os.Remove(oldMetricsConfigFilename); err != nil {
				return fmt.Errorf("failed to remove old configuration file %s", oldMetricsConfigFilename)
			}
			ux.Logger.PrintToUser("Old configuration file %s removed", oldMetricsConfigFilename)
		}
		return nil
	}
}
