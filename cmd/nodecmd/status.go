// Copyright (C) 2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package nodecmd

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/DioneProtocol/odyssey-cli/cmd/subnetcmd"
	"github.com/DioneProtocol/odyssey-cli/pkg/ansible"
	"github.com/DioneProtocol/odyssey-cli/pkg/models"
	"github.com/DioneProtocol/odyssey-cli/pkg/ssh"
	"github.com/DioneProtocol/odyssey-cli/pkg/utils"
	"github.com/DioneProtocol/odyssey-cli/pkg/ux"
	"github.com/DioneProtocol/odysseygo/ids"
	"github.com/DioneProtocol/odysseygo/utils/logging"
	"github.com/DioneProtocol/odysseygo/vms/omegavm/status"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"golang.org/x/exp/slices"
)

var subnetName string

func newStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status [clusterName]",
		Short: "(ALPHA Warning) Get node bootstrap status",
		Long: `(ALPHA Warning) This command is currently in experimental mode.

The node status command gets the bootstrap status of all nodes in a cluster with the Primary Network.
If no cluster is given, defaults to node list behaviour.

To get the bootstrap status of a node with a Subnet, use --subnet flag`,
		SilenceUsage: true,
		Args:         cobra.MinimumNArgs(0),
		RunE:         statusNode,
	}
	cmd.Flags().StringVar(&subnetName, "subnet", "", "specify the subnet the node is syncing with")

	return cmd
}

func statusNode(_ *cobra.Command, args []string) error {
	if len(args) == 0 {
		return list(nil, nil)
	}
	clusterName := args[0]
	if err := checkCluster(clusterName); err != nil {
		return err
	}
	ansibleHostIDs, err := ansible.GetAnsibleHostsFromInventory(app.GetAnsibleInventoryDirPath(clusterName))
	if err != nil {
		return err
	}
	hostIDs, err := utils.MapWithError(ansibleHostIDs, func(s string) (string, error) { _, o, err := models.HostAnsibleIDToCloudID(s); return o, err })
	if err != nil {
		return err
	}
	nodeIDs, err := utils.MapWithError(hostIDs, func(s string) (string, error) {
		n, err := getNodeID(app.GetNodeInstanceDirPath(s))
		return n.String(), err
	})
	if err != nil {
		return err
	}
	if subnetName != "" {
		// check subnet first
		if _, err := subnetcmd.ValidateSubnetNameAndGetChains([]string{subnetName}); err != nil {
			return err
		}
	}

	hosts, err := ansible.GetInventoryFromAnsibleInventoryFile(app.GetAnsibleInventoryDirPath(clusterName))
	if err != nil {
		return err
	}
	defer disconnectHosts(hosts)

	notBootstrappedNodes, err := checkHostsAreBootstrapped(hosts)
	if err != nil {
		return err
	}

	notHealthyNodes, err := checkHostsAreHealthy(hosts)
	if err != nil {
		return err
	}

	ux.Logger.PrintToUser("Getting odysseygo version of node(s)")

	wg := sync.WaitGroup{}
	wgResults := models.NodeResults{}
	for _, host := range hosts {
		wg.Add(1)
		go func(nodeResults *models.NodeResults, host *models.Host) {
			defer wg.Done()
			if resp, err := ssh.RunSSHCheckOdysseyGoVersion(host); err != nil {
				nodeResults.AddResult(host.NodeID, nil, err)
				return
			} else {
				if odysseyGoVersion, err := parseOdysseyGoOutput(resp); err != nil {
					nodeResults.AddResult(host.NodeID, nil, err)
				} else {
					nodeResults.AddResult(host.NodeID, odysseyGoVersion, err)
				}
			}
		}(&wgResults, host)
	}
	wg.Wait()
	if wgResults.HasErrors() {
		return fmt.Errorf("failed to get odysseygo version for node(s) %s", wgResults.GetErrorHostMap())
	}
	odysseygoVersionForNode := map[string]string{}
	for nodeID, odysseygoVersion := range wgResults.GetResultMap() {
		odysseygoVersionForNode[nodeID] = fmt.Sprintf("%v", odysseygoVersion)
	}

	notSyncedNodes := []string{}
	subnetSyncedNodes := []string{}
	subnetValidatingNodes := []string{}
	if subnetName != "" {
		clustersConfig, err := app.LoadClustersConfig()
		if err != nil {
			return err
		}
		network := clustersConfig.Clusters[clusterName].Network
		sc, err := app.LoadSidecar(subnetName)
		if err != nil {
			return err
		}
		blockchainID := sc.Networks[network.Name()].BlockchainID
		if blockchainID == ids.Empty {
			return ErrNoBlockchainID
		}
		hostsToCheckSyncStatus := []string{}
		for _, ansibleHostID := range ansibleHostIDs {
			if slices.Contains(notBootstrappedNodes, ansibleHostID) {
				notSyncedNodes = append(notSyncedNodes, ansibleHostID)
			} else {
				hostsToCheckSyncStatus = append(hostsToCheckSyncStatus, ansibleHostID)
			}
		}
		if len(hostsToCheckSyncStatus) != 0 {
			ux.Logger.PrintToUser("Getting subnet sync status of node(s)")
			hostsToCheck := utils.Filter(hosts, func(h *models.Host) bool { return slices.Contains(hostsToCheckSyncStatus, h.NodeID) })
			wg := sync.WaitGroup{}
			wgResults := models.NodeResults{}
			for _, host := range hostsToCheck {
				wg.Add(1)
				go func(nodeResults *models.NodeResults, host *models.Host) {
					defer wg.Done()
					if syncstatus, err := ssh.RunSSHSubnetSyncStatus(host, blockchainID.String()); err != nil {
						nodeResults.AddResult(host.NodeID, nil, err)
						return
					} else {
						if subnetSyncStatus, err := parseSubnetSyncOutput(syncstatus); err != nil {
							nodeResults.AddResult(host.NodeID, nil, err)
							return
						} else {
							nodeResults.AddResult(host.NodeID, subnetSyncStatus, err)
						}
					}
				}(&wgResults, host)
			}
			wg.Wait()
			if wgResults.HasErrors() {
				return fmt.Errorf("failed to check sync status for node(s) %s", wgResults.GetErrorHostMap())
			}
			for nodeID, subnetSyncStatus := range wgResults.GetResultMap() {
				switch subnetSyncStatus {
				case status.Syncing.String():
					subnetSyncedNodes = append(subnetSyncedNodes, nodeID)
				case status.Validating.String():
					subnetValidatingNodes = append(subnetValidatingNodes, nodeID)
				default:
					notSyncedNodes = append(notSyncedNodes, nodeID)
				}
			}
		}
	}
	clustersConfig, err := app.LoadClustersConfig()
	if err != nil {
		return err
	}
	ansibleHosts, err := ansible.GetHostMapfromAnsibleInventory(app.GetAnsibleInventoryDirPath(clusterName))
	if err != nil {
		return err
	}
	printOutput(
		clustersConfig,
		hostIDs,
		ansibleHostIDs,
		ansibleHosts,
		nodeIDs,
		odysseygoVersionForNode,
		notHealthyNodes,
		notBootstrappedNodes,
		notSyncedNodes,
		subnetSyncedNodes,
		subnetValidatingNodes,
		clusterName,
		subnetName,
	)
	return nil
}

func printOutput(
	clustersConfig models.ClustersConfig,
	hostIDs []string,
	ansibleHostIDs []string,
	ansibleHosts map[string]*models.Host,
	nodeIDs []string,
	odygoVersions map[string]string,
	notHealthyHosts []string,
	notBootstrappedHosts []string,
	notSyncedHosts []string,
	subnetSyncedHosts []string,
	subnetValidatingHosts []string,
	clusterName string,
	subnetName string,
) {
	if subnetName == "" && len(notBootstrappedHosts) == 0 {
		ux.Logger.PrintToUser("All nodes in cluster %s are bootstrapped to Primary Network!", clusterName)
	}
	if subnetName != "" && len(notSyncedHosts) == 0 {
		// all nodes are either synced to or validating subnet
		status := "synced to"
		if len(subnetSyncedHosts) == 0 {
			status = "validators of"
		}
		ux.Logger.PrintToUser("All nodes in cluster %s are %s Subnet %s", clusterName, status, subnetName)
	}
	ux.Logger.PrintToUser("")
	tit := fmt.Sprintf("STATUS FOR CLUSTER: %s", clusterName)
	ux.Logger.PrintToUser(tit)
	ux.Logger.PrintToUser(strings.Repeat("=", len(tit)))
	ux.Logger.PrintToUser("")
	header := []string{"Cloud ID", "Node ID", "IP", "Network", "Odygo Version", "Primary Network", "Healthy"}
	if subnetName != "" {
		header = append(header, "Subnet "+subnetName)
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(header)
	table.SetRowLine(true)
	for i, ansibleHostID := range ansibleHostIDs {
		bootstrappedStatus := logging.Green.Wrap("BOOTSTRAPPED")
		if slices.Contains(notBootstrappedHosts, ansibleHostID) {
			bootstrappedStatus = logging.Red.Wrap("NOT_BOOTSTRAPPED")
		}
		healthyStatus := logging.Green.Wrap("OK")
		if slices.Contains(notHealthyHosts, ansibleHostID) {
			healthyStatus = logging.Red.Wrap("UNHEALTHY")
		}
		row := []string{
			hostIDs[i],
			nodeIDs[i],
			ansibleHosts[ansibleHostID].IP,
			clustersConfig.Clusters[clusterName].Network.Name(),
			odygoVersions[ansibleHostID],
			bootstrappedStatus,
			healthyStatus,
		}
		if subnetName != "" {
			syncedStatus := logging.Red.Wrap("NOT_BOOTSTRAPPED")
			if slices.Contains(subnetSyncedHosts, ansibleHostID) {
				syncedStatus = logging.Green.Wrap("SYNCED")
			}
			if slices.Contains(subnetValidatingHosts, ansibleHostID) {
				syncedStatus = logging.Green.Wrap("VALIDATING")
			}
			row = append(row, syncedStatus)
		}
		table.Append(row)
	}
	table.Render()
}
