// Copyright (C) 2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package nodecmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/DioneProtocol/odyssey-cli/pkg/ansible"
	"github.com/DioneProtocol/odyssey-cli/pkg/binutils"
	"github.com/DioneProtocol/odyssey-cli/pkg/constants"
	"github.com/DioneProtocol/odyssey-cli/pkg/models"
	"github.com/DioneProtocol/odyssey-cli/pkg/ssh"
	"github.com/DioneProtocol/odyssey-cli/pkg/utils"
	"github.com/DioneProtocol/odyssey-cli/pkg/ux"
	"github.com/DioneProtocol/odyssey-cli/pkg/vm"
	"github.com/spf13/cobra"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

type nodeUpgradeInfo struct {
	OdysseyGoVersion      string   // odyssey go version to update to on cloud server
	SubnetEVMVersion      string   // subnet EVM version to update to on cloud server
	SubnetEVMIDsToUpgrade []string // list of ID of Subnet EVM to be upgraded to subnet EVM version to update to
}

func newUpgradeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "(ALPHA Warning) Update odysseygo or VM version for all node in a cluster",
		Long: `(ALPHA Warning) This command is currently in experimental mode.

The node update command suite provides a collection of commands for nodes to update
their odysseygo or VM version.

You can check the status after upgrade by calling odyssey node status`,
		SilenceUsage: true,
		Args:         cobra.ExactArgs(1),
		RunE:         upgrade,
	}

	return cmd
}

func upgrade(_ *cobra.Command, args []string) error {
	clusterName := args[0]
	if err := checkCluster(clusterName); err != nil {
		return err
	}
	hosts, err := ansible.GetInventoryFromAnsibleInventoryFile(app.GetAnsibleInventoryDirPath(clusterName))
	if err != nil {
		return err
	}
	defer disconnectHosts(hosts)
	toUpgradeNodesMap, err := getNodesUpgradeInfo(hosts)
	if err != nil {
		return err
	}
	for host, upgradeInfo := range toUpgradeNodesMap {
		if upgradeInfo.OdysseyGoVersion != "" {
			if err := upgradeOdysseyGo(host, upgradeInfo.OdysseyGoVersion); err != nil {
				return err
			}
		}
		if upgradeInfo.SubnetEVMVersion != "" {
			subnetEVMVersionToUpgradeToWoPrefix := strings.TrimPrefix(upgradeInfo.SubnetEVMVersion, "v")
			subnetEVMArchive := fmt.Sprintf(constants.SubnetEVMArchive, subnetEVMVersionToUpgradeToWoPrefix)
			subnetEVMReleaseURL := fmt.Sprintf(constants.SubnetEVMReleaseURL, upgradeInfo.SubnetEVMVersion, subnetEVMArchive)
			if err := getNewSubnetEVMRelease(host, subnetEVMReleaseURL, subnetEVMArchive, upgradeInfo.SubnetEVMVersion); err != nil {
				return err
			}
			if err := ssh.RunSSHStopNode(host); err != nil {
				return err
			}
			for _, vmID := range upgradeInfo.SubnetEVMIDsToUpgrade {
				subnetEVMBinaryPath := fmt.Sprintf(constants.CloudNodeSubnetEvmBinaryPath, vmID)
				if err := upgradeSubnetEVM(host, subnetEVMBinaryPath, upgradeInfo.SubnetEVMVersion); err != nil {
					return err
				}
			}
			if err := ssh.RunSSHStartNode(host); err != nil {
				return err
			}
		}
	}
	return nil
}

// getNodesUpgradeInfo gets the node versions of all given nodes and checks which
// nodes needs to have Odyssey Go & SubnetEVM upgraded. It first checks the subnet EVM version -
// it will install the newest subnet EVM version and install the latest odyssey Go that is still compatible with the Subnet EVM version
// if the node is not tracking any subnet, it will just install latestOdygoVersion
func getNodesUpgradeInfo(hosts []*models.Host) (map[*models.Host]nodeUpgradeInfo, error) {
	latestOdygoVersion, err := app.Downloader.GetLatestReleaseVersion(binutils.GetGithubLatestReleaseURL(
		constants.DioneProtocolOrg,
		constants.OdysseyGoRepoName,
	))
	if err != nil {
		return nil, err
	}
	latestSubnetEVMVersion, err := app.Downloader.GetLatestReleaseVersion(binutils.GetGithubLatestReleaseURL(
		constants.DioneProtocolOrg,
		constants.SubnetEVMRepoName,
	))
	if err != nil {
		return nil, err
	}
	rpcVersion, err := vm.GetRPCProtocolVersion(app, models.SubnetEvm, latestSubnetEVMVersion)
	if err != nil {
		return nil, err
	}
	nodeErrors := map[string]error{}
	nodesToUpgrade := make(map[*models.Host]nodeUpgradeInfo)

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
				if vmVersions, err := parseNodeVersionOutput(resp); err != nil {
					nodeResults.AddResult(host.NodeID, nil, err)
				} else {
					nodeResults.AddResult(host.NodeID, vmVersions, err)
				}
			}
		}(&wgResults, host)
	}
	wg.Wait()
	if wgResults.HasErrors() {
		return nil, fmt.Errorf("failed to get odysseygo version for node(s) %s", wgResults.GetErrorHostMap())
	}

	nodeIDToHost := map[string]*models.Host{}
	for _, host := range hosts {
		nodeIDToHost[host.NodeID] = host
	}

	for hostID, vmVersionsInterface := range wgResults.GetResultMap() {
		vmVersions, err := utils.ConvertInterfaceToMap(vmVersionsInterface)
		if err != nil {
			return nil, err
		}
		currentOdysseyGoVersion := vmVersions[constants.PlatformKeyName]
		odysseyGoVersionToUpdateTo := latestOdygoVersion
		nodeUpgradeInfo := nodeUpgradeInfo{}
		nodeUpgradeInfo.SubnetEVMIDsToUpgrade = []string{}
		for vmName, vmVersion := range vmVersions {
			// when calling info.getNodeVersion, this is what we get
			// "vmVersions":{"alpha":"v1.10.12","evm":"v0.12.5","n8Anw9kErmgk7KHviddYtecCmziLZTphDwfL1V2DfnFjWZXbE":"v0.5.6","omega":"v1.10.12"}},
			// we need to get the VM ID of the subnets that the node is currently validating, in the example above it is n8Anw9kErmgk7KHviddYtecCmziLZTphDwfL1V2DfnFjWZXbE
			if !checkIfKeyIsStandardVMName(vmName) {
				if vmVersion != latestSubnetEVMVersion {
					// update subnet EVM version
					ux.Logger.PrintToUser("Upgrading Subnet EVM version for node %s from version %s to version %s", hostID, vmVersion, latestSubnetEVMVersion)
					nodeUpgradeInfo.SubnetEVMVersion = latestSubnetEVMVersion
					nodeUpgradeInfo.SubnetEVMIDsToUpgrade = append(nodeUpgradeInfo.SubnetEVMIDsToUpgrade, vmName)
				}
				// find the highest version of odyssey go that is still compatible with current highest rpc
				odysseyGoVersionToUpdateTo, err = GetLatestOdygoVersionForRPC(rpcVersion)
				if err != nil {
					nodeErrors[hostID] = err
					continue
				}
			}
		}
		if _, hasFailed := nodeErrors[hostID]; hasFailed {
			continue
		}
		if currentOdysseyGoVersion != odysseyGoVersionToUpdateTo {
			ux.Logger.PrintToUser("Upgrading Odyssey Go version for node %s from version %s to version %s", hostID, currentOdysseyGoVersion, odysseyGoVersionToUpdateTo)
			nodeUpgradeInfo.OdysseyGoVersion = odysseyGoVersionToUpdateTo
		}
		nodesToUpgrade[nodeIDToHost[hostID]] = nodeUpgradeInfo
	}
	if len(nodeErrors) > 0 {
		ux.Logger.PrintToUser("Failed to upgrade nodes: ")
		for node, nodeErr := range nodeErrors {
			ux.Logger.PrintToUser("node %s failed to upgrade due to %s", node, nodeErr)
		}
		return nil, fmt.Errorf("failed to upgrade node(s) %s", maps.Keys(nodeErrors))
	}
	return nodesToUpgrade, nil
}

// checks if vmName is "alpha", "evm" or "omega"
func checkIfKeyIsStandardVMName(vmName string) bool {
	standardVMNames := []string{constants.PlatformKeyName, constants.EVMKeyName, constants.AVMKeyName}
	return slices.Contains(standardVMNames, vmName)
}

func upgradeOdysseyGo(
	host *models.Host,
	odyGoVersionToUpdateTo string,
) error {
	ux.Logger.PrintToUser("Upgrading Odyssey Go version of node %s to version %s ...", host.NodeID, odyGoVersionToUpdateTo)
	if err := ssh.RunSSHUpgradeOdysseygo(host, odyGoVersionToUpdateTo); err != nil {
		return err
	}
	ux.Logger.PrintToUser("Successfully upgraded Odyssey Go version of node %s!", host.NodeID)
	ux.Logger.PrintToUser("======================================")
	return nil
}

func upgradeSubnetEVM(
	host *models.Host,
	subnetEVMBinaryPath string,
	subnetEVMVersion string,
) error {
	ux.Logger.PrintToUser("Upgrading SubnetEVM version of node %s to version %s ...", host.NodeID, subnetEVMVersion)
	if err := ssh.RunSSHUpgradeSubnetEVM(host, subnetEVMBinaryPath); err != nil {
		return err
	}
	ux.Logger.PrintToUser("Successfully upgraded SubnetEVM version of node %s!", host.NodeID)
	ux.Logger.PrintToUser("======================================")
	return nil
}

func getNewSubnetEVMRelease(
	host *models.Host,
	subnetEVMReleaseURL string,
	subnetEVMArchive string,
	subnetEVMVersion string,
) error {
	ux.Logger.PrintToUser("Getting new SubnetEVM version %s ...", subnetEVMVersion)
	if err := ssh.RunSSHGetNewSubnetEVMRelease(host, subnetEVMReleaseURL, subnetEVMArchive); err != nil {
		return err
	}
	ux.Logger.PrintToUser("Successfully downloaded SubnetEVM version for node %s!", host.NodeID)
	ux.Logger.PrintToUser("======================================")
	return nil
}

func parseNodeVersionOutput(byteValue []byte) (map[string]interface{}, error) {
	var result map[string]interface{}
	if err := json.Unmarshal(byteValue, &result); err != nil {
		return nil, err
	}
	nodeIDInterface, ok := result["result"].(map[string]interface{})
	if ok {
		vmVersions, ok := nodeIDInterface["vmVersions"].(map[string]interface{})
		if ok {
			return vmVersions, nil
		}
	}
	return nil, nil
}
