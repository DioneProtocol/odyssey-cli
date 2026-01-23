// Copyright (C) 2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package keycmd

import (
	"context"
	"fmt"
	"time"

	"github.com/DioneProtocol/odyssey-cli/pkg/key"
	"github.com/DioneProtocol/odyssey-cli/pkg/models"
	"github.com/DioneProtocol/odyssey-cli/pkg/prompts"
	"github.com/DioneProtocol/odyssey-cli/pkg/subnet"
	"github.com/DioneProtocol/odyssey-cli/pkg/utils"
	"github.com/DioneProtocol/odyssey-cli/pkg/ux"
	"github.com/DioneProtocol/odysseygo/ids"
	odygoconstants "github.com/DioneProtocol/odysseygo/utils/constants"
	"github.com/DioneProtocol/odysseygo/utils/crypto/keychain"
	ledger "github.com/DioneProtocol/odysseygo/utils/crypto/ledger"
	"github.com/DioneProtocol/odysseygo/utils/formatting/address"
	"github.com/DioneProtocol/odysseygo/utils/logging"
	"github.com/DioneProtocol/odysseygo/utils/units"
	alphatxs "github.com/DioneProtocol/odysseygo/vms/alpha/txs"
	"github.com/DioneProtocol/odysseygo/vms/components/dione"
	"github.com/DioneProtocol/odysseygo/vms/omegavm/txs"
	"github.com/DioneProtocol/odysseygo/vms/secp256k1fx"
	"github.com/DioneProtocol/odysseygo/wallet/subnet/primary"
	"github.com/DioneProtocol/odysseygo/wallet/subnet/primary/common"
	"github.com/spf13/cobra"
)

const (
	sendFlag                = "send"
	receiveFlag             = "receive"
	keyNameFlag             = "key"
	ledgerIndexFlag         = "ledger"
	receiverAddrFlag        = "target-addr"
	amountFlag              = "amount"
	wrongLedgerIndexVal     = 32768
	receiveRecoveryStepFlag = "receive-recovery-step"

	sourceChainFlag      = "source-chain"
	destinationChainFlag = "destination-chain"
)

const (
	OChain = "O-Chain"
	AChain = "A-Chain"
	DChain = "D-Chain"
)

var (
	send                bool
	receive             bool
	keyName             string
	ledgerIndex         uint32
	force               bool
	receiverAddrStr     string
	amountFlt           float64
	receiveRecoveryStep uint64
	sourceChain         string
	destinationChain    string
)

func newTransferCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "transfer [options]",
		Short:        "Fund a ledger address or stored key from another one",
		Long:         `The key transfer command allows to transfer funds between stored keys or ledger addresses.`,
		RunE:         transferF,
		Args:         cobra.ExactArgs(0),
		SilenceUsage: true,
	}
	cmd.Flags().BoolVarP(
		&force,
		forceFlag,
		"f",
		false,
		"avoid transfer confirmation",
	)
	cmd.Flags().BoolVarP(
		&local,
		localFlag,
		"l",
		false,
		"transfer between local network addresses",
	)
	cmd.Flags().BoolVarP(
		&testnet,
		testnetFlag,
		"t",
		false,
		"transfer between testnet addresses",
	)
	cmd.Flags().BoolVarP(
		&mainnet,
		mainnetFlag,
		"m",
		false,
		"transfer between mainnet addresses",
	)
	cmd.Flags().BoolVarP(
		&send,
		sendFlag,
		"s",
		false,
		"send the transfer",
	)
	cmd.Flags().BoolVarP(
		&receive,
		receiveFlag,
		"g",
		false,
		"receive the transfer",
	)
	cmd.Flags().StringVarP(
		&sourceChain,
		sourceChainFlag,
		"c",
		"",
		fmt.Sprintf("source chain (%s %s %s)", OChain, AChain, DChain),
	)
	cmd.Flags().StringVarP(
		&destinationChain,
		destinationChainFlag,
		"d",
		"",
		fmt.Sprintf("destination chain (%s %s %s)", OChain, AChain, DChain),
	)
	cmd.Flags().StringVarP(
		&keyName,
		keyNameFlag,
		"k",
		"",
		"key associated to the sender or receiver address",
	)
	cmd.Flags().Uint32VarP(
		&ledgerIndex,
		ledgerIndexFlag,
		"i",
		wrongLedgerIndexVal,
		"ledger index associated to the sender or receiver address",
	)
	cmd.Flags().Uint64VarP(
		&receiveRecoveryStep,
		receiveRecoveryStepFlag,
		"r",
		0,
		"receive step to use for multiple step transaction recovery",
	)
	cmd.Flags().StringVarP(
		&receiverAddrStr,
		receiverAddrFlag,
		"a",
		"",
		"receiver address",
	)
	cmd.Flags().Float64VarP(
		&amountFlt,
		amountFlag,
		"o",
		0,
		"amount to send or receive for O-Chain to O-Chain transfer (DIONE units)",
	)
	return cmd
}

func transferF(*cobra.Command, []string) error {
	if send && receive {
		return fmt.Errorf("only one of %s, %s flags should be selected", sendFlag, receiveFlag)
	}

	if keyName != "" && ledgerIndex != wrongLedgerIndexVal {
		return fmt.Errorf("only one between a keyname or a ledger index must be given")
	}

	var network models.Network
	switch {
	case local:
		network = models.LocalNetwork
	case testnet:
		network = models.TestnetNetwork
	case mainnet:
		network = models.MainnetNetwork
	default:
		networkStr, err := app.Prompt.CaptureList(
			"Network to use",
			[]string{models.Mainnet.String(), models.Testnet.String(), models.Local.String()},
		)
		if err != nil {
			return err
		}
		network = models.NetworkFromString(networkStr)
	}

	var err error

	if !send && !receive {
		option, err := app.Prompt.CaptureList(
			"Step of the transfer",
			[]string{"Send", "Receive"},
		)
		if err != nil {
			return err
		}
		if option == "Send" {
			send = true
		} else {
			receive = true
		}
	}

	// Prompt for source chain if not provided
	if sourceChain == "" {
		sourceChain, err = app.Prompt.CaptureList(
			"Source chain",
			[]string{OChain, AChain, DChain},
		)
		if err != nil {
			return err
		}
	} else if sourceChain != OChain && sourceChain != AChain && sourceChain != DChain {
		return fmt.Errorf("invalid source chain: %s (must be %s %s %s)", sourceChain, OChain, AChain, DChain)
	}

	// Prompt for destination chain if not provided
	if destinationChain == "" {
		destinationChain, err = app.Prompt.CaptureList(
			"Destination chain",
			[]string{OChain, AChain, DChain},
		)
		if err != nil {
			return err
		}
	} else if destinationChain != OChain && destinationChain != AChain && destinationChain != DChain {
		return fmt.Errorf("invalid destination chain: %s (must be %s %s %s)", destinationChain, OChain, AChain, DChain)
	}

	if sourceChain != "" && destinationChain != "" && (sourceChain == DChain && destinationChain == DChain) {
		return fmt.Errorf("transfer from %s to %s is not supported", sourceChain, destinationChain)
	}

	if receive {
		if sourceChain != "" && destinationChain != "" && (sourceChain == destinationChain && (sourceChain == DChain || sourceChain == AChain)) {
			return fmt.Errorf("receive from same chain for A-Chain or D-Chain is not supported")
		}
	}

	if keyName == "" && ledgerIndex == wrongLedgerIndexVal {
		var useLedger bool
		goalStr := ""
		if send {
			goalStr = " for the sender address"
		} else {
			goalStr = " for the receiver address"
		}
		useLedger, keyName, err = prompts.GetTestnetKeyOrLedger(app.Prompt, goalStr, app.GetKeyDir())
		if err != nil {
			return err
		}
		if useLedger {
			ledgerIndex, err = app.Prompt.CaptureUint32("Ledger index to use")
			if err != nil {
				return err
			}
		}
	}

	if amountFlt == 0 {
		var promptStr string
		if send {
			promptStr = "Amount to send (DIONE units)"
		}

		if receive &&  sourceChain == OChain && destinationChain == OChain {
			promptStr = "Amount to receive (DIONE units)"
		}

		if promptStr != "" {
			amountFlt, err = app.Prompt.CaptureFloat(promptStr, func(v float64) error {
				if v <= 0 {
					return fmt.Errorf("value %f must be greater than zero", v)
				}
				return nil
			})
			if err != nil {
				return err
			}
		}
	}

	amount := uint64(amountFlt * float64(units.Dione))

	fee := network.GenesisParams().TxFee

	var kc keychain.Keychain
	if keyName != "" {
		keyPath := app.GetKeyPath(keyName)
		sk, err := key.LoadSoft(network.ID, keyPath)
		if err != nil {
			return err
		}
		kc = sk.KeyChain()
	} else {
		ledgerDevice, err := ledger.New()
		if err != nil {
			return err
		}
		ledgerIndices := []uint32{ledgerIndex}
		kc, err = keychain.NewLedgerKeychainFromIndices(ledgerDevice, ledgerIndices)
		if err != nil {
			return err
		}
	}

	var receiverAddr ids.ShortID
	if send {
		if receiverAddrStr == "" {
			receiverAddrStr, err = app.Prompt.CaptureOChainAddress("Receiver address", network)
			if err != nil {
				return err
			}
		}
		receiverAddr, err = address.ParseToID(receiverAddrStr)
		if err != nil {
			return err
		}
	} else {
		receiverAddr = kc.Addresses().List()[0]
		receiverAddrStr, err = address.Format("O", key.GetHRP(network.ID), receiverAddr[:])
		if err != nil {
			return err
		}
	}

	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("this operation is going to:")
	if send {
		addr := kc.Addresses().List()[0]
		addrStr, err := address.Format("O", key.GetHRP(network.ID), addr[:])
		if err != nil {
			return err
		}

		if sourceChain == destinationChain && addr == receiverAddr {
			return fmt.Errorf("sender addr is the same as receiver addr")
		}

		ux.Logger.PrintToUser("- send %.9f DIONE from %s to target address %s", float64(amount)/float64(units.Dione), addrStr, receiverAddrStr)
	} else {
		ux.Logger.PrintToUser("- receive DIONE at target address %s", receiverAddrStr)
	}
	ux.Logger.PrintToUser("")

	if !force {
		confStr := "Confirm transfer"
		conf, err := app.Prompt.CaptureNoYes(confStr)
		if err != nil {
			return err
		}
		if !conf {
			ux.Logger.PrintToUser("Cancelled")
			return nil
		}
	}

	to := secp256k1fx.OutputOwners{
		Threshold: 1,
		Addrs:     []ids.ShortID{receiverAddr},
	}

	if send {
		return sendF(amount, fee, kc, to, network)
	}
	
	return receiveF(amount, fee, kc, to, network)
}

func sendF(
	amount uint64,
	fee uint64,
	kc keychain.Keychain,
	to secp256k1fx.OutputOwners,
	network models.Network,
) error {
	if sourceChain == OChain && destinationChain == AChain {
		return oToA(amount, fee, kc, to, network)
	} else if sourceChain == OChain && destinationChain == DChain {
		return oToD(amount, fee, kc, to, network)
	} else if sourceChain == AChain && destinationChain == OChain {
		return aToO(amount, fee, kc, to, network)
	} else if sourceChain == AChain && destinationChain == DChain {
		return aToD(amount, fee, kc, to, network)
	} else if sourceChain == DChain && destinationChain == OChain {
		return dToO(amount, fee, kc, to, network)
	} else if sourceChain == DChain && destinationChain == AChain {
		return dToA(amount, fee, kc, to, network)
	} else if sourceChain == OChain && destinationChain == OChain {
		return oToO(amount, fee, kc, to, network)
	} else if sourceChain == AChain && destinationChain == AChain {
		return aToA(amount, fee, kc, to, network)
	} else {
		return fmt.Errorf("invalid source and destination chains: %s to %s", sourceChain, destinationChain)
	}
}

func receiveF(
	amount uint64,
	fee uint64,
	kc keychain.Keychain,
	to secp256k1fx.OutputOwners,
	network models.Network,
) error {
	if sourceChain == OChain && destinationChain == OChain {
		return receiveOFromO(amount, fee, kc, to, network)
	} else if sourceChain == OChain && destinationChain == AChain {
		return receiveAFromO(kc, to, network)
	} else if sourceChain == OChain && destinationChain == DChain {
		return receiveDFromO(kc, network)
	} else if sourceChain == DChain && destinationChain == OChain {
		return receiveOFromD(kc, to, network)
	} else if sourceChain == DChain && destinationChain == AChain {
		return receiveAFromD(kc, to, network)
	} else if sourceChain == AChain && destinationChain == OChain {
		return receiveOFromA(kc, to, network)
	} else if sourceChain == AChain && destinationChain == DChain {
		return receiveDFromA(kc, network)
	} else {
		return fmt.Errorf("invalid source and destination chains: %s to %s", sourceChain, destinationChain)
	}
}

func receiveAFromO(kc keychain.Keychain, to secp256k1fx.OutputOwners, network models.Network) error {
	wallet, err := primary.MakeWallet(
		context.Background(),
		&primary.WalletConfig{
			URI:           network.Endpoint,
			DIONEKeychain: kc,
			EthKeychain:   secp256k1fx.NewKeychain(),
		},
	)
	if err != nil {
		return err
	}
	ux.Logger.PrintToUser("Issuing ImportTx O -> A")

	if ledgerIndex != wrongLedgerIndexVal {
		ux.Logger.PrintToUser("*** Please sign ImportTx O -> A transaction on the ledger device *** ")
	}

	unsignedTx, err := wallet.A().Builder().NewImportTx(
		odygoconstants.OmegaChainID,
		&to,
	)
	if err != nil {
		return fmt.Errorf("error building tx: %w", err)
	}
	tx := alphatxs.Tx{Unsigned: unsignedTx}
	if err := wallet.A().Signer().Sign(context.Background(), &tx); err != nil {
		return fmt.Errorf("error signing tx: %w", err)
	}

	ctx, cancel := utils.GetAPIContext()
	defer cancel()
	err = wallet.A().IssueTx(
		&tx,
		common.WithContext(ctx),
	)
	if err != nil {
		if ctx.Err() != nil {
			err = fmt.Errorf("timeout issuing/verifying tx with ID %s: %w", tx.ID(), err)
		} else {
			err = fmt.Errorf("error issuing tx with ID %s: %w", tx.ID(), err)
		}
		return err
	}

	return nil
}

func receiveDFromO(kc keychain.Keychain, network models.Network) error {
	wallet, err := primary.MakeWallet(
		context.Background(),
		&primary.WalletConfig{
			URI:           network.Endpoint,
			DIONEKeychain: kc,
			EthKeychain:   secp256k1fx.NewKeychain(),
		},
	)
	if err != nil {
		return err
	}

	ethKc := secp256k1fx.NewKeychain()
	if secp256k1fxKc, ok := kc.(*secp256k1fx.Keychain); ok {
		ethKc = secp256k1fxKc
	}

	ethAddrs := ethKc.EthAddresses()
	ethAddrFromKc := ethAddrs.List()[0]

	ux.Logger.PrintToUser("Issuing ImportTx O -> D")

	if ledgerIndex != wrongLedgerIndexVal {
		ux.Logger.PrintToUser("*** Please sign ImportTx O -> D transaction on the ledger device *** ")
	}

	dWallet := wallet.D()

	_, err = dWallet.IssueImportTx(odygoconstants.OmegaChainID, ethAddrFromKc)

	if err != nil {
		return err
	}

	return nil
}

func receiveOFromD(kc keychain.Keychain, to secp256k1fx.OutputOwners, network models.Network) error {
	wallet, err := primary.MakeWallet(
		context.Background(),
		&primary.WalletConfig{
			URI:           network.Endpoint,
			DIONEKeychain: kc,
			EthKeychain:   secp256k1fx.NewKeychain(),
		},
	)
	if err != nil {
		return err
	}
	ux.Logger.PrintToUser("Issuing ImportTx D -> O")

	if ledgerIndex != wrongLedgerIndexVal {
		ux.Logger.PrintToUser("*** Please sign ImportTx D -> O transaction on the ledger device *** ")
	}

	_, err = subnet.IssueOFromChainImportTx(
		wallet.D().BlockchainID(),
		wallet,
		ledgerIndex != wrongLedgerIndexVal,
		true,
		&to,
	)
	if err != nil {
		return err
	}

	return nil
}

func receiveAFromD(kc keychain.Keychain, to secp256k1fx.OutputOwners, network models.Network) error {
	wallet, err := primary.MakeWallet(
		context.Background(),
		&primary.WalletConfig{
			URI:           network.Endpoint,
			DIONEKeychain: kc,
			EthKeychain:   secp256k1fx.NewKeychain(),
		},
	)
	if err != nil {
		return err
	}
	ux.Logger.PrintToUser("Issuing ImportTx D -> A")

	if ledgerIndex != wrongLedgerIndexVal {
		ux.Logger.PrintToUser("*** Please sign ImportTx D -> A transaction on the ledger device *** ")
	}

	unsignedTx, err := wallet.A().Builder().NewImportTx(
		wallet.D().BlockchainID(),
		&to,
	)
	if err != nil {
		return fmt.Errorf("error building tx: %w", err)
	}
	tx := alphatxs.Tx{Unsigned: unsignedTx}
	if err := wallet.A().Signer().Sign(context.Background(), &tx); err != nil {
		return fmt.Errorf("error signing tx: %w", err)
	}

	ctx, cancel := utils.GetAPIContext()
	defer cancel()
	err = wallet.A().IssueTx(
		&tx,
		common.WithContext(ctx),
	)
	if err != nil {
		if ctx.Err() != nil {
			err = fmt.Errorf("timeout issuing/verifying tx with ID %s: %w", tx.ID(), err)
		} else {
			err = fmt.Errorf("error issuing tx with ID %s: %w", tx.ID(), err)
		}
		return err
	}

	return nil
}

func receiveOFromA(kc keychain.Keychain, to secp256k1fx.OutputOwners, network models.Network) error {
	wallet, err := primary.MakeWallet(
		context.Background(),
		&primary.WalletConfig{
			URI:           network.Endpoint,
			DIONEKeychain: kc,
			EthKeychain:   secp256k1fx.NewKeychain(),
		},
	)
	if err != nil {
		ux.Logger.PrintToUser(logging.LightRed.Wrap(fmt.Sprintf("ERROR: restart from this step by using the same command with extra arguments: --%s %d", receiveRecoveryStepFlag, receiveRecoveryStep)))
		return err
	}
	ux.Logger.PrintToUser("Issuing ImportTx A -> O")

	if ledgerIndex != wrongLedgerIndexVal {
		ux.Logger.PrintToUser("*** Please sign ImportTx A -> O transaction on the ledger device *** ")
	}

	_, err = subnet.IssueOFromChainImportTx(
		wallet.A().BlockchainID(),
		wallet,
		ledgerIndex != wrongLedgerIndexVal,
		true,
		&to,
	)
	if err != nil {
		ux.Logger.PrintToUser(logging.LightRed.Wrap(fmt.Sprintf("ERROR: restart from this step by using the same command with extra arguments: --%s %d", receiveRecoveryStepFlag, receiveRecoveryStep)))
		return err
	}

	return nil
}

func receiveDFromA(kc keychain.Keychain, network models.Network) error {
	wallet, err := primary.MakeWallet(
		context.Background(),
		&primary.WalletConfig{
			URI:           network.Endpoint,
			DIONEKeychain: kc,
			EthKeychain:   secp256k1fx.NewKeychain(),
		},
	)
	if err != nil {
		return err
	}

	ux.Logger.PrintToUser("Issuing ImportTx A -> D")

	if ledgerIndex != wrongLedgerIndexVal {
		ux.Logger.PrintToUser("*** Please sign ImportTx A -> D transaction on the ledger device *** ")
	}

	dWallet := wallet.D()

	ethKc := secp256k1fx.NewKeychain()
	if secp256k1fxKc, ok := kc.(*secp256k1fx.Keychain); ok {
		ethKc = secp256k1fxKc
	}

	ethAddrs := ethKc.EthAddresses()
	ethAddrFromKc := ethAddrs.List()[0]

	_, err = dWallet.IssueImportTx(wallet.A().BlockchainID(), ethAddrFromKc)

	if err != nil {
		return err
	}

	return nil
}

func receiveOFromO(amount uint64, fee uint64, kc keychain.Keychain, to secp256k1fx.OutputOwners, network models.Network) error {
	if receiveRecoveryStep == 0 {
		err := receiveAFromO(kc, to, network)
		if err != nil {
			ux.Logger.PrintToUser(logging.LightRed.Wrap("ERROR: restart from this step by using the same command"))
			return err
		}

		time.Sleep(5 * time.Second)
		receiveRecoveryStep++
	}
	if receiveRecoveryStep == 1 {
		wallet, err := primary.MakeWallet(
			context.Background(),
			&primary.WalletConfig{
				URI:           network.Endpoint,
				DIONEKeychain: kc,
				EthKeychain:   secp256k1fx.NewKeychain(),
			},
		)
		if err != nil {
			ux.Logger.PrintToUser(logging.LightRed.Wrap(fmt.Sprintf("ERROR: restart from this step by using the same command with extra arguments: --%s %d", receiveRecoveryStepFlag, receiveRecoveryStep)))
			return err
		}
		ux.Logger.PrintToUser("Issuing ExportTx A -> O")
		_, err = subnet.IssueAToOExportTx(
			wallet,
			ledgerIndex != wrongLedgerIndexVal,
			true,
			wallet.O().DIONEAssetID(),
			amount+fee*1,
			&to,
		)
		if err != nil {
			ux.Logger.PrintToUser(logging.LightRed.Wrap(fmt.Sprintf("ERROR: restart from this step by using the same command with extra arguments: --%s %d", receiveRecoveryStepFlag, receiveRecoveryStep)))
			return err
		}
		time.Sleep(5 * time.Second)
		receiveRecoveryStep++
	}
	if receiveRecoveryStep == 2 {
		err := receiveOFromA(kc, to, network)
		if err != nil {
			ux.Logger.PrintToUser(logging.LightRed.Wrap(fmt.Sprintf("ERROR: restart from this step by using the same command with extra arguments: --%s %d", receiveRecoveryStepFlag, receiveRecoveryStep)))
			return err
		}
	}

	return nil
}

func oToO(amount uint64, fee uint64, kc keychain.Keychain, to secp256k1fx.OutputOwners, network models.Network) error {
	return oToA(amount, fee*3, kc, to, network)
}

func aToA(amount uint64, fee uint64, kc keychain.Keychain, to secp256k1fx.OutputOwners, network models.Network) error {
	wallet, err := primary.MakeWallet(
		context.Background(),
		&primary.WalletConfig{
			URI:           network.Endpoint,
			DIONEKeychain: kc,
			EthKeychain:   secp256k1fx.NewKeychain(),
		},
	)
	if err != nil {
		return err
	}

	if ledgerIndex != wrongLedgerIndexVal {
		ux.Logger.PrintToUser("*** Please sign 'Base Tx / A to A Chain' transaction on the ledger device *** ")
	}

	// For A-chain, we can use export/import pattern
	output := &dione.TransferableOutput{
		Asset: dione.Asset{ID: wallet.A().DIONEAssetID()},
		Out: &secp256k1fx.TransferOutput{
			Amt:          amount + fee,
			OutputOwners: to,
		},
	}
	outputs := []*dione.TransferableOutput{output}

	unsignedTx, err := wallet.A().Builder().NewBaseTx(
		outputs,
	)
	if err != nil {
		return fmt.Errorf("error building base tx: %w", err)
	}
	tx := alphatxs.Tx{Unsigned: unsignedTx}
	if err := wallet.A().Signer().Sign(context.Background(), &tx); err != nil {
		return fmt.Errorf("error signing tx: %w", err)
	}

	ctx, cancel := utils.GetAPIContext()
	defer cancel()
	if err := wallet.A().IssueTx(&tx, common.WithContext(ctx)); err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("timeout issuing tx with ID %s: %w", tx.ID(), err)
		}
		return fmt.Errorf("error issuing tx with ID %s: %w", tx.ID(), err)
	}

	ux.Logger.PrintToUser("Export transaction issued. Transaction ID: %s", tx.ID())
	ux.Logger.PrintToUser("Waiting for transaction to be accepted...")
	time.Sleep(2 * time.Second)

	return nil
}

func oToA(amount uint64, fee uint64, kc keychain.Keychain, to secp256k1fx.OutputOwners, network models.Network) error {
	wallet, err := primary.MakeWallet(
		context.Background(),
		&primary.WalletConfig{
			URI:           network.Endpoint,
			DIONEKeychain: kc,
			EthKeychain:   secp256k1fx.NewKeychain(),
		},
	)
	if err != nil {
		return err
	}
	output := &dione.TransferableOutput{
		Asset: dione.Asset{ID: wallet.O().DIONEAssetID()},
		Out: &secp256k1fx.TransferOutput{
			Amt:          amount + fee,
			OutputOwners: to,
		},
	}
	outputs := []*dione.TransferableOutput{output}
	ux.Logger.PrintToUser("Issuing ExportTx O -> A")

	if ledgerIndex != wrongLedgerIndexVal {
		ux.Logger.PrintToUser("*** Please sign 'Export Tx / O to A Chain' transaction on the ledger device *** ")
	}
	unsignedTx, err := wallet.O().Builder().NewExportTx(
		wallet.A().BlockchainID(),
		outputs,
	)
	if err != nil {
		return fmt.Errorf("error building tx: %w", err)
	}
	tx := txs.Tx{Unsigned: unsignedTx}
	if err := wallet.O().Signer().Sign(context.Background(), &tx); err != nil {
		return fmt.Errorf("error signing tx: %w", err)
	}

	ctx, cancel := utils.GetAPIContext()
	defer cancel()
	err = wallet.O().IssueTx(
		&tx,
		common.WithContext(ctx),
	)
	if err != nil {
		if ctx.Err() != nil {
			err = fmt.Errorf("timeout issuing/verifying tx with ID %s: %w", tx.ID(), err)
		} else {
			err = fmt.Errorf("error issuing tx with ID %s: %w", tx.ID(), err)
		}
		return err
	}
	return nil
}

func oToD(amount uint64, fee uint64, kc keychain.Keychain, to secp256k1fx.OutputOwners, network models.Network) error {
	if ledgerIndex != wrongLedgerIndexVal {
		ux.Logger.PrintToUser("*** Please sign 'Export Tx / O to D Chain' transaction on the ledger device *** ")
	}

	wallet, err := primary.MakeWallet(
		context.Background(),
		&primary.WalletConfig{
			URI:           network.Endpoint,
			DIONEKeychain: kc,
			EthKeychain:   secp256k1fx.NewKeychain(),
		},
	)
	if err != nil {
		return err
	}
	output := &dione.TransferableOutput{
		Asset: dione.Asset{ID: wallet.O().DIONEAssetID()},
		Out: &secp256k1fx.TransferOutput{
			Amt:          amount + fee,
			OutputOwners: to,
		},
	}
	outputs := []*dione.TransferableOutput{output}
	ux.Logger.PrintToUser("Issuing ExportTx O -> D")

	if ledgerIndex != wrongLedgerIndexVal {
		ux.Logger.PrintToUser("*** Please sign 'Export Tx / O to D Chain' transaction on the ledger device *** ")
	}
	unsignedTx, err := wallet.O().Builder().NewExportTx(
		wallet.D().BlockchainID(),
		outputs,
	)
	if err != nil {
		return fmt.Errorf("error building tx: %w", err)
	}
	tx := txs.Tx{Unsigned: unsignedTx}
	if err := wallet.O().Signer().Sign(context.Background(), &tx); err != nil {
		return fmt.Errorf("error signing tx: %w", err)
	}

	ctx, cancel := utils.GetAPIContext()
	defer cancel()
	err = wallet.O().IssueTx(
		&tx,
		common.WithContext(ctx),
	)
	if err != nil {
		if ctx.Err() != nil {
			err = fmt.Errorf("timeout issuing/verifying tx with ID %s: %w", tx.ID(), err)
		} else {
			err = fmt.Errorf("error issuing tx with ID %s: %w", tx.ID(), err)
		}
		return err
	}
	return nil
}

func aToO(amount uint64, fee uint64, kc keychain.Keychain, to secp256k1fx.OutputOwners, network models.Network) error {
	if ledgerIndex != wrongLedgerIndexVal {
		ux.Logger.PrintToUser("*** Please sign 'Export Tx / A to O Chain' transaction on the ledger device *** ")
	}

	wallet, err := primary.MakeWallet(
		context.Background(),
		&primary.WalletConfig{
			URI:           network.Endpoint,
			DIONEKeychain: kc,
			EthKeychain:   secp256k1fx.NewKeychain(),
		},
	)
	if err != nil {
		ux.Logger.PrintToUser(logging.LightRed.Wrap(fmt.Sprintf("ERROR: restart from this step by using the same command with extra arguments: --%s %d", receiveRecoveryStepFlag, receiveRecoveryStep)))
		return err
	}
	ux.Logger.PrintToUser("Issuing ExportTx A -> O")
	_, err = subnet.IssueAToOExportTx(
		wallet,
		ledgerIndex != wrongLedgerIndexVal,
		true,
		wallet.O().DIONEAssetID(),
		amount+fee*1,
		&to,
	)
	if err != nil {
		ux.Logger.PrintToUser(logging.LightRed.Wrap(fmt.Sprintf("ERROR: restart from this step by using the same command with extra arguments: --%s %d", receiveRecoveryStepFlag, receiveRecoveryStep)))
		return err
	}

	return nil
}

func aToD(amount uint64, fee uint64, kc keychain.Keychain, to secp256k1fx.OutputOwners, network models.Network) error {
	if ledgerIndex != wrongLedgerIndexVal {
		ux.Logger.PrintToUser("*** Please sign 'Export Tx / A to D Chain' transaction on the ledger device *** ")
	}

	wallet, err := primary.MakeWallet(
		context.Background(),
		&primary.WalletConfig{
			URI:           network.Endpoint,
			DIONEKeychain: kc,
			EthKeychain:   secp256k1fx.NewKeychain(),
		},
	)
	if err != nil {
		ux.Logger.PrintToUser(logging.LightRed.Wrap(fmt.Sprintf("ERROR: restart from this step by using the same command with extra arguments: --%s %d", receiveRecoveryStepFlag, receiveRecoveryStep)))
		return err
	}
	ux.Logger.PrintToUser("Issuing ExportTx A -> D")

	unsignedTx, err := wallet.A().Builder().NewExportTx(
		odygoconstants.OmegaChainID,
		[]*dione.TransferableOutput{
			{
				Asset: dione.Asset{
					ID: wallet.A().DIONEAssetID(),
				},
				Out: &secp256k1fx.TransferOutput{
					Amt:          amount,
					OutputOwners: to,
				},
			},
		},
	)
	if err != nil {
		return fmt.Errorf("error building tx: %w", err)
	}
	tx := alphatxs.Tx{Unsigned: unsignedTx}
	if err := wallet.A().Signer().Sign(context.Background(), &tx); err != nil {
		return fmt.Errorf("error signing tx: %w", err)
	}
	ctx, cancel := utils.GetAPIContext()
	defer cancel()
	err = wallet.A().IssueTx(
		&tx,
		common.WithContext(ctx),
	)
	if err != nil {
		if ctx.Err() != nil {
			err = fmt.Errorf("timeout issuing/verifying tx with ID %s: %w", tx.ID(), err)
		} else {
			err = fmt.Errorf("error issuing tx with ID %s: %w", tx.ID(), err)
		}
		return err
	}

	return nil
}

func dToO(amount uint64, fee uint64, kc keychain.Keychain, to secp256k1fx.OutputOwners, network models.Network) error {
	if ledgerIndex != wrongLedgerIndexVal {
		ux.Logger.PrintToUser("*** Please sign 'Export Tx / D to O Chain' transaction on the ledger device *** ")
	}

	ethKc := secp256k1fx.NewKeychain()
	if secp256k1fxKc, ok := kc.(*secp256k1fx.Keychain); ok {
		ethKc = secp256k1fxKc
	}

	wallet, err := primary.MakeWallet(context.Background(), &primary.WalletConfig{
		URI:           network.Endpoint,
		DIONEKeychain: kc,
		EthKeychain:   ethKc,
	})

	if err != nil {
		return fmt.Errorf("failed to initialize wallet: %w", err)
	}

	dWallet := wallet.D()

	ux.Logger.PrintToUser("Issuing ExportTx D -> O")

	exportTx, err := dWallet.IssueExportTx(
		odygoconstants.OmegaChainID,
		[]*secp256k1fx.TransferOutput{{
			Amt:          amount + fee*1,
			OutputOwners: to,
		}},
	)

	if err != nil {
		return fmt.Errorf("failed to issue export transaction: %w", err)
	}

	ux.Logger.PrintToUser("Export transaction issued. Transaction ID: %s", exportTx.ID())
	return nil
}

func dToA(amount uint64, fee uint64, kc keychain.Keychain, to secp256k1fx.OutputOwners, network models.Network) error {
	if ledgerIndex != wrongLedgerIndexVal {
		ux.Logger.PrintToUser("*** Please sign 'Export Tx / D to A Chain' transaction on the ledger device *** ")
	}

	ethKc := secp256k1fx.NewKeychain()
	if secp256k1fxKc, ok := kc.(*secp256k1fx.Keychain); ok {
		ethKc = secp256k1fxKc
	}

	wallet, err := primary.MakeWallet(context.Background(), &primary.WalletConfig{
		URI:           network.Endpoint,
		DIONEKeychain: kc,
		EthKeychain:   ethKc,
	})
	if err != nil {
		return fmt.Errorf("failed to initialize wallet: %w", err)
	}

	dWallet := wallet.D()

	ux.Logger.PrintToUser("Issuing ExportTx D -> A")

	exportTx, err := dWallet.IssueExportTx(
		wallet.A().BlockchainID(),
		[]*secp256k1fx.TransferOutput{{
			Amt:          amount + fee*1,
			OutputOwners: to,
		}},
	)

	if err != nil {
		return fmt.Errorf("failed to issue export transaction: %w", err)
	}

	ux.Logger.PrintToUser("Export transaction issued. Transaction ID: %s", exportTx.ID())

	return nil
}
