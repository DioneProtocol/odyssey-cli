// Copyright (C) 2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package transactioncmd

import (
	"errors"
	"fmt"

	"github.com/DioneProtocol/odyssey-cli/cmd/subnetcmd"
	"github.com/DioneProtocol/odyssey-cli/pkg/keychain"
	"github.com/DioneProtocol/odyssey-cli/pkg/models"
	"github.com/DioneProtocol/odyssey-cli/pkg/prompts"
	"github.com/DioneProtocol/odyssey-cli/pkg/subnet"
	"github.com/DioneProtocol/odyssey-cli/pkg/txutils"
	"github.com/DioneProtocol/odyssey-cli/pkg/ux"
	"github.com/DioneProtocol/odysseygo/ids"
	"github.com/spf13/cobra"
)

const inputTxPathFlag = "input-tx-filepath"

var (
	inputTxPath     string
	keyName         string
	useLedger       bool
	ledgerAddresses []string

	errNoSubnetID = errors.New("failed to find the subnet ID for this subnet, has it been deployed/created on this network?")
)

// odyssey transaction sign
func newTransactionSignCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "sign [subnetName]",
		Short:        "sign a transaction",
		Long:         "The transaction sign command signs a multisig transaction.",
		RunE:         signTx,
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
	}

	cmd.Flags().StringVar(&inputTxPath, inputTxPathFlag, "", "Path to the transaction file for signing")
	cmd.Flags().StringVarP(&keyName, "key", "k", "", "select the key to use [testnet only]")
	cmd.Flags().BoolVarP(&useLedger, "ledger", "g", false, "use ledger instead of key (always true on mainnet, defaults to false on testnet)")
	cmd.Flags().StringSliceVar(&ledgerAddresses, "ledger-addrs", []string{}, "use the given ledger addresses")
	return cmd
}

func signTx(_ *cobra.Command, args []string) error {
	var err error
	if inputTxPath == "" {
		inputTxPath, err = app.Prompt.CaptureExistingFilepath("What is the path to the transactions file which needs signing?")
		if err != nil {
			return err
		}
	}
	tx, err := txutils.LoadFromDisk(inputTxPath)
	if err != nil {
		return err
	}

	if len(ledgerAddresses) > 0 {
		useLedger = true
	}

	if useLedger && keyName != "" {
		return subnetcmd.ErrMutuallyExclusiveKeyLedger
	}

	// we need network to decide if ledger is forced (mainnet)
	network, err := txutils.GetNetwork(tx)
	if err != nil {
		return err
	}
	switch network.Kind {
	case models.Testnet, models.Local:
		if !useLedger && keyName == "" {
			useLedger, keyName, err = prompts.GetTestnetKeyOrLedger(app.Prompt, "sign transaction", app.GetKeyDir())
			if err != nil {
				return err
			}
		}
	case models.Mainnet:
		useLedger = true
		if keyName != "" {
			return subnetcmd.ErrStoredKeyOnMainnet
		}
	default:
		return errors.New("unsupported network")
	}

	// we need subnet wallet signing validation + process
	subnetName := args[0]
	sc, err := app.LoadSidecar(subnetName)
	if err != nil {
		return err
	}
	subnetID := sc.Networks[network.Name()].SubnetID
	if subnetID == ids.Empty {
		return errNoSubnetID
	}

	subnetIDFromTX, err := txutils.GetSubnetID(tx)
	if err != nil {
		return err
	}
	if subnetIDFromTX != ids.Empty {
		subnetID = subnetIDFromTX
	}

	controlKeys, _, err := txutils.GetOwners(network, subnetID)
	if err != nil {
		return err
	}

	// get the remaining tx signers so as to check that the wallet does contain an expected signer
	subnetAuthKeys, remainingSubnetAuthKeys, err := txutils.GetRemainingSigners(tx, controlKeys)
	if err != nil {
		return err
	}

	if len(remainingSubnetAuthKeys) == 0 {
		subnetcmd.PrintReadyToSignMsg(subnetName, inputTxPath)
		ux.Logger.PrintToUser("")
		return fmt.Errorf("tx is already fully signed")
	}

	// get keychain accessor
	kc, err := keychain.GetKeychain(app, false, useLedger, ledgerAddresses, keyName, network, 0)
	if err != nil {
		return err
	}

	// add control keys to the keychain whenever possible
	if err := kc.AddAddresses(controlKeys); err != nil {
		return err
	}

	deployer := subnet.NewPublicDeployer(app, kc, network)
	if err := deployer.Sign(tx, remainingSubnetAuthKeys, subnetID); err != nil {
		if errors.Is(err, subnet.ErrNoSubnetAuthKeysInWallet) {
			ux.Logger.PrintToUser("There are no required subnet auth keys present in the wallet")
			ux.Logger.PrintToUser("")
			ux.Logger.PrintToUser("Expected one of:")
			for _, addr := range remainingSubnetAuthKeys {
				ux.Logger.PrintToUser("  %s", addr)
			}
			ux.Logger.PrintToUser("")
			return fmt.Errorf("no remaining signer address present in wallet")
		}
		return err
	}

	// update the remaining tx signers after the signature has been done
	_, remainingSubnetAuthKeys, err = txutils.GetRemainingSigners(tx, controlKeys)
	if err != nil {
		return err
	}

	if err := subnetcmd.SaveNotFullySignedTx(
		"Tx",
		tx,
		subnetName,
		subnetAuthKeys,
		remainingSubnetAuthKeys,
		inputTxPath,
		true,
	); err != nil {
		return err
	}

	return nil
}
