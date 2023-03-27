// Copyright (C) 2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package subnetcmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/ava-labs/avalanche-cli/pkg/constants"
	"github.com/ava-labs/avalanche-cli/pkg/models"
	"github.com/ava-labs/avalanche-cli/pkg/prompts"
	"github.com/ava-labs/avalanche-cli/pkg/subnet"
	"github.com/ava-labs/avalanche-cli/pkg/ux"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/spf13/cobra"
)

// avalanche subnet deploy
func newRemoveValidatorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "removeValidator [subnetName]",
		Short: "Remove a permissioned validator from your subnet",
		Long: `The subnet removeValidator command stops a whitelisted, subnet network validator from
validating your deployed Subnet.

To remove the validator from the Subnet's allow list, you first need to provide
the subnetName and the validator's unique NodeID. You can bypass
these prompts by providing the values with flags.`,
		SilenceUsage: true,
		RunE:         removeValidator,
		Args:         cobra.ExactArgs(1),
	}
	cmd.Flags().StringVarP(&keyName, "key", "k", "", "select the key to use [fuji deploy only]")
	cmd.Flags().StringVar(&nodeIDStr, "nodeID", "", "set the NodeID of the validator to add")
	cmd.Flags().BoolVar(&deployLocal, "local", false, "remove from the locally deployed Subnet")
	cmd.Flags().BoolVar(&deployTestnet, "fuji", false, "remove from `fuji` deployment (alias for `testnet`)")
	cmd.Flags().BoolVar(&deployTestnet, "testnet", false, "remove from `testnet` deployment (alias for `fuji`)")
	cmd.Flags().BoolVar(&deployMainnet, "mainnet", false, "remove from `mainnet` deployment")
	cmd.Flags().StringSliceVar(&subnetAuthKeys, "subnet-auth-keys", nil, "control keys that will be used to authenticate the removeValidator tx")
	cmd.Flags().StringVar(&outputTxPath, "output-tx-path", "", "file path of the removeValidator tx")
	cmd.Flags().BoolVarP(&useLedger, "ledger", "g", false, "use ledger instead of key (always true on mainnet, defaults to false on fuji)")
	cmd.Flags().StringSliceVar(&ledgerAddresses, "ledger-addrs", []string{}, "use the given ledger addresses")
	return cmd
}

func removeValidator(_ *cobra.Command, args []string) error {
	var (
		nodeID ids.NodeID
		err    error
	)

	var network models.Network
	switch {
	case deployTestnet:
		network = models.Fuji
	case deployMainnet:
		network = models.Mainnet
	}

	if network == models.Undefined {
		networkStr, err := app.Prompt.CaptureList(
			"Choose a network to remove a validator from",
			[]string{models.Local.String(), models.Fuji.String(), models.Mainnet.String()},
		)
		if err != nil {
			return err
		}
		network = models.NetworkFromString(networkStr)
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
		return ErrMutuallyExlusiveKeyLedger
	}

	switch network {
	case models.Local:
		if err = removeFromLocal(); err != nil {
			return err
		}
	case models.Fuji:
		if !useLedger && keyName == "" {
			useLedger, keyName, err = prompts.GetFujiKeyOrLedger(app.Prompt, app.GetKeyDir())
			if err != nil {
				return err
			}
		}
	case models.Mainnet:
		useLedger = true
		if keyName != "" {
			return ErrStoredKeyOnMainnet
		}
	default:
		return errors.New("unsupported network")
	}

	// used in E2E to simulate public network execution paths on a local network
	if os.Getenv(constants.SimulatePublicNetwork) != "" {
		network = models.Local
	}

	chains, err := validateSubnetNameAndGetChains(args)
	if err != nil {
		return err
	}
	subnetName := chains[0]
	sc, err := app.LoadSidecar(subnetName)
	if err != nil {
		return err
	}

	subnetID := sc.Networks[network.String()].SubnetID
	if subnetID == ids.Empty {
		return errNoSubnetID
	}

	controlKeys, threshold, err := subnet.GetOwners(network, subnetID)
	if err != nil {
		return err
	}

	// get keys for add validator tx signing
	if subnetAuthKeys != nil {
		if err := prompts.CheckSubnetAuthKeys(subnetAuthKeys, controlKeys, threshold); err != nil {
			return err
		}
	} else {
		subnetAuthKeys, err = prompts.GetSubnetAuthKeys(app.Prompt, controlKeys, threshold)
		if err != nil {
			return err
		}
	}
	ux.Logger.PrintToUser("Your subnet auth keys for add validator tx creation: %s", subnetAuthKeys)

	if nodeIDStr == "" {
		nodeID, err = promptNodeID()
		if err != nil {
			return err
		}
	} else {
		nodeID, err = ids.NodeIDFromString(nodeIDStr)
		if err != nil {
			return err
		}
	}

	// TODO check that this guy actually is a validator on the subnet

	ux.Logger.PrintToUser("NodeID: %s", nodeID.String())
	ux.Logger.PrintToUser("Network: %s", network.String())
	ux.Logger.PrintToUser("Inputs complete, issuing transaction to remove the sepecified validator information...")

	// get keychain accesor
	kc, err := GetKeychain(useLedger, ledgerAddresses, keyName, network)
	if err != nil {
		return err
	}
	deployer := subnet.NewPublicDeployer(app, useLedger, kc, network)
	isFullySigned, tx, err := deployer.RemoveValidator(subnetAuthKeys, subnetID, nodeID)
	if err != nil {
		return err
	}
	if !isFullySigned {
		if err := SaveNotFullySignedTx(
			"Remove Validator",
			tx,
			network,
			subnetName,
			subnetID,
			subnetAuthKeys,
			outputTxPath,
			false,
		); err != nil {
			return err
		}
	}

	return err
}

func removeFromLocal() error {
	return nil
}
