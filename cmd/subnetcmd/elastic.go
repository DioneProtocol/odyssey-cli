// Copyright (C) 2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package subnetcmd

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"github.com/DioneProtocol/odyssey-cli/pkg/constants"
	es "github.com/DioneProtocol/odyssey-cli/pkg/elasticsubnet"
	"github.com/DioneProtocol/odyssey-cli/pkg/keychain"
	"github.com/DioneProtocol/odyssey-cli/pkg/metrics"
	"github.com/DioneProtocol/odyssey-cli/pkg/models"
	"github.com/DioneProtocol/odyssey-cli/pkg/prompts"
	subnet "github.com/DioneProtocol/odyssey-cli/pkg/subnet"
	"github.com/DioneProtocol/odyssey-cli/pkg/txutils"
	"github.com/DioneProtocol/odyssey-cli/pkg/utils"
	"github.com/DioneProtocol/odyssey-cli/pkg/ux"
	"github.com/DioneProtocol/odysseygo/genesis"
	"github.com/DioneProtocol/odysseygo/ids"
	"github.com/DioneProtocol/odysseygo/vms/components/verify"
	"github.com/DioneProtocol/odysseygo/vms/omegavm"
	"github.com/DioneProtocol/odysseygo/vms/secp256k1fx"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

const (
	localDeployment      = "Existing local deployment"
	testnetDeployment    = "Testnet"
	mainnetDeployment    = "Mainnet (coming soon)"
	subnetIsElasticError = "subnet is already elastic"
)

var (
	transformLocal      bool
	tokenNameFlag       string
	tokenSymbolFlag     string
	useDefaultConfig    bool
	overrideWarning     bool
	transformValidators bool
	denominationFlag    int
)

// odyssey subnet elastic
func newElasticCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "elastic [subnetName]",
		Short: "Transforms a subnet into elastic subnet",
		Long: `The elastic command enables anyone to be a validator of a Subnet by simply staking its token on the
O-Chain. When enabling Elastic Validation, the creator permanently locks the Subnet from future modification
(they relinquish their control keys), specifies an Odyssey Native Token (ONT) that validators must use for staking
and that will be distributed as staking rewards, and provides a set of parameters that govern how the Subnet’s staking
mechanics will work.`,
		SilenceUsage:      true,
		Args:              cobra.ExactArgs(1),
		RunE:              transformElasticSubnet,
		PersistentPostRun: handlePostRun,
	}
	cmd.Flags().BoolVarP(&transformLocal, "local", "l", false, "transform a subnet on a local network")
	cmd.Flags().BoolVar(&deployTestnet, "testnet", false, "remove from `testnet` deployment")
	cmd.Flags().StringVar(&tokenNameFlag, "tokenName", "", "specify the token name")
	cmd.Flags().StringVar(&tokenSymbolFlag, "tokenSymbol", "", "specify the token symbol")
	cmd.Flags().BoolVar(&useDefaultConfig, "default", false, "use default elastic subnet config values")
	cmd.Flags().BoolVar(&overrideWarning, "force", false, "override transform into elastic subnet warning")
	cmd.Flags().Uint64Var(&stakeAmount, "stake-amount", 0, "amount of tokens to stake on validator")
	cmd.Flags().StringVar(&startTimeStr, "start-time", "", "start time that validator starts validating")
	cmd.Flags().DurationVar(&duration, "staking-period", 0, "how long validator validates for after start time")
	cmd.Flags().BoolVar(&transformValidators, "transform-validators", false, "transform validators to permissionless validators")
	cmd.Flags().IntVar(&denominationFlag, "denomination", -1, "specify the token denomination")
	cmd.Flags().BoolVarP(&useLedger, "ledger", "g", false, "use ledger instead of key (always true on mainnet, defaults to false on testnet)")
	cmd.Flags().StringSliceVar(&ledgerAddresses, "ledger-addrs", []string{}, "use the given ledger addresses")
	cmd.Flags().StringVarP(&keyName, "key", "k", "", "select the key to use [testnet only]")
	cmd.Flags().StringSliceVar(&subnetAuthKeys, "subnet-auth-keys", nil, "control keys that will be used to authenticate the transformSubnet tx")
	cmd.Flags().StringVar(&outputTxPath, "output-tx-path", "", "file path of the transformSubnet tx")
	return cmd
}

func checkIfSubnetIsElasticOnLocal(sc models.Sidecar) bool {
	if _, ok := sc.ElasticSubnet[models.Local.String()]; ok {
		return true
	}
	return false
}

func createAssetID(deployer *subnet.PublicDeployer,
	maxSupply uint64,
	subnetID ids.ID,
	tokenName string,
	tokenSymbol string,
	tokenDenomination int,
	recipientAddr ids.ShortID,
) (ids.ID, error) {
	if tokenDenomination > math.MaxUint8 {
		return ids.Empty, errors.New("token denomination cannot exceed 32")
	}
	owner := &secp256k1fx.OutputOwners{
		Threshold: 1,
		Addrs: []ids.ShortID{
			recipientAddr,
		},
	}
	initialState := map[uint32][]verify.State{
		0: {
			&secp256k1fx.TransferOutput{
				Amt:          maxSupply,
				OutputOwners: *owner,
			},
		},
	}
	return deployer.CreateAssetTx(subnetID, tokenName, tokenSymbol, byte(tokenDenomination), initialState)
}

func exportToOChain(deployer *subnet.PublicDeployer,
	subnetID ids.ID,
	subnetAssetID ids.ID,
	recipientAddr ids.ShortID,
	maxSupply uint64,
) (ids.ID, error) {
	owner := &secp256k1fx.OutputOwners{
		Threshold: 1,
		Addrs: []ids.ShortID{
			recipientAddr,
		},
	}
	return deployer.ExportToOChainTx(subnetID, subnetAssetID, owner, maxSupply)
}

func importFromAChain(deployer *subnet.PublicDeployer,
	subnetID ids.ID,
	recipientAddr ids.ShortID,
) (ids.ID, error) {
	owner := &secp256k1fx.OutputOwners{
		Threshold: 1,
		Addrs: []ids.ShortID{
			recipientAddr,
		},
	}
	return deployer.ImportFromAChain(subnetID, owner)
}

func promptDeployFirst(cmd *cobra.Command, args []string, prompt string, err error) error {
	yes, promptErr := app.Prompt.CaptureNoYes(prompt)
	if promptErr != nil {
		return promptErr
	}
	if !yes {
		return err
	}
	return runDeploy(cmd, args)
}

func transformElasticSubnet(cmd *cobra.Command, args []string) error {
	subnetName := args[0]

	if !app.SubnetConfigExists(subnetName) {
		prompt := fmt.Sprintf("Subnet %s is not created yet. Do you want to create it first?", args[0])
		err := promptDeployFirst(cmd, args, prompt, errors.New("subnet does not exist"))
		if err != nil {
			return err
		}
		ux.Logger.PrintToUser("Now transforming subnet ...")
	}

	sc, err := app.LoadSidecar(subnetName)
	if err != nil {
		return fmt.Errorf("unable to load sidecar: %w", err)
	}

	networkOptions := getNetworkOptions(sc)
	if len(networkOptions) == 0 {
		prompt := fmt.Sprintf("Subnet %s is not deployed yet. Do you want to deploy it first?", args[0])
		err := promptDeployFirst(cmd, args, prompt, nil)
		if err != nil {
			return err
		}
		// need to refresh sidecar if we deployed
		sc, err = app.LoadSidecar(subnetName)
		if err != nil {
			return fmt.Errorf("unable to load sidecar: %w", err)
		}
		ux.Logger.PrintToUser("Now transforming subnet ... \n")
	}

	network := models.UndefinedNetwork
	switch {
	case deployTestnet:
		network = models.TestnetNetwork
	case deployMainnet:
		network = models.MainnetNetwork
	case transformLocal:
		network = models.LocalNetwork
	}

	if network.Kind == models.Undefined {
		networkToUpgrade, err := selectNetworkToTransform(sc)
		if err != nil {
			return err
		}
		switch networkToUpgrade {
		case localDeployment:
			network = models.LocalNetwork
		case testnetDeployment:
			network = models.TestnetNetwork
		default:
			return errors.New("elastic subnet transformation is not yet supported on Mainnet")
		}
	}

	if outputTxPath != "" {
		if _, err := os.Stat(outputTxPath); err == nil {
			return fmt.Errorf("outputTxPath %q already exists", outputTxPath)
		}
	}

	if len(ledgerAddresses) > 0 {
		useLedger = true
	}

	if useLedger && keyName != "" {
		return ErrMutuallyExclusiveKeyLedger
	}

	subnetID := sc.Networks[network.Name()].SubnetID
	if os.Getenv(constants.SimulatePublicNetwork) != "" {
		subnetID = sc.Networks[models.Local.String()].SubnetID
	}
	if subnetID == ids.Empty {
		return errNoSubnetID
	}

	if network.Kind != models.Local {
		isAlreadyElastic, err := CheckSubnetIsElastic(subnetID, network)
		if err != nil && err.Error() != subnetIsElasticError {
			return err
		}
		if isAlreadyElastic {
			return errors.New(subnetIsElasticError)
		}
	}

	tokenName := ""
	if tokenNameFlag == "" {
		tokenName, err = getTokenName()
		if err != nil {
			return err
		}
	} else {
		tokenName = tokenNameFlag
	}

	tokenSymbol := ""
	if tokenSymbolFlag == "" {
		tokenSymbol, err = getTokenSymbol()
		if err != nil {
			return err
		}
	} else {
		tokenSymbol = tokenSymbolFlag
	}

	tokenDenomination := 0
	if network.Kind != models.Local {
		if denominationFlag == -1 {
			tokenDenomination, err = getTokenDenomination()
			if err != nil {
				return err
			}
		} else {
			tokenDenomination = denominationFlag
		}
	}

	elasticSubnetConfig, err := es.GetElasticSubnetConfig(app, tokenSymbol, useDefaultConfig)
	if err != nil {
		return err
	}
	elasticSubnetConfig.SubnetID = subnetID

	switch network.Kind {
	case models.Local:
		return transformElasticSubnetLocal(sc, subnetName, tokenName, tokenSymbol, elasticSubnetConfig, cmd)
	case models.Testnet:
		if !useLedger && keyName == "" {
			useLedger, keyName, err = prompts.GetTestnetKeyOrLedger(app.Prompt, constants.PayTxsFeesMsg, app.GetKeyDir())
			if err != nil {
				return err
			}
		}
	case models.Mainnet:
		return errors.New("unsupported network")
	default:
		return errors.New("unsupported network")
	}

	// get keychain accessor
	fee := network.GenesisParams().CreateAssetTxFee + network.GenesisParams().TransformSubnetTxFee + network.GenesisParams().TxFee*2
	kc, err := keychain.GetKeychain(app, false, useLedger, ledgerAddresses, keyName, network, fee)
	if err != nil {
		return err
	}

	network.HandlePublicNetworkSimulation()

	recipientAddr := kc.Addresses().List()[0]
	deployer := subnet.NewPublicDeployer(app, kc, network)
	txHasOccurred, txID := checkIfTxHasOccurred(&sc, network, "CreateAssetTx")
	var assetID ids.ID
	// TODO: replace sleep functions with sticky API sessions
	if txHasOccurred {
		ux.Logger.PrintToUser(fmt.Sprintf("Skipping CreateAssetTx, transforming subnet with asset ID %s...", txID.String()))
		assetID = txID
	} else {
		assetID, err = createAssetID(deployer, elasticSubnetConfig.MaxSupply, subnetID, tokenName, tokenSymbol, tokenDenomination, recipientAddr)
		if err != nil {
			return err
		}
		err = app.UpdateSidecarElasticSubnetPartialTx(&sc, network, "CreateAssetTx", assetID)
		if err != nil {
			return err
		}
		// we need to sleep after each operation to make sure that UTXO is available for consumption
		time.Sleep(5 * time.Second)
	}

	txHasOccurred, _ = checkIfTxHasOccurred(&sc, network, "ExportTx")
	if !txHasOccurred {
		txID, err = exportToOChain(deployer, subnetID, assetID, recipientAddr, elasticSubnetConfig.MaxSupply)
		if err != nil {
			return err
		}
		err = app.UpdateSidecarElasticSubnetPartialTx(&sc, network, "ExportTx", txID)
		if err != nil {
			return err
		}
		time.Sleep(5 * time.Second)
	} else {
		ux.Logger.PrintToUser("Skipping ExportTx...")
	}

	txHasOccurred, _ = checkIfTxHasOccurred(&sc, network, "ImportTx")
	if !txHasOccurred {
		txID, err = importFromAChain(deployer, subnetID, recipientAddr)
		if err != nil {
			return err
		}
		err = app.UpdateSidecarElasticSubnetPartialTx(&sc, network, "ImportTx", txID)
		if err != nil {
			return err
		}
		time.Sleep(5 * time.Second)
	} else {
		ux.Logger.PrintToUser("Skipping ImportTx...")
	}

	controlKeys, threshold, err := txutils.GetOwners(network, subnetID)
	if err != nil {
		return err
	}

	// add control keys to the keychain whenever possible
	if err := kc.AddAddresses(controlKeys); err != nil {
		return err
	}

	kcKeys, err := kc.OChainFormattedStrAddresses()
	if err != nil {
		return err
	}

	// get keys for add validator tx signing
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
	ux.Logger.PrintToUser("Your subnet auth keys for issue transform subnet tx: %s", subnetAuthKeys)

	isFullySigned, txID, tx, remainingSubnetAuthKeys, err := deployer.TransformSubnetTx(controlKeys, subnetAuthKeys, elasticSubnetConfig, subnetID, assetID)
	if err != nil {
		return err
	}
	flags := make(map[string]string)
	flags[constants.Network] = network.Name()
	if !isFullySigned {
		flags[constants.MultiSig] = "multi-sig"
	} else {
		flags[constants.MultiSig] = "non-multi-sig"
	}
	metrics.HandleTracking(cmd, app, flags)
	if !isFullySigned {
		if err := SaveNotFullySignedTx(
			"Transform Subnet",
			tx,
			subnetName,
			subnetAuthKeys,
			remainingSubnetAuthKeys,
			outputTxPath,
			false,
		); err != nil {
			return err
		}
	} else {
		elasticSubnetConfig.AssetID = assetID
		if err = app.CreateElasticSubnetConfig(subnetName, &elasticSubnetConfig); err != nil {
			return err
		}
		if err = app.UpdateSidecarElasticSubnet(&sc, network, subnetID, assetID, txID, tokenName, tokenSymbol); err != nil {
			return fmt.Errorf("elastic subnet transformation was successful, but failed to update sidecar: %w", err)
		}
		PrintTransformResults(subnetName, txID, subnetID, tokenName, tokenSymbol, assetID)
	}
	return nil
}

func transformElasticSubnetLocal(sc models.Sidecar, subnetName string, tokenName string, tokenSymbol string, elasticSubnetConfig models.ElasticSubnetConfig, cmd *cobra.Command) error {
	if checkIfSubnetIsElasticOnLocal(sc) {
		return fmt.Errorf("%s is already an elastic subnet", subnetName)
	}
	var err error
	subnetID := sc.Networks[models.Local.String()].SubnetID
	if subnetID == ids.Empty {
		return errNoSubnetID
	}

	if !overrideWarning {
		yes, err := app.Prompt.CaptureNoYes("WARNING: Transforming a Permissioned Subnet into an Elastic Subnet is an irreversible operation. Continue?")
		if err != nil {
			return err
		}
		if !yes {
			return nil
		}
	}

	ux.Logger.PrintToUser("Starting Elastic Subnet Transformation")
	cancel := make(chan struct{})
	go ux.PrintWait(cancel)
	testKey := genesis.EWOQKey
	keyChain := secp256k1fx.NewKeychain(testKey)
	txID, assetID, err := subnet.IssueTransformSubnetTx(elasticSubnetConfig, keyChain, subnetID, tokenName, tokenSymbol, elasticSubnetConfig.MaxSupply)
	close(cancel)
	if err != nil {
		return err
	}
	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("Subnet Successfully Transformed To Elastic Subnet!")

	elasticSubnetConfig.AssetID = assetID
	if err = app.CreateElasticSubnetConfig(subnetName, &elasticSubnetConfig); err != nil {
		return err
	}
	if err = app.UpdateSidecarElasticSubnet(&sc, models.LocalNetwork, subnetID, assetID, txID, tokenName, tokenSymbol); err != nil {
		return fmt.Errorf("elastic subnet transformation was successful, but failed to update sidecar: %w", err)
	}

	if !transformValidators {
		if !overrideWarning {
			yes, err := app.Prompt.CaptureNoYes("Do you want to transform existing validators to permissionless validators with equal weight? " +
				"Press <No> if you want to customize the structure of your permissionless validators")
			if err != nil {
				return err
			}
			if !yes {
				return nil
			}
			ux.Logger.PrintToUser("Transforming validators to permissionless validators")
			if err = transformValidatorsToPermissionlessLocal(sc, subnetID, subnetName); err != nil {
				return err
			}
		}
	} else {
		ux.Logger.PrintToUser("Transforming validators to permissionless validators")
		if err = transformValidatorsToPermissionlessLocal(sc, subnetID, subnetName); err != nil {
			return err
		}
	}

	PrintTransformResults(subnetName, txID, subnetID, tokenName, tokenSymbol, assetID)
	flags := make(map[string]string)
	flags[constants.Network] = models.Local.String()
	metrics.HandleTracking(cmd, app, flags)
	return nil
}

// select which network to transform to elastic subnet
func promptNetworkElastic(sc models.Sidecar, prompt string) (string, error) {
	var networkOptions []string
	for network := range sc.Networks {
		switch network {
		case models.Local.String():
			networkOptions = append(networkOptions, localDeployment)
		case models.Testnet.String():
			networkOptions = append(networkOptions, testnetDeployment)
		case models.Mainnet.String():
			networkOptions = append(networkOptions, mainnetDeployment)
		}
	}

	if len(networkOptions) == 0 {
		return "", errors.New("no deployment target available, please first deploy created subnet")
	}

	selectedDeployment, err := app.Prompt.CaptureList(prompt, networkOptions)
	if err != nil {
		return "", err
	}
	return selectedDeployment, nil
}

func getNetworkOptions(sc models.Sidecar) []string {
	var networkOptions []string
	for network := range sc.Networks {
		switch network {
		case models.Local.String():
			networkOptions = append(networkOptions, localDeployment)
		case models.Testnet.String():
			networkOptions = append(networkOptions, testnetDeployment)
		case models.Mainnet.String():
			networkOptions = append(networkOptions, mainnetDeployment)
		}
	}
	return networkOptions
}

// select which network to transform to elastic subnet
func selectNetworkToTransform(sc models.Sidecar) (string, error) {
	networkPrompt := "Which network should transform into an elastic Subnet?"
	networkOptions := getNetworkOptions(sc)
	if len(networkOptions) == 0 {
		return "", errors.New("no deployment target available, please first deploy created subnet")
	}

	selectedDeployment, err := app.Prompt.CaptureList(networkPrompt, networkOptions)
	if err != nil {
		return "", err
	}
	return selectedDeployment, nil
}

func PrintTransformResults(chain string, txID ids.ID, subnetID ids.ID, tokenName string, tokenSymbol string, assetID ids.ID) {
	const art = "\n  ______ _           _   _         _____       _                _     _______                   __                        _____                              __       _ " +
		"\n |  ____| |         | | (_)       / ____|     | |              | |   |__   __|                 / _|                      / ____|                            / _|     | |" +
		"\n | |__  | | __ _ ___| |_ _  ___  | (___  _   _| |__  _ __   ___| |_     | |_ __ __ _ _ __  ___| |_ ___  _ __ _ __ ___   | (___  _   _  ___ ___ ___  ___ ___| |_ _   _| |" +
		"\n |  __| | |/ _` / __| __| |/ __|  \\___ \\| | | | '_ \\| '_ \\ / _ \\ __|    | | '__/ _` | '_ \\/ __|  _/ _ \\| '__| '_ ` _ \\   \\___ \\| | | |/ __/ __/ _ \\/ __/ __|  _| | | | |" +
		"\n | |____| | (_| \\__ \\ |_| | (__   ____) | |_| | |_) | | | |  __/ |_     | | | | (_| | | | \\__ \\ || (_) | |  | | | | | |  ____) | |_| | (_| (_|  __/\\__ \\__ \\ | | |_| | |" +
		"\n |______|_|\\__,_|___/\\__|_|\\___| |_____/ \\__,_|_.__/|_| |_|\\___|\\__|    |_|_|  \\__,_|_| |_|___/_| \\___/|_|  |_| |_| |_| |_____/ \\__,_|\\___\\___\\___||___/___/_|  \\__,_|_|" +
		"\n"
	fmt.Print(art)

	table := tablewriter.NewWriter(os.Stdout)
	table.SetRowLine(true)
	table.SetAutoMergeCells(true)
	table.Append([]string{"Token Name", tokenName})
	table.Append([]string{"Token Symbol", tokenSymbol})
	table.Append([]string{"Asset ID", assetID.String()})
	table.Append([]string{"Chain Name", chain})
	table.Append([]string{"Subnet ID", subnetID.String()})
	table.Append([]string{"O-Chain TXID", txID.String()})
	table.Render()
}

func getTokenName() (string, error) {
	ux.Logger.PrintToUser("Select a name for your subnet's native token")
	tokenName, err := app.Prompt.CaptureString("Token name")
	if err != nil {
		return "", err
	}
	return tokenName, nil
}

func getTokenSymbol() (string, error) {
	ux.Logger.PrintToUser("Select a symbol for your subnet's native token")
	tokenSymbol, err := app.Prompt.CaptureString("Token symbol")
	if err != nil {
		return "", err
	}
	return tokenSymbol, nil
}

func checkAllLocalNodesAreCurrentValidators(subnetID ids.ID) error {
	api := constants.LocalAPIEndpoint
	oClient := omegavm.NewClient(api)

	ctx := context.Background()
	validators, err := oClient.GetCurrentValidators(ctx, subnetID, nil)
	if err != nil {
		return err
	}
	defaultLocalNetworkNodeIDs, err := getLocalNetworkIDs()
	if err != nil {
		return err
	}
	for _, localVal := range defaultLocalNetworkNodeIDs {
		currentValidator := false
		for _, validator := range validators {
			if validator.NodeID.String() == localVal {
				currentValidator = true
			}
		}
		if !currentValidator {
			return fmt.Errorf("%s is still not a current validator of the elastic subnet", localVal)
		}
	}
	return nil
}

func transformValidatorsToPermissionlessLocal(sc models.Sidecar, subnetID ids.ID, subnetName string) error {
	stakedTokenAmount, err := promptStakeAmount(subnetName, true, models.LocalNetwork)
	if err != nil {
		return err
	}

	validators, err := subnet.GetSubnetValidators(subnetID)
	if err != nil {
		return err
	}

	validatorList := make([]ids.NodeID, len(validators))
	for i, v := range validators {
		validatorList[i] = v.NodeID
	}

	numToRemoveInitially := len(validatorList) - 1
	for _, validator := range validatorList {
		// Remove first 4 nodes locally, wait for minimum lead time (25 seconds) and then remove the last node
		// so that we don't end up with a subnet without any current validators
		if numToRemoveInitially > 0 {
			err = handleRemoveAndAddValidators(sc, subnetID, validator, stakedTokenAmount)
			if err != nil {
				return err
			}
			numToRemoveInitially -= 1
		} else {
			ux.Logger.PrintToUser("Waiting for the first four nodes to be activated as permissionless validators...")
			time.Sleep(constants.StakingMinimumLeadTime)
			err = handleRemoveAndAddValidators(sc, subnetID, validator, stakedTokenAmount)
			if err != nil {
				return err
			}
		}
	}
	time.Sleep(constants.StakingMinimumLeadTime)
	return checkAllLocalNodesAreCurrentValidators(subnetID)
}

func handleRemoveAndAddValidators(sc models.Sidecar, subnetID ids.ID, validator ids.NodeID, stakedAmount uint64) error {
	startTime := time.Now().Add(constants.StakingMinimumLeadTime).UTC()
	duration = 24 * 50 * time.Hour
	endTime := uint64(startTime.Add(duration).Unix())
	testKey := genesis.EWOQKey
	keyChain := secp256k1fx.NewKeychain(testKey)
	_, err := subnet.IssueRemoveSubnetValidatorTx(keyChain, subnetID, validator)
	if err != nil {
		return err
	}
	ux.Logger.PrintToUser(fmt.Sprintf("Validator %s removed", validator.String()))
	assetID := sc.ElasticSubnet[models.Local.String()].AssetID
	txID, err := subnet.IssueAddPermissionlessValidatorTx(keyChain, subnetID, validator, stakedAmount, assetID, uint64(startTime.Unix()), endTime)
	if err != nil {
		return err
	}
	ux.Logger.PrintToUser(fmt.Sprintf("%s successfully joined elastic subnet as permissionless validator!", validator.String()))
	if err = app.UpdateSidecarPermissionlessValidator(&sc, models.LocalNetwork, validator.String(), txID); err != nil {
		return fmt.Errorf("joining permissionless subnet was successful, but failed to update sidecar: %w", err)
	}
	return nil
}

func getTokenDenomination() (int, error) {
	ux.Logger.PrintToUser("What's the denomination for your token?")
	ux.Logger.PrintToUser("Denomination determines how balances of this asset are displayed by user interfaces. " +
		"If denomination is 0, 100 units of this asset are displayed as 100. If denomination is 1, 100 units of this asset are displayed as 10.0.")
	tokenDenomination, err := app.Prompt.CapturePositiveInt(
		"Token Denomination",
		[]prompts.Comparator{
			{
				Label: "Min Denomination Value",
				Type:  prompts.MoreThanEq,
				Value: 0,
			},
			{
				Label: "Max Denomination Value",
				Type:  prompts.LessThanEq,
				Value: 32,
			},
		},
	)
	if err != nil {
		return 0, err
	}
	return tokenDenomination, nil
}

func CheckSubnetIsElastic(subnetID ids.ID, network models.Network) (bool, error) {
	oClient := omegavm.NewClient(network.Endpoint)
	ctx, cancel := utils.GetAPIContext()
	defer cancel()
	_, _, err := oClient.GetCurrentSupply(ctx, subnetID)
	if err != nil {
		// if subnet is already elastic it will return "not found" error
		if strings.Contains(err.Error(), "not found") {
			return false, errors.New(subnetIsElasticError)
		}
		return false, err
	}
	return true, nil
}

func checkIfTxHasOccurred(
	sc *models.Sidecar,
	network models.Network,
	txName string,
) (bool, ids.ID) {
	if sc.ElasticSubnet == nil {
		return false, ids.Empty
	}
	if sc.ElasticSubnet[network.Name()].Txs != nil {
		txID, ok := sc.ElasticSubnet[network.Name()].Txs[txName]
		if ok {
			return true, txID
		}
	}
	return false, ids.Empty
}
