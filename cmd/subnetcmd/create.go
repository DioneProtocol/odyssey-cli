// Copyright (C) 2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package subnetcmd

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/DioneProtocol/odyssey-cli/pkg/constants"
	"github.com/DioneProtocol/odyssey-cli/pkg/metrics"

	"github.com/DioneProtocol/odyssey-cli/pkg/models"
	"github.com/DioneProtocol/odyssey-cli/pkg/ux"
	"github.com/DioneProtocol/odyssey-cli/pkg/vm"
	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"
)

const (
	forceFlag = "force"
	latest    = "latest"
)

var (
	forceCreate         bool
	useSubnetEvm        bool
	genesisFile         string
	vmFile              string
	useCustom           bool
	evmVersion          string
	evmChainID          uint64
	evmToken            string
	evmDefaults         bool
	useLatestEvmVersion bool
	useRepo             bool

	errIllegalNameCharacter = errors.New(
		"illegal name character: only letters, no special characters allowed")
)

// odyssey subnet create
func newCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create [subnetName]",
		Short: "Create a new subnet configuration",
		Long: `The subnet create command builds a new genesis file to configure your Subnet.
By default, the command runs an interactive wizard. It walks you through
all the steps you need to create your first Subnet.

The tool supports deploying Subnet-EVM, and custom VMs. You
can create a custom, user-generated genesis with a custom VM by providing
the path to your genesis and VM binaries with the --genesis and --vm flags.

By default, running the command with a subnetName that already exists
causes the command to fail. If you’d like to overwrite an existing
configuration, pass the -f flag.`,
		SilenceUsage:      true,
		Args:              cobra.ExactArgs(1),
		RunE:              createSubnetConfig,
		PersistentPostRun: handlePostRun,
	}
	cmd.Flags().StringVar(&genesisFile, "genesis", "", "file path of genesis to use")
	cmd.Flags().BoolVar(&useSubnetEvm, "evm", false, "use the Subnet-EVM as the base template")
	cmd.Flags().StringVar(&evmVersion, "vm-version", "", "version of Subnet-EVM template to use")
	cmd.Flags().Uint64Var(&evmChainID, "evm-chain-id", 0, "chain ID to use with Subnet-EVM")
	cmd.Flags().StringVar(&evmToken, "evm-token", "", "token name to use with Subnet-EVM")
	cmd.Flags().BoolVar(&evmDefaults, "evm-defaults", false, "use default settings for fees/airdrop/precompiles with Subnet-EVM")
	cmd.Flags().BoolVar(&useCustom, "custom", false, "use a custom VM template")
	cmd.Flags().BoolVar(&useLatestEvmVersion, latest, false, "use latest Subnet-EVM version, takes precedence over --vm-version")
	cmd.Flags().BoolVarP(&forceCreate, forceFlag, "f", false, "overwrite the existing configuration if one exists")
	cmd.Flags().StringVar(&vmFile, "vm", "", "file path of custom vm to use. alias to custom-vm-path")
	cmd.Flags().StringVar(&vmFile, "custom-vm-path", "", "file path of custom vm to use")
	cmd.Flags().StringVar(&customVMRepoURL, "custom-vm-repo-url", "", "custom vm repository url")
	cmd.Flags().StringVar(&customVMBranch, "custom-vm-branch", "", "custom vm branch")
	cmd.Flags().StringVar(&customVMBuildScript, "custom-vm-build-script", "", "custom vm build-script")
	cmd.Flags().BoolVar(&useRepo, "from-github-repo", false, "generate custom VM binary from github repository")
	return cmd
}

func CallCreate(
	cmd *cobra.Command,
	subnetName string,
	forceCreateParam bool,
	genesisFileParam string,
	useSubnetEvmParam bool,
	useCustomParam bool,
	evmVersionParam string,
	evmChainIDParam uint64,
	evmTokenParam string,
	evmDefaultsParam bool,
	useLatestEvmVersionParam bool,
	customVMRepoURLParam string,
	customVMBranchParam string,
	customVMBuildScriptParam string,
) error {
	forceCreate = forceCreateParam
	genesisFile = genesisFileParam
	useSubnetEvm = useSubnetEvmParam
	evmVersion = evmVersionParam
	evmChainID = evmChainIDParam
	evmToken = evmTokenParam
	evmDefaults = evmDefaultsParam
	useLatestEvmVersion = useLatestEvmVersionParam
	useCustom = useCustomParam
	customVMRepoURL = customVMRepoURLParam
	customVMBranch = customVMBranchParam
	customVMBuildScript = customVMBuildScriptParam
	return createSubnetConfig(cmd, []string{subnetName})
}

func moreThanOneVMSelected() bool {
	vmVars := []bool{useSubnetEvm, useCustom}
	firstSelect := false
	for _, val := range vmVars {
		if firstSelect && val {
			return true
		} else if val {
			firstSelect = true
		}
	}
	return false
}

func getVMFromFlag() models.VMType {
	if useSubnetEvm {
		return models.SubnetEvm
	}
	if useCustom {
		return models.CustomVM
	}
	return ""
}

// override postrun function from root.go, so that we don't double send metrics for the same command
func handlePostRun(_ *cobra.Command, _ []string) {}

func createSubnetConfig(cmd *cobra.Command, args []string) error {
	subnetName := args[0]
	if app.GenesisExists(subnetName) && !forceCreate {
		return errors.New("configuration already exists. Use --" + forceFlag + " parameter to overwrite")
	}

	if err := checkInvalidSubnetNames(subnetName); err != nil {
		return fmt.Errorf("subnet name %q is invalid: %w", subnetName, err)
	}

	if moreThanOneVMSelected() {
		return errors.New("too many VMs selected. Provide at most one VM selection flag")
	}

	subnetType := getVMFromFlag()

	if subnetType == "" {
		subnetTypeStr, err := app.Prompt.CaptureList(
			"Choose your VM",
			[]string{models.SubnetEvm, models.CustomVM},
		)
		if err != nil {
			return err
		}
		subnetType = models.VMTypeFromString(subnetTypeStr)
	}

	var (
		genesisBytes []byte
		sc           *models.Sidecar
		err          error
	)

	if useLatestEvmVersion {
		evmVersion = latest
	}

	if evmVersion != latest && evmVersion != "" && !semver.IsValid(evmVersion) {
		return fmt.Errorf("invalid version string, should be semantic version (ex: v1.1.1): %s", evmVersion)
	}

	switch subnetType {
	case models.SubnetEvm:
		genesisBytes, sc, err = vm.CreateEvmSubnetConfig(app, subnetName, genesisFile, evmVersion, evmChainID, evmToken, evmDefaults)
		if err != nil {
			return err
		}
	case models.CustomVM:
		genesisBytes, sc, err = vm.CreateCustomSubnetConfig(
			app,
			subnetName,
			genesisFile,
			useRepo,
			customVMRepoURL,
			customVMBranch,
			customVMBuildScript,
			vmFile,
		)
		if err != nil {
			return err
		}
	default:
		return errors.New("not implemented")
	}

	if err = app.WriteGenesisFile(subnetName, genesisBytes); err != nil {
		return err
	}

	sc.ImportedFromOPM = false
	if err = app.CreateSidecar(sc); err != nil {
		return err
	}
	if subnetType == models.SubnetEvm {
		err = sendMetrics(cmd, subnetType.RepoName(), subnetName)
		if err != nil {
			return err
		}
	}
	ux.Logger.PrintToUser("Successfully created subnet configuration")
	return nil
}

func sendMetrics(cmd *cobra.Command, repoName, subnetName string) error {
	flags := make(map[string]string)
	flags[constants.SubnetType] = repoName
	genesis, err := app.LoadEvmGenesis(subnetName)
	if err != nil {
		return err
	}
	conf := genesis.Config.GenesisPrecompiles
	precompiles := make([]string, 6)
	for precompileName := range conf {
		precompileTag := "precompile-" + precompileName
		flags[precompileTag] = precompileName
		precompiles = append(precompiles, precompileName)
	}
	numAirdropAddresses := len(genesis.Alloc)
	for address := range genesis.Alloc {
		if address.String() != vm.PrefundedEwoqAddress.String() {
			precompileTag := "precompile-" + constants.CustomAirdrop
			flags[precompileTag] = constants.CustomAirdrop
			precompiles = append(precompiles, constants.CustomAirdrop)
			break
		}
	}
	sort.Strings(precompiles)
	precompilesJoined := strings.Join(precompiles, ",")
	flags[constants.PrecompileType] = precompilesJoined
	flags[constants.NumberOfAirdrops] = strconv.Itoa(numAirdropAddresses)
	metrics.HandleTracking(cmd, app, flags)
	return nil
}

func checkInvalidSubnetNames(name string) error {
	// this is currently exactly the same code as in odysseygo/vms/omegavm/create_chain_tx.go
	for _, r := range name {
		if r > unicode.MaxASCII || !(unicode.IsLetter(r) || unicode.IsNumber(r) || r == ' ') {
			return errIllegalNameCharacter
		}
	}

	return nil
}
