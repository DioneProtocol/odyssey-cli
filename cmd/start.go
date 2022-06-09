// Copyright (C) 2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package cmd

import (
	"fmt"
	"strings"

	"github.com/ava-labs/avalanche-cli/pkg/binutils"
	"github.com/ava-labs/avalanche-cli/pkg/constants"
	"github.com/ava-labs/avalanche-cli/pkg/subnet"
	"github.com/ava-labs/avalanche-cli/ux"
	"github.com/ava-labs/avalanche-network-runner/client"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start [snapshotName]",
	Short: "Starts a stopped local network",
	Long: `The network start command starts a local, multi-node Avalanche network
on your machine. If "snapshotName" is provided, that snapshot will be used for starting the network 
if it can be found. Otherwise, the last saved unnamed (default) snapshot will be used. The command may fail if the local network
is already running or if no subnets have been deployed.`,

	RunE:         startNetwork,
	Args:         cobra.MaximumNArgs(1),
	SilenceUsage: true,
}

func startNetwork(cmd *cobra.Command, args []string) error {
	sd := subnet.NewLocalSubnetDeployer(log, baseDir)

	if err := sd.StartServer(); err != nil {
		return err
	}

	avalancheGoBinPath, pluginDir, err := sd.SetupLocalEnv()
	if err != nil {
		return err
	}

	cli, err := binutils.NewGRPCClient()
	if err != nil {
		return err
	}

	var snapshotName, startMsg string
	if len(args) > 0 {
		snapshotName = args[0]
		startMsg = fmt.Sprintf("Starting previously deployed and stopped snapshot %s...", snapshotName)
	} else {
		snapshotName = constants.DefaultSnapshotName
		startMsg = "Starting previously deployed and stopped snapshot"
	}

	ctx := binutils.GetAsyncContext()

	ux.Logger.PrintToUser(startMsg)
	loadSnapshotOpts := []client.OpOption{
		client.WithPluginDir(pluginDir),
		client.WithExecPath(avalancheGoBinPath),
	}
	_, err = cli.LoadSnapshot(
		ctx,
		snapshotName,
		loadSnapshotOpts...,
	)
	if err != nil {
		// TODO: use error type not string comparison
		if !strings.Contains(err.Error(), "already bootstrapped") {
			return fmt.Errorf("failed to start network with the persisted snapshot: %s", err)
		}
		ux.Logger.PrintToUser("Network has already been booted. Wait until healthy...")
	} else {
		ux.Logger.PrintToUser("Booting Network. Wait until healthy...")
	}

	// TODO: this should probably be extracted from the deployer and
	// used as an independent helper
	clusterInfo, err := sd.WaitForHealthy(ctx, cli, healthCheckInterval)
	if err != nil {
		return fmt.Errorf("failed waiting for network to become healthy: %s", err)
	}

	endpoints := []string{}
	for _, nodeInfo := range clusterInfo.NodeInfos {
		for vmID, vmInfo := range clusterInfo.CustomVms {
			endpoints = append(endpoints, fmt.Sprintf("Endpoint at node %s for blockchain %q: %s/ext/bc/%s/rpc", nodeInfo.Name, vmID, nodeInfo.GetUri(), vmInfo.BlockchainId))
		}
	}

	fmt.Println()
	if len(endpoints) == 0 {
		ux.Logger.PrintToUser("Network ready to use. No local network node endpoints")
	} else {
		ux.Logger.PrintToUser("Network ready to use. Local network node endpoints:")
		for _, u := range endpoints {
			ux.Logger.PrintToUser(u)
		}
	}

	return nil
}
