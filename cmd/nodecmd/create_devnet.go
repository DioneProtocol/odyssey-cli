// Copyright (C) 2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package nodecmd

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	coreth_params "github.com/DioneProtocol/coreth/params"
	"github.com/DioneProtocol/odyssey-cli/pkg/ansible"
	"github.com/DioneProtocol/odyssey-cli/pkg/constants"
	"github.com/DioneProtocol/odyssey-cli/pkg/key"
	"github.com/DioneProtocol/odyssey-cli/pkg/models"
	"github.com/DioneProtocol/odyssey-cli/pkg/ssh"
	"github.com/DioneProtocol/odyssey-cli/pkg/utils"
	"github.com/DioneProtocol/odyssey-cli/pkg/ux"
	"github.com/DioneProtocol/odysseygo/config"
	"github.com/DioneProtocol/odysseygo/utils/crypto/bls"
	"github.com/DioneProtocol/odysseygo/utils/formatting"
	"github.com/DioneProtocol/odysseygo/vms/omegavm/signer"
)

// difference between unlock schedule locktime and startime in original genesis
const (
	genesisLocktimeStartimeDelta    = 2836800
	hexa0Str                        = "0x0"
	defaultLocalDChainFundedAddress = "8db97C7cEcE249c2b98bDC0226Cc4C2A57BF52FC"
	defaultLocalDChainFundedBalance = "0x295BE96E64066972000000"
	allocationCommonEthAddress      = "0xb3d82b1367d362de99ab59a658165aff520cbd4d"
)

func generateCustomDchainGenesis() ([]byte, error) {
	dChainGenesisMap := map[string]interface{}{}
	dChainGenesisMap["config"] = coreth_params.OdysseyLocalChainConfig
	dChainGenesisMap["nonce"] = hexa0Str
	dChainGenesisMap["timestamp"] = hexa0Str
	dChainGenesisMap["extraData"] = "0x00"
	dChainGenesisMap["gasLimit"] = "0x5f5e100"
	dChainGenesisMap["difficulty"] = hexa0Str
	dChainGenesisMap["mixHash"] = "0x0000000000000000000000000000000000000000000000000000000000000000"
	dChainGenesisMap["coinbase"] = "0x0000000000000000000000000000000000000000"
	dChainGenesisMap["alloc"] = map[string]interface{}{
		defaultLocalDChainFundedAddress: map[string]interface{}{
			"balance": defaultLocalDChainFundedBalance,
		},
	}
	dChainGenesisMap["number"] = hexa0Str
	dChainGenesisMap["gasUsed"] = hexa0Str
	dChainGenesisMap["parentHash"] = "0x0000000000000000000000000000000000000000000000000000000000000000"
	return json.Marshal(dChainGenesisMap)
}

func generateCustomGenesis(
	networkID uint32,
	walletAddr string,
	stakingAddr string,
	hosts []*models.Host,
) ([]byte, error) {
	genesisMap := map[string]interface{}{}

	// dchain
	dChainGenesisBytes, err := generateCustomDchainGenesis()
	if err != nil {
		return nil, err
	}
	genesisMap["dChainGenesis"] = string(dChainGenesisBytes)

	// ochain genesis
	genesisMap["networkID"] = networkID
	startTime := time.Now().Unix()
	genesisMap["startTime"] = startTime
	initialStakers := []map[string]interface{}{}
	for _, host := range hosts {
		nodeDirPath := app.GetNodeInstanceDirPath(host.GetCloudID())
		blsPath := filepath.Join(nodeDirPath, constants.BLSKeyFileName)
		blsKey, err := os.ReadFile(blsPath)
		if err != nil {
			return nil, err
		}
		blsSk, err := bls.SecretKeyFromBytes(blsKey)
		if err != nil {
			return nil, err
		}
		p := signer.NewProofOfPossession(blsSk)
		pk, err := formatting.Encode(formatting.HexNC, p.PublicKey[:])
		if err != nil {
			return nil, err
		}
		pop, err := formatting.Encode(formatting.HexNC, p.ProofOfPossession[:])
		if err != nil {
			return nil, err
		}
		nodeID, err := getNodeID(nodeDirPath)
		if err != nil {
			return nil, err
		}
		initialStaker := map[string]interface{}{
			"nodeID":        nodeID,
			"rewardAddress": walletAddr,
			"delegationFee": 1000000,
			"signer": map[string]interface{}{
				"proofOfPossession": pop,
				"publicKey":         pk,
			},
		}
		initialStakers = append(initialStakers, initialStaker)
	}
	genesisMap["initialStakeDuration"] = 31536000
	genesisMap["initialStakeDurationOffset"] = 5400
	genesisMap["initialStakers"] = initialStakers
	lockTime := startTime + genesisLocktimeStartimeDelta
	allocations := []interface{}{}
	alloc := map[string]interface{}{
		"dioneAddr":     walletAddr,
		"ethAddr":       allocationCommonEthAddress,
		"initialAmount": 300000000000000000,
		"unlockSchedule": []interface{}{
			map[string]interface{}{"amount": 20000000000000000},
			map[string]interface{}{"amount": 10000000000000000, "locktime": lockTime},
		},
	}
	allocations = append(allocations, alloc)
	alloc = map[string]interface{}{
		"dioneAddr":     stakingAddr,
		"ethAddr":       allocationCommonEthAddress,
		"initialAmount": 0,
		"unlockSchedule": []interface{}{
			map[string]interface{}{"amount": 10000000000000000, "locktime": lockTime},
		},
	}
	allocations = append(allocations, alloc)
	genesisMap["allocations"] = allocations
	genesisMap["initialStakedFunds"] = []interface{}{
		stakingAddr,
	}
	genesisMap["message"] = "{{ fun_quote }}"

	return json.MarshalIndent(genesisMap, "", " ")
}

func setupDevnet(clusterName string, hosts []*models.Host) error {
	if err := checkCluster(clusterName); err != nil {
		return err
	}
	inventoryPath := app.GetAnsibleInventoryDirPath(clusterName)
	ansibleHostIDs, err := ansible.GetAnsibleHostsFromInventory(inventoryPath)
	if err != nil {
		return err
	}
	ansibleHosts, err := ansible.GetHostMapfromAnsibleInventory(app.GetAnsibleInventoryDirPath(clusterName))
	if err != nil {
		return err
	}
	cloudHostIDs, err := utils.MapWithError(ansibleHostIDs, func(s string) (string, error) { _, o, err := models.HostAnsibleIDToCloudID(s); return o, err })
	if err != nil {
		return err
	}
	nodeIDs, err := utils.MapWithError(cloudHostIDs, func(s string) (string, error) {
		n, err := getNodeID(app.GetNodeInstanceDirPath(s))
		return n.String(), err
	})
	if err != nil {
		return err
	}

	// set devnet network
	network := models.NewDevnetNetwork(ansibleHosts[ansibleHostIDs[0]].IP, 9650)
	ux.Logger.PrintToUser("Devnet Network Id: %d", network.ID)
	ux.Logger.PrintToUser("Devnet Endpoint: %s", network.Endpoint)

	// get random staking key for devnet genesis
	k, err := key.NewSoft(network.ID)
	if err != nil {
		return err
	}
	stakingAddrStr := k.A()[0]

	// get ewoq key as funded key for devnet genesis
	k, err = key.LoadEwoq(network.ID)
	if err != nil {
		return err
	}
	walletAddrStr := k.A()[0]

	// create genesis file at each node dir
	genesisBytes, err := generateCustomGenesis(network.ID, walletAddrStr, stakingAddrStr, hosts)
	if err != nil {
		return err
	}
	for _, cloudHostID := range cloudHostIDs {
		outFile := filepath.Join(app.GetNodeInstanceDirPath(cloudHostID), "genesis.json")
		if err := os.WriteFile(outFile, genesisBytes, constants.WriteReadReadPerms); err != nil {
			return err
		}
	}

	// create odysseygo conf node.json at each node dir
	bootstrapIPs := []string{}
	bootstrapIDs := []string{}
	for i, ansibleHostID := range ansibleHostIDs {
		cloudHostID := cloudHostIDs[i]
		confMap := map[string]interface{}{}
		confMap[config.HTTPHostKey] = ""
		confMap[config.PublicIPKey] = ansibleHosts[ansibleHostID].IP
		confMap[config.NetworkNameKey] = fmt.Sprintf("network-%d", network.ID)
		confMap[config.BootstrapIDsKey] = strings.Join(bootstrapIDs, ",")
		confMap[config.BootstrapIPsKey] = strings.Join(bootstrapIPs, ",")
		confMap[config.GenesisFileKey] = "/home/ubuntu/.odysseygo/configs/genesis.json"
		bootstrapIDs = append(bootstrapIDs, nodeIDs[i])
		bootstrapIPs = append(bootstrapIPs, ansibleHosts[ansibleHostID].IP+":9651")
		confBytes, err := json.MarshalIndent(confMap, "", " ")
		if err != nil {
			return err
		}
		outFile := filepath.Join(app.GetNodeInstanceDirPath(cloudHostID), "node.json")
		if err := os.WriteFile(outFile, confBytes, constants.WriteReadReadPerms); err != nil {
			return err
		}
	}
	// update node/s genesis + conf and start
	wg := sync.WaitGroup{}
	wgResults := models.NodeResults{}
	for _, host := range hosts {
		wg.Add(1)
		go func(nodeResults *models.NodeResults, host *models.Host) {
			defer wg.Done()
			keyPath := filepath.Join(app.GetNodesDir(), host.GetCloudID())
			if err := ssh.RunSSHSetupDevNet(host, keyPath); err != nil {
				nodeResults.AddResult(host.NodeID, nil, err)
				return
			}
		}(&wgResults, host)
	}
	wg.Wait()
	for _, node := range hosts {
		if wgResults.HasNodeIDWithError(node.NodeID) {
			ux.Logger.PrintToUser("Node %s is ERROR with error: %s", node.NodeID, wgResults.GetErrorHostMap()[node.NodeID])
		} else {
			ux.Logger.PrintToUser("Node %s is SETUP as devnet", node.NodeID)
		}
	}
	// stop execution if at least one node failed
	if wgResults.HasErrors() {
		return fmt.Errorf("failed to deploy node(s) %s", wgResults.GetErrorHostMap())
	}

	// update cluster config with network information
	clustersConfig, err := app.LoadClustersConfig()
	if err != nil {
		return err
	}
	clusterConfig := clustersConfig.Clusters[clusterName]
	clustersConfig.Clusters[clusterName] = models.ClusterConfig{
		Network: network,
		Nodes:   clusterConfig.Nodes,
	}
	return app.WriteClustersConfigFile(&clustersConfig)
}
