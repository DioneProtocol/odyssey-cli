// Copyright (C) 2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package subnetcmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/DioneProtocol/odyssey-cli/pkg/binutils"
	"github.com/DioneProtocol/odyssey-cli/pkg/constants"
	"github.com/DioneProtocol/odyssey-cli/pkg/key"
	"github.com/DioneProtocol/odyssey-cli/pkg/keychain"
	"github.com/DioneProtocol/odyssey-cli/pkg/localnetworkinterface"
	"github.com/DioneProtocol/odyssey-cli/pkg/metrics"
	"github.com/DioneProtocol/odyssey-cli/pkg/models"
	"github.com/DioneProtocol/odyssey-cli/pkg/prompts"
	"github.com/DioneProtocol/odyssey-cli/pkg/subnet"
	"github.com/DioneProtocol/odyssey-cli/pkg/txutils"
	"github.com/DioneProtocol/odyssey-cli/pkg/ux"
	"github.com/DioneProtocol/odyssey-cli/pkg/vm"
	onrutils "github.com/DioneProtocol/odyssey-network-runner/utils"
	"github.com/DioneProtocol/odysseygo/ids"
	"github.com/DioneProtocol/odysseygo/utils/logging"
	"github.com/DioneProtocol/odysseygo/vms/omegavm/txs"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"golang.org/x/mod/semver"
)

var (
	deployLocal              bool
	deployDevnet             bool
	deployTestnet            bool
	deployMainnet            bool
	endpoint                 string
	sameControlKey           bool
	keyName                  string
	threshold                uint32
	controlKeys              []string
	subnetAuthKeys           []string
	userProvidedOdygoVersion string
	outputTxPath             string
	useLedger                bool
	useEwoq                  bool
	ledgerAddresses          []string
	subnetIDStr              string
	mainnetChainID           uint32
	skipCreatePrompt         bool
	odygoBinaryPath          string

	errMutuallyExclusiveNetworks = errors.New("--local, --testnet, --mainnet are mutually exclusive")

	errMutuallyExclusiveControlKeys = errors.New("--control-keys and --same-control-key are mutually exclusive")

	ErrMutuallyExclusiveKeyLedger = errors.New("key source flags --key, --ledger/--ledger-addrs are mutually exclusive")
	ErrStoredKeyOnMainnet         = errors.New("key --key is not available for mainnet operations")
)

// odyssey subnet deploy
func newDeployCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy [subnetName]",
		Short: "Deploys a subnet configuration",
		Long: `The subnet deploy command deploys your Subnet configuration locally, to Testnet, or to Mainnet.

At the end of the call, the command prints the RPC URL you can use to interact with the Subnet.

Odyssey-CLI only supports deploying an individual Subnet once per network. Subsequent
attempts to deploy the same Subnet to the same network (local, Testnet, Mainnet) aren't
allowed. If you'd like to redeploy a Subnet locally for testing, you must first call
odyssey network clean to reset all deployed chain state. Subsequent local deploys
redeploy the chain with fresh state. You can deploy the same Subnet to multiple networks,
so you can take your locally tested Subnet and deploy it on Testnet or Mainnet.`,
		SilenceUsage:      true,
		RunE:              deploySubnet,
		PersistentPostRun: handlePostRun,
		Args:              cobra.ExactArgs(1),
	}
	cmd.Flags().StringVar(&endpoint, "endpoint", "", "use the given endpoint for network operations")
	cmd.Flags().BoolVarP(&deployLocal, "local", "l", false, "deploy to a local network")
	cmd.Flags().BoolVar(&deployDevnet, "devnet", false, "deploy to a devnet network")
	cmd.Flags().BoolVarP(&deployTestnet, "testnet", "t", false, "deploy to testnet")
	cmd.Flags().BoolVarP(&deployMainnet, "mainnet", "m", false, "deploy to mainnet")
	cmd.Flags().StringVar(&userProvidedOdygoVersion, "odysseygo-version", "latest", "use this version of odysseygo (ex: v1.17.12)")
	cmd.Flags().StringVarP(&keyName, "key", "k", "", "select the key to use [testnet/devnet deploy only]")
	cmd.Flags().BoolVarP(&sameControlKey, "same-control-key", "s", false, "use the fee-paying key as control key")
	cmd.Flags().Uint32Var(&threshold, "threshold", 0, "required number of control key signatures to make subnet changes")
	cmd.Flags().StringSliceVar(&controlKeys, "control-keys", nil, "addresses that may make subnet changes")
	cmd.Flags().StringSliceVar(&subnetAuthKeys, "subnet-auth-keys", nil, "control keys that will be used to authenticate chain creation")
	cmd.Flags().StringVar(&outputTxPath, "output-tx-path", "", "file path of the blockchain creation tx")
	cmd.Flags().BoolVarP(&useEwoq, "ewoq", "e", false, "use ewoq key [testnet/devnet deploy only]")
	cmd.Flags().BoolVarP(&useLedger, "ledger", "g", false, "use ledger instead of key (always true on mainnet, defaults to false on testnet/devnet)")
	cmd.Flags().StringSliceVar(&ledgerAddresses, "ledger-addrs", []string{}, "use the given ledger addresses")
	cmd.Flags().StringVarP(&subnetIDStr, "subnet-id", "u", "", "deploy into given subnet id")
	cmd.Flags().Uint32Var(&mainnetChainID, "mainnet-chain-id", 0, "use different ChainID for mainnet deployment")
	cmd.Flags().StringVar(&odygoBinaryPath, "odysseygo-path", "", "use this odysseygo binary path")
	return cmd
}

func CallDeploy(
	cmd *cobra.Command,
	subnetName string,
	deployLocalParam bool,
	deployDevnetParam bool,
	deployTestnetParam bool,
	deployMainnetParam bool,
	endpointParam string,
	keyNameParam string,
	useLedgerParam bool,
	useEwoqParam bool,
	sameControlKeyParam bool,
) error {
	deployLocal = deployLocalParam
	deployTestnet = deployTestnetParam
	deployMainnet = deployMainnetParam
	deployDevnet = deployDevnetParam
	endpoint = endpointParam
	sameControlKey = sameControlKeyParam
	keyName = keyNameParam
	useLedger = useLedgerParam
	useEwoq = useEwoqParam
	return deploySubnet(cmd, []string{subnetName})
}

func getChainsInSubnet(subnetName string) ([]string, error) {
	subnets, err := os.ReadDir(app.GetSubnetDir())
	if err != nil {
		return nil, fmt.Errorf("failed to read baseDir: %w", err)
	}

	chains := []string{}

	for _, s := range subnets {
		if !s.IsDir() {
			continue
		}
		sidecarFile := filepath.Join(app.GetSubnetDir(), s.Name(), constants.SidecarFileName)
		if _, err := os.Stat(sidecarFile); err == nil {
			// read in sidecar file
			jsonBytes, err := os.ReadFile(sidecarFile)
			if err != nil {
				return nil, fmt.Errorf("failed reading file %s: %w", sidecarFile, err)
			}

			var sc models.Sidecar
			err = json.Unmarshal(jsonBytes, &sc)
			if err != nil {
				return nil, fmt.Errorf("failed unmarshalling file %s: %w", sidecarFile, err)
			}
			if sc.Subnet == subnetName {
				chains = append(chains, sc.Name)
			}
		}
	}
	return chains, nil
}

func checkSubnetEVMDefaultAddressNotInAlloc(network models.Network, chain string) error {
	if network.Kind != models.Local && network.Kind != models.Devnet && os.Getenv(constants.SimulatePublicNetwork) == "" {
		genesis, err := app.LoadEvmGenesis(chain)
		if err != nil {
			return err
		}
		allocAddressMap := genesis.Alloc
		for address := range allocAddressMap {
			if address.String() == vm.PrefundedEwoqAddress.String() {
				return fmt.Errorf("can't airdrop to default address on public networks, please edit the genesis by calling `odyssey subnet create %s --force`", chain)
			}
		}
	}
	return nil
}

func runDeploy(cmd *cobra.Command, args []string) error {
	skipCreatePrompt = true
	return deploySubnet(cmd, args)
}

func updateSubnetEVMGenesisChainID(genesisBytes []byte, newChainID uint) ([]byte, error) {
	var genesisMap map[string]interface{}
	if err := json.Unmarshal(genesisBytes, &genesisMap); err != nil {
		return nil, err
	}
	configI, ok := genesisMap["config"]
	if !ok {
		return nil, fmt.Errorf("config field not found on genesis")
	}
	config, ok := configI.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("expected genesis config field to be a map[string]interface, found %T", configI)
	}
	config["chainId"] = float64(newChainID)
	return json.MarshalIndent(genesisMap, "", "  ")
}

// updates sidecar with genesis mainnet id to use
// given either by cmdline flag, original genesis id, or id obtained from the user
func getSubnetEVMMainnetChainID(sc *models.Sidecar, subnetName string) error {
	// get original chain id
	evmGenesis, err := app.LoadEvmGenesis(subnetName)
	if err != nil {
		return err
	}
	if evmGenesis.Config == nil {
		return fmt.Errorf("invalid subnet evm genesis format: config is nil")
	}
	if evmGenesis.Config.ChainID == nil {
		return fmt.Errorf("invalid subnet evm genesis format: config chain id is nil")
	}
	originalChainID := evmGenesis.Config.ChainID.Uint64()
	// handle cmdline flag if given
	if mainnetChainID != 0 {
		sc.SubnetEVMMainnetChainID = uint(mainnetChainID)
	}
	// prompt the user
	if sc.SubnetEVMMainnetChainID == 0 {
		useSameChainID := "Use same ChainID"
		useNewChainID := "Use new ChainID"
		listOptions := []string{useNewChainID, useSameChainID}
		newChainIDPrompt := "Using the same ChainID for both Testnet and Mainnet could lead to a replay attack. Do you want to use a different ChainID?"
		var (
			err      error
			decision string
		)
		decision, err = app.Prompt.CaptureList(newChainIDPrompt, listOptions)
		if err != nil {
			return err
		}
		if decision == useSameChainID {
			sc.SubnetEVMMainnetChainID = uint(originalChainID)
		} else {
			ux.Logger.PrintToUser("Enter your subnet's ChainID. It can be any positive integer != %d.", originalChainID)
			newChainID, err := app.Prompt.CapturePositiveInt(
				"ChainID",
				[]prompts.Comparator{
					{
						Label: "Zero",
						Type:  prompts.MoreThan,
						Value: 0,
					},
					{
						Label: "Original Chain ID",
						Type:  prompts.NotEq,
						Value: originalChainID,
					},
				},
			)
			if err != nil {
				return err
			}
			sc.SubnetEVMMainnetChainID = uint(newChainID)
		}
	}
	return app.UpdateSidecar(sc)
}

// deploySubnet is the cobra command run for deploying subnets
func deploySubnet(cmd *cobra.Command, args []string) error {
	chains, err := ValidateSubnetNameAndGetChains(args)
	if err != nil {
		if !strings.Contains(err.Error(), "Invalid subnet") {
			return err
		}
		if !skipCreatePrompt {
			yes, promptErr := app.Prompt.CaptureNoYes(fmt.Sprintf("Subnet %s is not found. Do you want to create it first?", args[0]))
			if promptErr != nil {
				return promptErr
			}
			if !yes {
				return err
			}
		}
		createErr := createSubnetConfig(cmd, args)
		if createErr != nil {
			return createErr
		}
		chains, err = ValidateSubnetNameAndGetChains(args)
		if err != nil {
			return err
		}
		ux.Logger.PrintToUser("Now deploying subnet %s", chains[0])
	}

	chain := chains[0]

	sidecar, err := app.LoadSidecar(chain)
	if err != nil {
		return fmt.Errorf("failed to load sidecar for later update: %w", err)
	}

	if sidecar.ImportedFromOPM {
		return errors.New("unable to deploy subnets imported from a repo")
	}

	if outputTxPath != "" {
		if _, err := os.Stat(outputTxPath); err == nil {
			return fmt.Errorf("outputTxPath %q already exists", outputTxPath)
		}
	}

	network, err := GetNetworkFromCmdLineFlags(
		deployLocal,
		deployDevnet,
		deployTestnet,
		deployMainnet,
		endpoint,
		true,
		[]models.NetworkKind{models.Local, models.Devnet, models.Testnet, models.Mainnet},
	)
	if err != nil {
		return err
	}

	isEVMGenesis, err := hasSubnetEVMGenesis(chain)
	if err != nil {
		return err
	}
	if sidecar.VM == models.SubnetEvm && !isEVMGenesis {
		return fmt.Errorf("failed to validate SubnetEVM genesis format")
	}

	chainGenesis, err := app.LoadRawGenesis(chain)
	if err != nil {
		return err
	}

	if isEVMGenesis {
		// is is a subnet evm or a custom vm based on subnet evm
		if network.Kind == models.Mainnet {
			err = getSubnetEVMMainnetChainID(&sidecar, chain)
			if err != nil {
				return err
			}
			chainGenesis, err = updateSubnetEVMGenesisChainID(chainGenesis, sidecar.SubnetEVMMainnetChainID)
			if err != nil {
				return err
			}
		}
		err = checkSubnetEVMDefaultAddressNotInAlloc(network, chain)
		if err != nil {
			return err
		}
	}

	ux.Logger.PrintToUser("Deploying %s to %s", chains, network.Name())

	if network.Kind == models.Local {
		app.Log.Debug("Deploy local")

		genesisPath := app.GetGenesisPath(chain)

		// copy vm binary to the expected location, first downloading it if necessary
		var vmBin string
		switch sidecar.VM {
		case models.SubnetEvm:
			_, vmBin, err = binutils.SetupSubnetEVM(app, sidecar.VMVersion)
			if err != nil {
				return fmt.Errorf("failed to install subnet-evm: %w", err)
			}
		case models.CustomVM:
			vmBin = binutils.SetupCustomBin(app, chain)
		default:
			return fmt.Errorf("unknown vm: %s", sidecar.VM)
		}

		// check if selected version matches what is currently running
		nc := localnetworkinterface.NewStatusChecker()
		odygoVersion, err := CheckForInvalidDeployAndGetOdygoVersion(nc, sidecar.RPCVersion)
		if err != nil {
			return err
		}
		if odygoBinaryPath == "" {
			userProvidedOdygoVersion = odygoVersion
		}

		deployer := subnet.NewLocalDeployer(app, userProvidedOdygoVersion, odygoBinaryPath, vmBin)
		subnetID, blockchainID, err := deployer.DeployToLocalNetwork(chain, chainGenesis, genesisPath)
		if err != nil {
			if deployer.BackendStartedHere() {
				if innerErr := binutils.KillgRPCServerProcess(app); innerErr != nil {
					app.Log.Warn("tried to kill the gRPC server process but it failed", zap.Error(innerErr))
				}
			}
			return err
		}
		flags := make(map[string]string)
		flags[constants.Network] = network.Name()
		metrics.HandleTracking(cmd, app, flags)
		return app.UpdateSidecarNetworks(&sidecar, network, subnetID, blockchainID)
	}

	// from here on we are assuming a public deploy

	createSubnet := true
	var subnetID ids.ID
	if subnetIDStr != "" {
		subnetID, err = ids.FromString(subnetIDStr)
		if err != nil {
			return err
		}
		createSubnet = false
	} else if sidecar.Networks != nil {
		model, ok := sidecar.Networks[network.Name()]
		if ok {
			if model.SubnetID != ids.Empty && model.BlockchainID == ids.Empty {
				subnetID = model.SubnetID
				createSubnet = false
			}
		}
	}

	fee := network.GenesisParams().CreateBlockchainTxFee
	if createSubnet {
		fee += network.GenesisParams().CreateSubnetTxFee
	}
	kc, err := keychain.GetKeychainFromCmdLineFlags(
		app,
		constants.PayTxsFeesMsg,
		network,
		keyName,
		useEwoq,
		useLedger,
		ledgerAddresses,
		fee,
	)
	if err != nil {
		return err
	}

	network.HandlePublicNetworkSimulation()

	if createSubnet {
		// accept only one control keys specification
		if len(controlKeys) > 0 && sameControlKey {
			return errMutuallyExclusiveControlKeys
		}
		// use first fee-paying key as control key
		if sameControlKey {
			kcKeys, err := kc.OChainFormattedStrAddresses()
			if err != nil {
				return err
			}
			if len(kcKeys) == 0 {
				return fmt.Errorf("no keys found on keychain")
			}
			controlKeys = kcKeys[:1]
		}
		// prompt for control keys
		if controlKeys == nil {
			var cancelled bool
			controlKeys, cancelled, err = getControlKeys(kc)
			if err != nil {
				return err
			}
			if cancelled {
				ux.Logger.PrintToUser("User cancelled. No subnet deployed")
				return nil
			}
		}
		ux.Logger.PrintToUser("Your Subnet's control keys: %s", controlKeys)
		// validate and prompt for threshold
		if threshold == 0 && subnetAuthKeys != nil {
			threshold = uint32(len(subnetAuthKeys))
		}
		if int(threshold) > len(controlKeys) {
			return fmt.Errorf("given threshold is greater than number of control keys")
		}
		if threshold == 0 {
			threshold, err = getThreshold(len(controlKeys))
			if err != nil {
				return err
			}
		}
	} else {
		ux.Logger.PrintToUser(logging.Green.Wrap(
			fmt.Sprintf("Deploying into pre-existent subnet ID %s", subnetID.String()),
		))
		controlKeys, threshold, err = txutils.GetOwners(network, subnetID)
		if err != nil {
			return err
		}
	}

	// add control keys to the keychain whenever possible
	if err := kc.AddAddresses(controlKeys); err != nil {
		return err
	}

	kcKeys, err := kc.OChainFormattedStrAddresses()
	if err != nil {
		return err
	}

	// get keys for blockchain tx signing
	if subnetAuthKeys != nil {
		if err := prompts.CheckSubnetAuthKeys(kcKeys, subnetAuthKeys, controlKeys, threshold); err != nil {
			return err
		}
	} else {
		subnetAuthKeys, err = prompts.GetSubnetAuthKeys(app.Prompt, kcKeys, controlKeys, threshold)
		if err != nil {
			return err
		}
	}
	ux.Logger.PrintToUser("Your subnet auth keys for chain creation: %s", subnetAuthKeys)

	// deploy to public network
	deployer := subnet.NewPublicDeployer(app, kc, network)

	if createSubnet {
		subnetID, err = deployer.DeploySubnet(controlKeys, threshold)
		if err != nil {
			return err
		}
		// get the control keys in the same order as the tx
		controlKeys, threshold, err = txutils.GetOwners(network, subnetID)
		if err != nil {
			return err
		}
	}

	isFullySigned, blockchainID, tx, remainingSubnetAuthKeys, err := deployer.DeployBlockchain(controlKeys, subnetAuthKeys, subnetID, chain, chainGenesis)
	if err != nil {
		ux.Logger.PrintToUser(logging.Red.Wrap(
			fmt.Sprintf("error deploying blockchain: %s. fix the issue and try again with a new deploy cmd", err),
		))
	}

	savePartialTx := !isFullySigned && err == nil

	if err := PrintDeployResults(chain, subnetID, blockchainID); err != nil {
		return err
	}

	if savePartialTx {
		if err := SaveNotFullySignedTx(
			"Blockchain Creation",
			tx,
			chain,
			subnetAuthKeys,
			remainingSubnetAuthKeys,
			outputTxPath,
			false,
		); err != nil {
			return err
		}
	}

	flags := make(map[string]string)
	flags[constants.Network] = network.Name()
	metrics.HandleTracking(cmd, app, flags)

	// update sidecar
	// TODO: need to do something for backwards compatibility?
	return app.UpdateSidecarNetworks(&sidecar, network, subnetID, blockchainID)
}

func getControlKeys(kc *keychain.Keychain) ([]string, bool, error) {
	controlKeysInitialPrompt := "Configure which addresses may make changes to the subnet.\n" +
		"These addresses are known as your control keys. You will also\n" +
		"set how many control keys are required to make a subnet change (the threshold)."
	moreKeysPrompt := "How would you like to set your control keys?"

	ux.Logger.PrintToUser(controlKeysInitialPrompt)

	const (
		useAll = "Use all stored keys"
		custom = "Custom list"
	)

	var feePaying string
	var listOptions []string
	if kc.UsesLedger {
		feePaying = "Use ledger address"
	} else {
		feePaying = "Use fee-paying key"
	}
	if kc.Network.Kind == models.Mainnet {
		listOptions = []string{feePaying, custom}
	} else {
		listOptions = []string{feePaying, useAll, custom}
	}

	listDecision, err := app.Prompt.CaptureList(moreKeysPrompt, listOptions)
	if err != nil {
		return nil, false, err
	}

	var (
		keys      []string
		cancelled bool
	)

	switch listDecision {
	case feePaying:
		var kcKeys []string
		kcKeys, err = kc.OChainFormattedStrAddresses()
		if err != nil {
			return nil, false, err
		}
		if len(kcKeys) == 0 {
			return nil, false, fmt.Errorf("no keys found on keychain")
		}
		keys = kcKeys[:1]
	case useAll:
		keys, err = useAllKeys(kc.Network)
	case custom:
		keys, cancelled, err = enterCustomKeys(kc.Network)
	}
	if err != nil {
		return nil, false, err
	}
	if cancelled {
		return nil, true, nil
	}
	return keys, false, nil
}

func useAllKeys(network models.Network) ([]string, error) {
	existing := []string{}

	files, err := os.ReadDir(app.GetKeyDir())
	if err != nil {
		return nil, err
	}

	keyPaths := make([]string, 0, len(files))

	for _, f := range files {
		if strings.HasSuffix(f.Name(), constants.KeySuffix) {
			keyPaths = append(keyPaths, filepath.Join(app.GetKeyDir(), f.Name()))
		}
	}

	for _, kp := range keyPaths {
		k, err := key.LoadSoft(network.ID, kp)
		if err != nil {
			return nil, err
		}

		existing = append(existing, k.O()...)
	}

	return existing, nil
}

func enterCustomKeys(network models.Network) ([]string, bool, error) {
	controlKeysPrompt := "Enter control keys"
	for {
		// ask in a loop so that if some condition is not met we can keep asking
		controlKeys, cancelled, err := controlKeysLoop(controlKeysPrompt, network)
		if err != nil {
			return nil, false, err
		}
		if cancelled {
			return nil, cancelled, nil
		}
		if len(controlKeys) != 0 {
			return controlKeys, false, nil
		}
		ux.Logger.PrintToUser("This tool does not allow to proceed without any control key set")
	}
}

// controlKeysLoop asks as many controlkeys the user requires, until Done or Cancel is selected
func controlKeysLoop(controlKeysPrompt string, network models.Network) ([]string, bool, error) {
	label := "Control key"
	info := "Control keys are O-Chain addresses which have admin rights on the subnet.\n" +
		"Only private keys which control such addresses are allowed to make changes on the subnet"
	addressPrompt := "Enter O-Chain address (Example: O-...)"
	return prompts.CaptureListDecision(
		// we need this to be able to mock test
		app.Prompt,
		// the main prompt for entering address keys
		controlKeysPrompt,
		// the Capture function to use
		func(s string) (string, error) { return app.Prompt.CaptureOChainAddress(s, network) },
		// the prompt for each address
		addressPrompt,
		// label describes the entity we are prompting for (e.g. address, control key, etc.)
		label,
		// optional parameter to allow the user to print the info string for more information
		info,
	)
}

// getThreshold prompts for the threshold of addresses as a number
func getThreshold(maxLen int) (uint32, error) {
	if maxLen == 1 {
		return uint32(1), nil
	}
	// create a list of indexes so the user only has the option to choose what is the threshold
	// instead of entering
	indexList := make([]string, maxLen)
	for i := 0; i < maxLen; i++ {
		indexList[i] = strconv.Itoa(i + 1)
	}
	threshold, err := app.Prompt.CaptureList("Select required number of control key signatures to make a subnet change", indexList)
	if err != nil {
		return 0, err
	}
	intTh, err := strconv.ParseUint(threshold, 0, 32)
	if err != nil {
		return 0, err
	}
	// this now should technically not happen anymore, but let's leave it as a double stitch
	if int(intTh) > maxLen {
		return 0, fmt.Errorf("the threshold can't be bigger than the number of control keys")
	}
	return uint32(intTh), err
}

func ValidateSubnetNameAndGetChains(args []string) ([]string, error) {
	// this should not be necessary but some bright guy might just be creating
	// the genesis by hand or something...
	if err := checkInvalidSubnetNames(args[0]); err != nil {
		return nil, fmt.Errorf("subnet name %s is invalid: %w", args[0], err)
	}
	// Check subnet exists
	// TODO create a file that lists chains by subnet for fast querying
	chains, err := getChainsInSubnet(args[0])
	if err != nil {
		return nil, fmt.Errorf("failed to getChainsInSubnet: %w", err)
	}

	if len(chains) == 0 {
		return nil, errors.New("Invalid subnet " + args[0])
	}

	return chains, nil
}

func SaveNotFullySignedTx(
	txName string,
	tx *txs.Tx,
	chain string,
	subnetAuthKeys []string,
	remainingSubnetAuthKeys []string,
	outputTxPath string,
	forceOverwrite bool,
) error {
	signedCount := len(subnetAuthKeys) - len(remainingSubnetAuthKeys)
	ux.Logger.PrintToUser("")
	if signedCount == len(subnetAuthKeys) {
		ux.Logger.PrintToUser("All %d required %s signatures have been signed. "+
			"Saving tx to disk to enable commit.", len(subnetAuthKeys), txName)
	} else {
		ux.Logger.PrintToUser("%d of %d required %s signatures have been signed. "+
			"Saving tx to disk to enable remaining signing.", signedCount, len(subnetAuthKeys), txName)
	}
	if outputTxPath == "" {
		ux.Logger.PrintToUser("")
		var err error
		if forceOverwrite {
			outputTxPath, err = app.Prompt.CaptureString("Path to export partially signed tx to")
		} else {
			outputTxPath, err = app.Prompt.CaptureNewFilepath("Path to export partially signed tx to")
		}
		if err != nil {
			return err
		}
	}
	if forceOverwrite {
		ux.Logger.PrintToUser("")
		ux.Logger.PrintToUser("Overwriting %s", outputTxPath)
	}
	if err := txutils.SaveToDisk(tx, outputTxPath, forceOverwrite); err != nil {
		return err
	}
	if signedCount == len(subnetAuthKeys) {
		PrintReadyToSignMsg(chain, outputTxPath)
	} else {
		PrintRemainingToSignMsg(chain, remainingSubnetAuthKeys, outputTxPath)
	}
	return nil
}

func PrintReadyToSignMsg(
	chain string,
	outputTxPath string,
) {
	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("Tx is fully signed, and ready to be committed")
	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("Commit command:")
	ux.Logger.PrintToUser("  odyssey transaction commit %s --input-tx-filepath %s", chain, outputTxPath)
}

func PrintRemainingToSignMsg(
	chain string,
	remainingSubnetAuthKeys []string,
	outputTxPath string,
) {
	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("Addresses remaining to sign the tx")
	for _, subnetAuthKey := range remainingSubnetAuthKeys {
		ux.Logger.PrintToUser("  %s", subnetAuthKey)
	}
	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("Connect a ledger with one of the remaining addresses or choose a stored key "+
		"and run the signing command, or send %q to another user for signing.", outputTxPath)
	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("Signing command:")
	ux.Logger.PrintToUser("  odyssey transaction sign %s --input-tx-filepath %s", chain, outputTxPath)
	ux.Logger.PrintToUser("")
}

func PrintDeployResults(chain string, subnetID ids.ID, blockchainID ids.ID) error {
	vmID, err := onrutils.VMID(chain)
	if err != nil {
		return fmt.Errorf("failed to create VM ID from %s: %w", chain, err)
	}
	header := []string{"Deployment results", ""}
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(header)
	table.SetRowLine(true)
	table.SetAutoMergeCells(true)
	table.Append([]string{"Chain Name", chain})
	table.Append([]string{"Subnet ID", subnetID.String()})
	table.Append([]string{"VM ID", vmID.String()})
	if blockchainID != ids.Empty {
		table.Append([]string{"Blockchain ID", blockchainID.String()})
		table.Append([]string{"O-Chain TXID", blockchainID.String()})
	}
	table.Render()
	return nil
}

// Determines the appropriate version of odysseygo to run with. Returns an error if
// that version conflicts with the current deployment.
func CheckForInvalidDeployAndGetOdygoVersion(network localnetworkinterface.StatusChecker, configuredRPCVersion int) (string, error) {
	// get current network
	runningOdygoVersion, runningRPCVersion, networkRunning, err := network.GetCurrentNetworkVersion()
	if err != nil {
		return "", err
	}

	desiredOdygoVersion := userProvidedOdygoVersion

	// RPC Version was made available in the info API in odysseygo version v1.9.2. For prior versions,
	// we will need to skip this check.
	skipRPCCheck := false
	if semver.Compare(runningOdygoVersion, constants.OdysseyGoCompatibilityVersionAdded) == -1 {
		skipRPCCheck = true
	}

	if networkRunning {
		if userProvidedOdygoVersion == "latest" {
			if runningRPCVersion != configuredRPCVersion && !skipRPCCheck {
				return "", fmt.Errorf(
					"the current odysseygo deployment uses rpc version %d but your subnet has version %d and is not compatible",
					runningRPCVersion,
					configuredRPCVersion,
				)
			}
			desiredOdygoVersion = runningOdygoVersion
		} else if runningOdygoVersion != userProvidedOdygoVersion {
			// user wants a specific version
			return "", errors.New("incompatible odysseygo version selected")
		}
	} else if userProvidedOdygoVersion == "latest" {
		// find latest odygo version for this rpc version
		desiredOdygoVersion, err = vm.GetLatestOdysseyGoByProtocolVersion(
			app, configuredRPCVersion, constants.OdysseyGoCompatibilityURL)
		if err != nil {
			return "", err
		}
	}
	return desiredOdygoVersion, nil
}

func hasSubnetEVMGenesis(subnetName string) (bool, error) {
	if _, err := app.LoadRawGenesis(subnetName); err != nil {
		return false, err
	}
	// from here, we are sure to have a genesis file
	genesis, err := app.LoadEvmGenesis(subnetName)
	if err != nil {
		return false, nil
	}
	if err := genesis.Verify(); err != nil {
		return false, nil
	}
	return true, nil
}
