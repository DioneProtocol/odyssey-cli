// Copyright (C) 2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package networkcmd

import (
	"os"
	"path/filepath"
	"regexp"

	"github.com/DioneProtocol/odyssey-cli/pkg/elasticsubnet"

	"github.com/DioneProtocol/odyssey-cli/pkg/binutils"
	"github.com/DioneProtocol/odyssey-cli/pkg/constants"
	"github.com/DioneProtocol/odyssey-cli/pkg/models"
	"github.com/DioneProtocol/odyssey-cli/pkg/subnet"
	"github.com/DioneProtocol/odyssey-cli/pkg/ux"
	"github.com/shirou/gopsutil/process"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var hard bool

func newCleanCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clean",
		Short: "Stop the running local network and delete state",
		Long: `The network clean command shuts down your local, multi-node network. All deployed Subnets
shutdown and delete their state. You can restart the network by deploying a new Subnet
configuration.`,
		RunE:         clean,
		Args:         cobra.ExactArgs(0),
		SilenceUsage: true,
	}

	cmd.Flags().BoolVar(
		&hard,
		"hard",
		false,
		"Also clean downloaded odysseygo and plugin binaries",
	)

	return cmd
}

func clean(*cobra.Command, []string) error {
	app.Log.Info("killing gRPC server process...")

	if _, err := subnet.SetDefaultSnapshot(app.GetSnapshotsDir(), true, ""); err != nil {
		app.Log.Warn("failed resetting default snapshot", zap.Error(err))
	}

	if err := binutils.KillgRPCServerProcess(app); err != nil {
		app.Log.Warn("failed killing server process", zap.Error(err))
	} else {
		ux.Logger.PrintToUser("Process terminated.")
	}

	if hard {
		ux.Logger.PrintToUser("hard clean requested via flag, removing all downloaded odysseygo and plugin binaries")
		binDir := filepath.Join(app.GetBaseDir(), constants.OdysseyCliBinDir)
		cleanBins(binDir)
		_ = killAllBackendsByName()
	}

	if err := app.ResetPluginsDir(); err != nil {
		return err
	}

	if err := removeLocalDeployInfoFromSidecars(); err != nil {
		return err
	}
	if err := removeLocalElasticSubnetInfoFromSidecars(); err != nil {
		return err
	}
	return nil
}

func removeLocalDeployInfoFromSidecars() error {
	// Remove all local deployment info from sidecar files
	deployedSubnets, err := subnet.GetLocallyDeployedSubnetsFromFile(app)
	if err != nil {
		return err
	}

	for _, subnet := range deployedSubnets {
		sc, err := app.LoadSidecar(subnet)
		if err != nil {
			return err
		}

		delete(sc.Networks, models.Local.String())
		if err = app.UpdateSidecar(&sc); err != nil {
			return err
		}
	}
	return nil
}

func removeLocalElasticSubnetInfoFromSidecars() error {
	// Remove all local elastic subnet info from sidecar files
	elasticSubnets, err := elasticsubnet.GetLocalElasticSubnetsFromFile(app)
	if err != nil {
		return err
	}

	for _, subnet := range elasticSubnets {
		sc, err := app.LoadSidecar(subnet)
		if err != nil {
			return err
		}

		delete(sc.ElasticSubnet, models.Local.String())
		if err = app.UpdateSidecar(&sc); err != nil {
			return err
		}
		if err = deleteElasticSubnetConfigFile(subnet); err != nil {
			return err
		}
	}
	return nil
}

func deleteElasticSubnetConfigFile(subnetName string) error {
	elasticSubetConfigPath := app.GetElasticSubnetConfigPath(subnetName)
	if err := os.Remove(elasticSubetConfigPath); err != nil {
		return err
	}
	return nil
}

func cleanBins(dir string) {
	if err := os.RemoveAll(dir); err != nil {
		ux.Logger.PrintToUser("Removal failed: %s", err)
	}
	ux.Logger.PrintToUser("All existing binaries removed.")
}

func killAllBackendsByName() error {
	procs, err := process.Processes()
	if err != nil {
		return err
	}
	regex := regexp.MustCompile(".* " + constants.BackendCmd + ".*")
	for _, p := range procs {
		name, err := p.Cmdline()
		if err != nil {
			// ignore errors for processes that just died (macos implementation)
			continue
		}
		if regex.MatchString(name) {
			if err := p.Terminate(); err != nil {
				return err
			}
		}
	}
	return nil
}
