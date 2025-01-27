// Copyright (C) 2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package opm

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/DioneProtocol/odyssey-cli/cmd/subnetcmd/upgradecmd"
	"github.com/DioneProtocol/odyssey-cli/pkg/application"
	"github.com/DioneProtocol/odyssey-cli/pkg/binutils"
	"github.com/DioneProtocol/odyssey-cli/pkg/constants"
	"github.com/DioneProtocol/odyssey-cli/pkg/models"
	"github.com/DioneProtocol/odyssey-cli/tests/e2e/commands"
	"github.com/DioneProtocol/odyssey-cli/tests/e2e/utils"
	onrutils "github.com/DioneProtocol/odyssey-network-runner/utils"
	"github.com/DioneProtocol/odysseygo/ids"
	"github.com/DioneProtocol/odysseygo/utils/logging"
	"github.com/DioneProtocol/subnet-evm/params"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

const (
	subnetName       = "e2eSubnetTest"
	secondSubnetName = "e2eSecondSubnetTest"

	subnetEVMVersion1 = "v0.5.6"
	subnetEVMVersion2 = "v0.5.6"

	odygoRPC1Version = "v1.10.10"
	odygoRPC2Version = "v1.10.10"

	controlKeys = "O-custom18jma8ppw3nhx5r4ap8clazz0dps7rv5u9xde7p"
	keyName     = "ewoq"

	upgradeBytesPath = "tests/e2e/assets/test_upgrade.json"

	upgradeBytesPath2 = "tests/e2e/assets/test_upgrade_2.json"
)

var (
	binaryToVersion map[string]string
	err             error
)

// need to have this outside the normal suite because of the BeforeEach
var _ = ginkgo.Describe("[Upgrade expect network failure]", ginkgo.Ordered, func() {
	ginkgo.AfterEach(func() {
		commands.CleanNetworkHard()
		err := utils.DeleteConfigs(subnetName)
		gomega.Expect(err).Should(gomega.BeNil())
	})

	ginkgo.It("fails on stopped network", func() {
		commands.CreateSubnetEvmConfig(subnetName, utils.SubnetEvmGenesisPath)

		_, err = commands.ImportUpgradeBytes(subnetName, upgradeBytesPath)
		gomega.Expect(err).Should(gomega.BeNil())

		// we want to simulate a situation here where the subnet has been deployed
		// but the network is stopped
		// the code would detect it hasn't been deployed yet so report that error first
		// therefore we can just manually edit the file to fake it had been deployed
		app := application.New()
		app.Setup(utils.GetBaseDir(), logging.NoLog{}, nil, nil, nil)
		sc := models.Sidecar{
			Name:     subnetName,
			Subnet:   subnetName,
			Networks: make(map[string]models.NetworkData),
		}
		sc.Networks[models.Local.String()] = models.NetworkData{
			SubnetID:     ids.GenerateTestID(),
			BlockchainID: ids.GenerateTestID(),
		}
		err = app.UpdateSidecar(&sc)
		gomega.Expect(err).Should(gomega.BeNil())

		out, err := commands.ApplyUpgradeLocal(subnetName)
		gomega.Expect(err).Should(gomega.HaveOccurred())
		gomega.Expect(out).Should(gomega.ContainSubstring(binutils.ErrGRPCTimeout.Error()))
	})
})

// upgrade a public network
// the approach is rather simple: import the upgrade file,
// call the apply command which "just" installs the file at an expected path,
// and then check the file is there and has the correct content.
var _ = ginkgo.Describe("[Upgrade public network]", ginkgo.Ordered, func() {
	ginkgo.AfterEach(func() {
		commands.CleanNetworkHard()
		err := utils.DeleteConfigs(subnetName)
		gomega.Expect(err).Should(gomega.BeNil())
	})

	ginkgo.It("can create and apply to public node", func() {
		commands.CreateSubnetEvmConfig(subnetName, utils.SubnetEvmGenesisPath)

		// simulate as if this had already been deployed to testnet
		// by just entering fake data into the struct
		app := application.New()
		app.Setup(utils.GetBaseDir(), logging.NoLog{}, nil, nil, nil)

		sc, err := app.LoadSidecar(subnetName)
		gomega.Expect(err).Should(gomega.BeNil())

		blockchainID := ids.GenerateTestID()
		sc.Networks = make(map[string]models.NetworkData)
		sc.Networks[models.Testnet.String()] = models.NetworkData{
			SubnetID:     ids.GenerateTestID(),
			BlockchainID: blockchainID,
		}
		err = app.UpdateSidecar(&sc)
		gomega.Expect(err).Should(gomega.BeNil())

		// import the upgrade bytes file so have one
		_, err = commands.ImportUpgradeBytes(subnetName, upgradeBytesPath)
		gomega.Expect(err).Should(gomega.BeNil())

		// we'll set a fake chain config dir to not mess up with a potential real one
		// in the system
		odysseygoConfigDir, err := os.MkdirTemp("", "cli-tmp-odygo-conf-dir")
		gomega.Expect(err).Should(gomega.BeNil())
		defer os.RemoveAll(odysseygoConfigDir)

		// now we try to apply
		_, err = commands.ApplyUpgradeToPublicNode(subnetName, odysseygoConfigDir)
		gomega.Expect(err).Should(gomega.BeNil())

		// we expect the file to be present at the expected location and being
		// the same content as the original one
		expectedPath := filepath.Join(odysseygoConfigDir, blockchainID.String(), constants.UpgradeBytesFileName)
		gomega.Expect(expectedPath).Should(gomega.BeARegularFile())
		ori, err := os.ReadFile(upgradeBytesPath)
		gomega.Expect(err).Should(gomega.BeNil())
		cp, err := os.ReadFile(expectedPath)
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(ori).Should(gomega.Equal(cp))
	})
})

var _ = ginkgo.Describe("[Upgrade local network]", ginkgo.Ordered, func() {
	_ = ginkgo.BeforeAll(func() {
		mapper := utils.NewVersionMapper()
		binaryToVersion, err = utils.GetVersionMapping(mapper)
		gomega.Expect(err).Should(gomega.BeNil())
	})

	ginkgo.BeforeEach(func() {
		output, err := commands.CreateKeyFromPath(keyName, utils.EwoqKeyPath)
		if err != nil {
			fmt.Println(output)
			utils.PrintStdErr(err)
		}
		gomega.Expect(err).Should(gomega.BeNil())
	})

	ginkgo.AfterEach(func() {
		commands.CleanNetworkHard()
		err := utils.DeleteConfigs(subnetName)
		gomega.Expect(err).Should(gomega.BeNil())
		err = utils.DeleteConfigs(secondSubnetName)
		gomega.Expect(err).Should(gomega.BeNil())
		_ = utils.DeleteKey(keyName)
		utils.DeleteCustomBinary(subnetName)
	})

	ginkgo.It("fails on undeployed subnet", func() {
		commands.CreateSubnetEvmConfig(subnetName, utils.SubnetEvmGenesisPath)

		_, err = commands.ImportUpgradeBytes(subnetName, upgradeBytesPath)
		gomega.Expect(err).Should(gomega.BeNil())

		_ = commands.StartNetwork()

		out, err := commands.ApplyUpgradeLocal(subnetName)
		gomega.Expect(err).Should(gomega.HaveOccurred())
		gomega.Expect(out).Should(gomega.ContainSubstring(upgradecmd.ErrSubnetNotDeployedOutput))
	})

	ginkgo.It("can create and apply to locally running subnet", func() {
		commands.CreateSubnetEvmConfig(subnetName, utils.SubnetEvmGenesisPath)

		deployOutput := commands.DeploySubnetLocally(subnetName)

		_, err = commands.ImportUpgradeBytes(subnetName, upgradeBytesPath)
		gomega.Expect(err).Should(gomega.BeNil())

		_, err = commands.ApplyUpgradeLocal(subnetName)
		gomega.Expect(err).Should(gomega.BeNil())

		upgradeBytes, err := os.ReadFile(upgradeBytesPath)
		gomega.Expect(err).Should(gomega.BeNil())

		var precmpUpgrades params.UpgradeConfig
		err = json.Unmarshal(upgradeBytes, &precmpUpgrades)
		gomega.Expect(err).Should(gomega.BeNil())

		rpcs, err := utils.ParseRPCsFromOutput(deployOutput)
		if err != nil {
			fmt.Println(deployOutput)
		}
		err = utils.CheckUpgradeIsDeployed(rpcs[0], precmpUpgrades)
		gomega.Expect(err).Should(gomega.BeNil())

		app := application.New()
		app.Setup(utils.GetBaseDir(), logging.NoLog{}, nil, nil, nil)

		stripped := stripWhitespaces(string(upgradeBytes))
		lockUpgradeBytes, err := app.ReadLockUpgradeFile(subnetName)
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect([]byte(stripped)).Should(gomega.Equal(lockUpgradeBytes))
	})

	ginkgo.It("can't upgrade transactionAllowList precompile because admin address doesn't have enough token", func() {
		commands.CreateSubnetEvmConfig(subnetName, utils.SubnetEvmGenesisPath)

		commands.DeploySubnetLocally(subnetName)

		_, err = commands.ImportUpgradeBytes(subnetName, upgradeBytesPath2)
		gomega.Expect(err).Should(gomega.BeNil())

		_, err = commands.ApplyUpgradeLocal(subnetName)
		gomega.Expect(err).Should(gomega.HaveOccurred())
	})

	ginkgo.It("can upgrade transactionAllowList precompile because admin address has enough tokens", func() {
		commands.CreateSubnetEvmConfig(subnetName, utils.SubnetEvmGenesisPath)

		commands.DeploySubnetLocally(subnetName)

		_, err = commands.ImportUpgradeBytes(subnetName, upgradeBytesPath)
		gomega.Expect(err).Should(gomega.BeNil())

		_, err = commands.ApplyUpgradeLocal(subnetName)
		gomega.Expect(err).Should(gomega.BeNil())
	})

	ginkgo.It("upgrade SubnetEVM local deployment", func() {
		commands.CreateSubnetEvmConfigWithVersion(subnetName, utils.SubnetEvmGenesisPath, subnetEVMVersion1)
		deployOutput := commands.DeploySubnetLocally(subnetName)
		rpcs, err := utils.ParseRPCsFromOutput(deployOutput)
		if err != nil {
			fmt.Println(deployOutput)
		}

		// check running version
		// remove string suffix starting with /ext
		nodeURI := strings.Split(rpcs[0], "/ext")[0]
		vmid, err := onrutils.VMID(subnetName)
		gomega.Expect(err).Should(gomega.BeNil())
		version, err := utils.GetNodeVMVersion(nodeURI, vmid.String())
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(version).Should(gomega.Equal(subnetEVMVersion1))

		// stop network
		commands.StopNetwork()

		// upgrade
		commands.UpgradeVMLocal(subnetName, subnetEVMVersion2)

		// restart network
		commands.StartNetwork()

		// check running version
		version, err = utils.GetNodeVMVersion(nodeURI, vmid.String())
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(version).Should(gomega.Equal(subnetEVMVersion2))

		commands.DeleteSubnetConfig(subnetName)
	})

	ginkgo.It("upgrade custom vm local deployment", func() {
		// download vm bins
		customVMPath1, err := utils.DownloadCustomVMBin(subnetEVMVersion1)
		gomega.Expect(err).Should(gomega.BeNil())
		customVMPath2, err := utils.DownloadCustomVMBin(subnetEVMVersion2)
		gomega.Expect(err).Should(gomega.BeNil())

		// create and deploy
		commands.CreateCustomVMConfig(subnetName, utils.SubnetEvmGenesisPath, customVMPath1)
		// need to set odygo version manually since VMs are custom
		commands.StartNetworkWithVersion(odygoRPC1Version)
		deployOutput := commands.DeploySubnetLocally(subnetName)
		rpcs, err := utils.ParseRPCsFromOutput(deployOutput)
		if err != nil {
			fmt.Println(deployOutput)
		}

		// check running version
		// remove string suffix starting with /ext from rpc url to get node uri
		nodeURI := strings.Split(rpcs[0], "/ext")[0]
		vmid, err := onrutils.VMID(subnetName)
		gomega.Expect(err).Should(gomega.BeNil())
		version, err := utils.GetNodeVMVersion(nodeURI, vmid.String())
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(version).Should(gomega.Equal(subnetEVMVersion1))

		// stop network
		commands.StopNetwork()

		// upgrade
		commands.UpgradeCustomVMLocal(subnetName, customVMPath2)

		// restart network
		commands.StartNetworkWithVersion(odygoRPC2Version)

		// check running version
		version, err = utils.GetNodeVMVersion(nodeURI, vmid.String())
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(version).Should(gomega.Equal(subnetEVMVersion2))

		commands.DeleteSubnetConfig(subnetName)
	})
})

func stripWhitespaces(str string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			// if the character is a space, drop it
			return -1
		}
		// else keep it in the string
		return r
	}, str)
}
