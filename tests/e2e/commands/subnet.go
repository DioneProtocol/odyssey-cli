// Copyright (C) 2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package commands

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/DioneProtocol/odyssey-cli/pkg/constants"
	"github.com/DioneProtocol/odyssey-cli/pkg/models"
	"github.com/DioneProtocol/odyssey-cli/tests/e2e/utils"
	"github.com/onsi/gomega"
)

const subnetEVMMainnetChainID = 11

/* #nosec G204 */
func CreateSubnetEvmConfig(subnetName string, genesisPath string) (string, string) {
	mapper := utils.NewVersionMapper()
	mapping, err := utils.GetVersionMapping(mapper)
	gomega.Expect(err).Should(gomega.BeNil())
	// let's use a SubnetEVM version which has a guaranteed compatible odygo
	CreateSubnetEvmConfigWithVersion(subnetName, genesisPath, mapping[utils.LatestEVM2OdygoKey])
	return mapping[utils.LatestEVM2OdygoKey], mapping[utils.LatestOdygo2EVMKey]
}

/* #nosec G204 */
func CreateSubnetEvmConfigWithVersion(subnetName string, genesisPath string, version string) {
	// Check config does not already exist
	exists, err := utils.SubnetConfigExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeFalse())

	// Create config
	cmdArgs := []string{SubnetCmd, "create", "--genesis", genesisPath, "--evm", subnetName, "--" + constants.SkipUpdateFlag}
	if version == "" {
		cmdArgs = append(cmdArgs, "--latest")
	} else {
		cmdArgs = append(cmdArgs, "--vm-version", version)
	}
	cmd := exec.Command(CLIBinary, cmdArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(cmd.String())
		fmt.Println(string(output))
		utils.PrintStdErr(err)
	}
	gomega.Expect(err).Should(gomega.BeNil())

	// Config should now exist
	exists, err = utils.SubnetConfigExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())
}

/* #nosec G204 */
func ConfigureChainConfig(subnetName string, genesisPath string) {
	// run configure
	cmdArgs := []string{SubnetCmd, "configure", subnetName, "--chain-config", genesisPath, "--" + constants.SkipUpdateFlag}
	cmd := exec.Command(CLIBinary, cmdArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(string(output))
		fmt.Println(err)
	}
	gomega.Expect(err).Should(gomega.BeNil())

	// Config should now exist
	exists, err := utils.ChainConfigExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())
}

/* #nosec G204 */
func ConfigurePerNodeChainConfig(subnetName string, perNodeChainConfigPath string) {
	// run configure
	cmdArgs := []string{SubnetCmd, "configure", subnetName, "--per-node-chain-config", perNodeChainConfigPath, "--" + constants.SkipUpdateFlag}
	cmd := exec.Command(CLIBinary, cmdArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(string(output))
		fmt.Println(err)
	}
	gomega.Expect(err).Should(gomega.BeNil())

	// Config should now exist
	exists, err := utils.PerNodeChainConfigExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())
}

/* #nosec G204 */
func CreateCustomVMConfig(subnetName string, genesisPath string, vmPath string) {
	// Check config does not already exist
	exists, err := utils.SubnetConfigExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeFalse())
	// Check vm binary does not already exist
	exists, err = utils.SubnetCustomVMExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeFalse())

	// Create config
	cmd := exec.Command(
		CLIBinary,
		SubnetCmd,
		"create",
		"--genesis",
		genesisPath,
		"--custom",
		subnetName,
		"--custom-vm-path",
		vmPath,
		"--"+constants.SkipUpdateFlag,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		var (
			exitErr *exec.ExitError
			stderr  string
		)
		if errors.As(err, &exitErr) {
			stderr = string(exitErr.Stderr)
		}
		fmt.Println(string(output))
		utils.PrintStdErr(err)
		fmt.Println(stderr)
		gomega.Expect(err).Should(gomega.BeNil())
	}

	// Config should now exist
	exists, err = utils.SubnetConfigExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())
	exists, err = utils.SubnetCustomVMExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())
}

/* #nosec G204 */
func DeleteSubnetConfig(subnetName string) {
	// Config should exist
	exists, err := utils.SubnetConfigExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())

	// Now delete config
	cmd := exec.Command(CLIBinary, SubnetCmd, "delete", subnetName, "--"+constants.SkipUpdateFlag)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(cmd.String())
		fmt.Println(string(output))
		utils.PrintStdErr(err)
	}
	gomega.Expect(err).Should(gomega.BeNil())

	// Config should no longer exist
	exists, err = utils.SubnetConfigExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeFalse())
}

func DeleteElasticSubnetConfig(subnetName string) {
	var err error
	elasticSubnetConfig := filepath.Join(utils.GetBaseDir(), constants.SubnetDir, subnetName, constants.ElasticSubnetConfigFileName)
	if _, err = os.Stat(elasticSubnetConfig); errors.Is(err, os.ErrNotExist) {
		// does *not* exist
		err = nil
	} else {
		err = os.Remove(elasticSubnetConfig)
	}
	gomega.Expect(err).Should(gomega.BeNil())
}

// Returns the deploy output
/* #nosec G204 */
func DeploySubnetLocally(subnetName string) string {
	return DeploySubnetLocallyWithArgs(subnetName, "", "")
}

/* #nosec G204 */
func DeploySubnetLocallyExpectError(subnetName string) {
	mapper := utils.NewVersionMapper()
	mapping, err := utils.GetVersionMapping(mapper)
	gomega.Expect(err).Should(gomega.BeNil())

	DeploySubnetLocallyWithArgsExpectError(subnetName, mapping[utils.OnlyOdygoKey], "")
}

// Returns the deploy output
/* #nosec G204 */
func DeploySubnetLocallyWithViperConf(subnetName string, confPath string) string {
	mapper := utils.NewVersionMapper()
	mapping, err := utils.GetVersionMapping(mapper)
	gomega.Expect(err).Should(gomega.BeNil())

	return DeploySubnetLocallyWithArgs(subnetName, mapping[utils.OnlyOdygoKey], confPath)
}

// Returns the deploy output
/* #nosec G204 */
func DeploySubnetLocallyWithVersion(subnetName string, version string) string {
	return DeploySubnetLocallyWithArgs(subnetName, version, "")
}

// Returns the deploy output
/* #nosec G204 */
func DeploySubnetLocallyWithArgs(subnetName string, version string, confPath string) string {
	// Check config exists
	exists, err := utils.SubnetConfigExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())

	// Deploy subnet locally
	cmdArgs := []string{SubnetCmd, "deploy", "--local", subnetName, "--" + constants.SkipUpdateFlag}
	if version != "" {
		cmdArgs = append(cmdArgs, "--odysseygo-version", version)
	}
	if confPath != "" {
		cmdArgs = append(cmdArgs, "--config", confPath)
	}
	cmd := exec.Command(CLIBinary, cmdArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		var (
			exitErr *exec.ExitError
			stderr  string
		)
		if errors.As(err, &exitErr) {
			stderr = string(exitErr.Stderr)
		}
		fmt.Println(string(output))
		utils.PrintStdErr(err)
		fmt.Println(stderr)
	}
	gomega.Expect(err).Should(gomega.BeNil())

	return string(output)
}

func DeploySubnetLocallyWithArgsAndOutput(subnetName string, version string, confPath string) ([]byte, error) {
	// Check config exists
	exists, err := utils.SubnetConfigExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())

	// Deploy subnet locally
	cmdArgs := []string{SubnetCmd, "deploy", "--local", subnetName, "--" + constants.SkipUpdateFlag}
	if version != "" {
		cmdArgs = append(cmdArgs, "--odysseygo-version", version)
	}
	if confPath != "" {
		cmdArgs = append(cmdArgs, "--config", confPath)
	}
	cmd := exec.Command(CLIBinary, cmdArgs...)
	return cmd.CombinedOutput()
}

/* #nosec G204 */
func DeploySubnetLocallyWithArgsExpectError(subnetName string, version string, confPath string) {
	_, err := DeploySubnetLocallyWithArgsAndOutput(subnetName, version, confPath)
	gomega.Expect(err).Should(gomega.HaveOccurred())
}

// simulates testnet deploy execution path on a local network
/* #nosec G204 */
func SimulateTestnetDeploy(
	subnetName string,
	key string,
	controlKeys string,
) string {
	// Check config exists
	exists, err := utils.SubnetConfigExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())

	// enable simulation of public network execution paths on a local network
	err = os.Setenv(constants.SimulatePublicNetwork, "true")
	gomega.Expect(err).Should(gomega.BeNil())

	// Deploy subnet locally
	cmd := exec.Command(
		CLIBinary,
		SubnetCmd,
		"deploy",
		"--testnet",
		"--threshold",
		"1",
		"--key",
		key,
		"--control-keys",
		controlKeys,
		subnetName,
		"--"+constants.SkipUpdateFlag,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(cmd.String())
		fmt.Println(string(output))
		utils.PrintStdErr(err)
	}
	gomega.Expect(err).Should(gomega.BeNil())

	// disable simulation of public network execution paths on a local network
	err = os.Unsetenv(constants.SimulatePublicNetwork)
	gomega.Expect(err).Should(gomega.BeNil())

	return string(output)
}

// simulates mainnet deploy execution path on a local network
/* #nosec G204 */
func SimulateMainnetDeploy(
	subnetName string,
	mainnetChainID int,
	errorIsExpected bool,
) string {
	// Check config exists
	exists, err := utils.SubnetConfigExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())

	// enable simulation of public network execution paths on a local network
	err = os.Setenv(constants.SimulatePublicNetwork, "true")
	gomega.Expect(err).Should(gomega.BeNil())

	if mainnetChainID == 0 {
		mainnetChainID = subnetEVMMainnetChainID
	}

	// Deploy subnet locally
	return utils.ExecCommand(
		CLIBinary,
		[]string{
			SubnetCmd,
			"deploy",
			"--mainnet",
			"--threshold",
			"1",
			"--same-control-key",
			"--mainnet-chain-id",
			fmt.Sprint(mainnetChainID),
			subnetName,
			"--" + constants.SkipUpdateFlag,
		},
		true,
		errorIsExpected,
	)
}

// simulates multisig mainnet deploy execution path on a local network
/* #nosec G204 */
func SimulateMultisigMainnetDeploy(
	subnetName string,
	subnetControlAddrs []string,
	chainCreationAuthAddrs []string,
	txPath string,
	errorIsExpected bool,
) string {
	// Check config exists
	exists, err := utils.SubnetConfigExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())

	// enable simulation of public network execution paths on a local network
	err = os.Setenv(constants.SimulatePublicNetwork, "true")
	gomega.Expect(err).Should(gomega.BeNil())

	// Multisig deploy for local subnet with possible tx file generation
	return utils.ExecCommand(
		CLIBinary,
		[]string{
			SubnetCmd,
			"deploy",
			"--mainnet",
			"--control-keys",
			strings.Join(subnetControlAddrs, ","),
			"--subnet-auth-keys",
			strings.Join(chainCreationAuthAddrs, ","),
			"--output-tx-path",
			txPath,
			"--mainnet-chain-id",
			fmt.Sprint(subnetEVMMainnetChainID),
			subnetName,
			"--" + constants.SkipUpdateFlag,
		},
		true,
		errorIsExpected,
	)
}

// transaction signing with ledger
/* #nosec G204 */
func TransactionSignWithLedger(
	subnetName string,
	txPath string,
	errorIsExpected bool,
) string {
	// Check config exists
	exists, err := utils.SubnetConfigExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())

	return utils.ExecCommand(
		CLIBinary,
		[]string{
			"transaction",
			"sign",
			subnetName,
			"--input-tx-filepath",
			txPath,
			"--ledger",
			"--" + constants.SkipUpdateFlag,
		},
		true,
		errorIsExpected,
	)
}

// transaction commit
/* #nosec G204 */
func TransactionCommit(
	subnetName string,
	txPath string,
	errorIsExpected bool,
) string {
	// Check config exists
	exists, err := utils.SubnetConfigExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())

	return utils.ExecCommand(
		CLIBinary,
		[]string{
			"transaction",
			"commit",
			subnetName,
			"--input-tx-filepath",
			txPath,
			"--" + constants.SkipUpdateFlag,
		},
		true,
		errorIsExpected,
	)
}

// simulates testnet add validator execution path on a local network
/* #nosec G204 */
func SimulateTestnetAddValidator(
	subnetName string,
	key string,
	nodeID string,
	start string,
	period string,
	weight string,
) string {
	// Check config exists
	exists, err := utils.SubnetConfigExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())

	// enable simulation of public network execution paths on a local network
	err = os.Setenv(constants.SimulatePublicNetwork, "true")
	gomega.Expect(err).Should(gomega.BeNil())

	cmd := exec.Command(
		CLIBinary,
		SubnetCmd,
		"addValidator",
		"--testnet",
		"--key",
		key,
		"--nodeID",
		nodeID,
		"--start-time",
		start,
		"--staking-period",
		period,
		"--weight",
		weight,
		subnetName,
		"--"+constants.SkipUpdateFlag,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(cmd.String())
		fmt.Println(string(output))
		utils.PrintStdErr(err)
	}
	gomega.Expect(err).Should(gomega.BeNil())

	// disable simulation of public network execution paths on a local network
	err = os.Unsetenv(constants.SimulatePublicNetwork)
	gomega.Expect(err).Should(gomega.BeNil())

	return string(output)
}

// simulates testnet add validator execution path on a local network
func SimulateTestnetRemoveValidator(
	subnetName string,
	key string,
	nodeID string,
) string {
	// Check config exists
	exists, err := utils.SubnetConfigExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())

	// enable simulation of public network execution paths on a local network
	err = os.Setenv(constants.SimulatePublicNetwork, "true")
	gomega.Expect(err).Should(gomega.BeNil())

	cmd := exec.Command(
		CLIBinary,
		SubnetCmd,
		"removeValidator",
		"--testnet",
		"--key",
		key,
		"--nodeID",
		nodeID,
		subnetName,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(cmd.String())
		fmt.Println(string(output))
		utils.PrintStdErr(err)
	}
	gomega.Expect(err).Should(gomega.BeNil())

	// disable simulation of public network execution paths on a local network
	err = os.Unsetenv(constants.SimulatePublicNetwork)
	gomega.Expect(err).Should(gomega.BeNil())

	return string(output)
}

func SimulateTestnetTransformSubnet(
	subnetName string,
	key string,
) (string, error) {
	// Check config exists
	exists, err := utils.SubnetConfigExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())

	// enable simulation of public network execution paths on a local network
	err = os.Setenv(constants.SimulatePublicNetwork, "true")
	gomega.Expect(err).Should(gomega.BeNil())
	cmd := exec.Command(
		CLIBinary,
		SubnetCmd,
		ElasticTransformCmd,
		"--testnet",
		"--key",
		key,
		"--tokenName",
		"BLIZZARD",
		"--tokenSymbol",
		"BRRR",
		"--denomination",
		"0",
		"--default",
		"--force",
		subnetName,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(cmd.String())
		fmt.Println(string(output))
		utils.PrintStdErr(err)
		err2 := os.Unsetenv(constants.SimulatePublicNetwork)
		gomega.Expect(err2).Should(gomega.BeNil())
		return "", err
	}

	// disable simulation of public network execution paths on a local network
	err = os.Unsetenv(constants.SimulatePublicNetwork)
	gomega.Expect(err).Should(gomega.BeNil())

	return string(output), nil
}

func SimulateTestnetAddPermissionlessValidator(
	subnetName string,
	key string,
	nodeID string,
	stakeAmount string,
	stakingPeriod string,
) (string, error) {
	// Check config exists
	exists, err := utils.SubnetConfigExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())

	// enable simulation of public network execution paths on a local network
	err = os.Setenv(constants.SimulatePublicNetwork, "true")
	gomega.Expect(err).Should(gomega.BeNil())
	startTimeStr := time.Now().Add(constants.StakingStartLeadTime).UTC().Format(constants.TimeParseLayout)
	cmd := exec.Command(
		CLIBinary,
		SubnetCmd,
		JoinCmd,
		"--testnet",
		"--key",
		key,
		"--elastic",
		"--nodeID",
		nodeID,
		"--stake-amount",
		stakeAmount,
		"--start-time",
		startTimeStr,
		"--staking-period",
		stakingPeriod,
		subnetName,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(cmd.String())
		fmt.Println(string(output))
		utils.PrintStdErr(err)
		err2 := os.Unsetenv(constants.SimulatePublicNetwork)
		gomega.Expect(err2).Should(gomega.BeNil())
		return "", err
	}

	// disable simulation of public network execution paths on a local network
	err = os.Unsetenv(constants.SimulatePublicNetwork)
	gomega.Expect(err).Should(gomega.BeNil())

	return string(output), nil
}

// simulates mainnet add validator execution path on a local network
/* #nosec G204 */
func SimulateMainnetAddValidator(
	subnetName string,
	nodeID string,
	start string,
	period string,
	weight string,
) string {
	// Check config exists
	exists, err := utils.SubnetConfigExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())

	// enable simulation of public network execution paths on a local network
	err = os.Setenv(constants.SimulatePublicNetwork, "true")
	gomega.Expect(err).Should(gomega.BeNil())

	return utils.ExecCommand(
		CLIBinary,
		[]string{
			SubnetCmd,
			"addValidator",
			"--mainnet",
			"--nodeID",
			nodeID,
			"--start-time",
			start,
			"--staking-period",
			period,
			"--weight",
			weight,
			subnetName,
			"--" + constants.SkipUpdateFlag,
		},
		true,
		false,
	)
}

// simulates testnet join execution path on a local network
/* #nosec G204 */
func SimulateTestnetJoin(
	subnetName string,
	odysseygoConfig string,
	pluginDir string,
	nodeID string,
) string {
	// Check config exists
	exists, err := utils.SubnetConfigExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())

	// enable simulation of public network execution paths on a local network
	err = os.Setenv(constants.SimulatePublicNetwork, "true")
	gomega.Expect(err).Should(gomega.BeNil())

	cmd := exec.Command(
		CLIBinary,
		SubnetCmd,
		"join",
		"--testnet",
		"--odysseygo-config",
		odysseygoConfig,
		"--plugin-dir",
		pluginDir,
		"--nodeID",
		nodeID,
		"--force-write",
		subnetName,
		"--"+constants.SkipUpdateFlag,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(string(output))
		fmt.Println(err)
	}
	gomega.Expect(err).Should(gomega.BeNil())

	// disable simulation of public network execution paths on a local network
	err = os.Unsetenv(constants.SimulatePublicNetwork)
	gomega.Expect(err).Should(gomega.BeNil())

	return string(output)
}

// simulates mainnet join execution path on a local network
/* #nosec G204 */
func SimulateMainnetJoin(
	subnetName string,
	odysseygoConfig string,
	pluginDir string,
	nodeID string,
) string {
	// Check config exists
	exists, err := utils.SubnetConfigExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())

	// enable simulation of public network execution paths on a local network
	err = os.Setenv(constants.SimulatePublicNetwork, "true")
	gomega.Expect(err).Should(gomega.BeNil())

	cmd := exec.Command(
		CLIBinary,
		SubnetCmd,
		"join",
		"--mainnet",
		"--odysseygo-config",
		odysseygoConfig,
		"--plugin-dir",
		pluginDir,
		"--nodeID",
		nodeID,
		"--force-write",
		subnetName,
		"--"+constants.SkipUpdateFlag,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(cmd.String())
		fmt.Println(string(output))
		utils.PrintStdErr(err)
	}
	gomega.Expect(err).Should(gomega.BeNil())

	// disable simulation of public network execution paths on a local network
	err = os.Unsetenv(constants.SimulatePublicNetwork)
	gomega.Expect(err).Should(gomega.BeNil())

	return string(output)
}

/* #nosec G204 */
func ImportSubnetConfig(repoAlias string, subnetName string) {
	// Check config does not already exist
	exists, err := utils.SubnetConfigExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeFalse())
	// Check vm binary does not already exist
	exists, err = utils.SubnetCustomVMExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeFalse())

	// Create config
	cmd := exec.Command(
		CLIBinary,
		SubnetCmd,
		"import",
		"file",
		"--repo",
		repoAlias,
		"--subnet",
		subnetName,
		"--"+constants.SkipUpdateFlag,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		var (
			exitErr *exec.ExitError
			stderr  string
		)
		if errors.As(err, &exitErr) {
			stderr = string(exitErr.Stderr)
		}
		fmt.Println(string(output))
		fmt.Println(err)
		fmt.Println(stderr)
	}

	// Config should now exist
	exists, err = utils.OPMConfigExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())
	exists, err = utils.SubnetOPMVMExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())
}

/* #nosec G204 */
func ImportSubnetConfigFromURL(repoURL string, branch string, subnetName string) {
	// Check config does not already exist
	exists, err := utils.SubnetConfigExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeFalse())
	// Check vm binary does not already exist
	exists, err = utils.SubnetCustomVMExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeFalse())

	// Create config
	cmd := exec.Command(
		CLIBinary,
		SubnetCmd,
		"import",
		"file",
		"--repo",
		repoURL,
		"--branch",
		branch,
		"--subnet",
		subnetName,
		"--"+constants.SkipUpdateFlag,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		var (
			exitErr *exec.ExitError
			stderr  string
		)
		if errors.As(err, &exitErr) {
			stderr = string(exitErr.Stderr)
		}
		fmt.Println(string(output))
		utils.PrintStdErr(err)
		fmt.Println(stderr)
	}

	// Config should now exist
	exists, err = utils.OPMConfigExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())
	exists, err = utils.SubnetOPMVMExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())
}

/* #nosec G204 */
func DescribeSubnet(subnetName string) (string, error) {
	// Create config
	cmd := exec.Command(
		CLIBinary,
		SubnetCmd,
		"describe",
		subnetName,
		"--"+constants.SkipUpdateFlag,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(cmd.String())
		fmt.Println(string(output))
		utils.PrintStdErr(err)
	}
	return string(output), err
}

/* #nosec G204 */
func SimulateGetSubnetStatsTestnet(subnetName, subnetID string) string {
	// Check config does already exist:
	// We want to run stats on an existing subnet
	exists, err := utils.SubnetConfigExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())

	// add the subnet ID to the `testnet` section so that the `stats` command
	// can find it (as this is a simulation with a `local` network,
	// it got written in to the `local` network section)
	err = utils.AddSubnetIDToSidecar(subnetName, models.TestnetNetwork, subnetID)
	gomega.Expect(err).Should(gomega.BeNil())
	// run stats
	cmd := exec.Command(
		CLIBinary,
		SubnetCmd,
		"stats",
		subnetName,
		"--testnet",
		"--"+constants.SkipUpdateFlag,
	)
	output, err := cmd.CombinedOutput()
	var exitErr *exec.ExitError
	if err != nil {
		stderr := ""
		if errors.As(err, &exitErr) {
			stderr = string(exitErr.Stderr)
		}
		fmt.Println(string(output))
		fmt.Println(err)
		fmt.Println(stderr)
	}
	gomega.Expect(exitErr).Should(gomega.BeNil())
	return string(output)
}

func TransformElasticSubnetLocally(subnetName string) (string, error) {
	// Check config exists
	exists, err := utils.SubnetConfigExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())

	cmd := exec.Command(
		CLIBinary,
		SubnetCmd,
		ElasticTransformCmd,
		"--local",
		"--tokenName",
		"BLIZZARD",
		"--tokenSymbol",
		"BRRR",
		"--default",
		"--force",
		subnetName,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		var stderr string
		fmt.Println(string(output))
		utils.PrintStdErr(err)
		fmt.Println(stderr)
	}
	return string(output), err
}

func TransformElasticSubnetLocallyandTransformValidators(subnetName string, stakeAmount string) (string, error) {
	// Check config exists
	exists, err := utils.SubnetConfigExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())

	cmd := exec.Command(
		CLIBinary,
		SubnetCmd,
		ElasticTransformCmd,
		"--local",
		"--tokenName",
		"BLIZZARD",
		"--tokenSymbol",
		"BRRR",
		"--default",
		"--force",
		"--transform-validators",
		"--stake-amount",
		stakeAmount,
		subnetName,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		var stderr string
		fmt.Println(string(output))
		utils.PrintStdErr(err)
		fmt.Println(stderr)
	}
	return string(output), err
}

func RemoveValidator(subnetName string, nodeID string) (string, error) {
	// Check config exists
	exists, err := utils.SubnetConfigExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())

	cmd := exec.Command(
		CLIBinary,
		SubnetCmd,
		RemoveValidatorCmd,
		"--local",
		"--nodeID",
		nodeID,
		subnetName,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		var stderr string
		fmt.Println(string(output))
		utils.PrintStdErr(err)
		fmt.Println(stderr)
	}
	return string(output), err
}

func AddPermissionlessValidator(subnetName string, nodeID string, stakeAmount string, stakingPeriod string) (string, error) {
	// Check config exists
	exists, err := utils.SubnetConfigExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())

	startTimeStr := time.Now().Add(constants.StakingStartLeadTime).UTC().Format(constants.TimeParseLayout)
	cmd := exec.Command(
		CLIBinary,
		SubnetCmd,
		JoinCmd,
		"--local",
		"--elastic",
		"--nodeID",
		nodeID,
		"--stake-amount",
		stakeAmount,
		"--start-time",
		startTimeStr,
		"--staking-period",
		stakingPeriod,
		subnetName,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		var stderr string
		fmt.Println(string(output))
		utils.PrintStdErr(err)
		fmt.Println(stderr)
	}
	return string(output), err
}

/* #nosec G204 */
func ListValidators(subnetName string, network string) (string, error) {
	// Create config
	cmd := exec.Command(
		CLIBinary,
		SubnetCmd,
		"validators",
		subnetName,
		"--"+network,
		"--"+constants.SkipUpdateFlag,
	)

	out, err := cmd.CombinedOutput()
	return string(out), err
}

func AddPermissionlessDelegator(subnetName string, nodeID string, stakeAmount string, stakingPeriod string) (string, error) {
	// Check config exists
	exists, err := utils.SubnetConfigExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())
	startTimeStr := time.Now().Add(constants.StakingStartLeadTime).UTC().Format(constants.TimeParseLayout)

	cmd := exec.Command(
		CLIBinary,
		SubnetCmd,
		AddPermissionlessDelegatorCmd,
		"--local",
		"--nodeID",
		nodeID,
		"--stake-amount",
		stakeAmount,
		"--start-time",
		startTimeStr,
		"--staking-period",
		stakingPeriod,
		subnetName,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		var stderr string
		fmt.Println(string(output))
		utils.PrintStdErr(err)
		fmt.Println(stderr)
	}
	return string(output), err
}
