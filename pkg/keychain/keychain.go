// Copyright (C) 2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package keychain

import (
	"errors"
	"fmt"

	"github.com/DioneProtocol/odyssey-cli/cmd/flags"
	"github.com/DioneProtocol/odyssey-cli/pkg/application"
	"github.com/DioneProtocol/odyssey-cli/pkg/key"
	"github.com/DioneProtocol/odyssey-cli/pkg/models"
	"github.com/DioneProtocol/odyssey-cli/pkg/prompts"
	"github.com/DioneProtocol/odyssey-cli/pkg/utils"
	"github.com/DioneProtocol/odyssey-cli/pkg/ux"
	"github.com/DioneProtocol/odysseygo/ids"
	"github.com/DioneProtocol/odysseygo/utils/crypto/keychain"
	"github.com/DioneProtocol/odysseygo/utils/crypto/ledger"
	"github.com/DioneProtocol/odysseygo/utils/formatting/address"
	"github.com/DioneProtocol/odysseygo/utils/logging"
	"github.com/DioneProtocol/odysseygo/utils/set"
	"github.com/DioneProtocol/odysseygo/utils/units"
	"github.com/DioneProtocol/odysseygo/vms/omegavm"
)

const (
	numLedgerIndicesToSearch           = 1000
	numLedgerIndicesToSearchForBalance = 100
)

var (
	ErrMutuallyExclusiveKeySource = errors.New("key source flags --key, --ewoq, --ledger/--ledger-addrs are mutually exclusive")
	ErrStoredKeyOrEwoqOnMainnet   = errors.New("key sources --key, --ewoq are not available for mainnet operations")
	ErrNonEwoqKeyOnDevnet         = errors.New("key source --ewoq is the only one available for devnet operations")
	ErrEwoqKeyOnTestnet           = errors.New("key source --ewoq is not available for testnet operations")
)

type Keychain struct {
	Network       models.Network
	Keychain      keychain.Keychain
	Ledger        keychain.Ledger
	UsesLedger    bool
	LedgerIndices []uint32
}

func NewKeychain(network models.Network, keychain keychain.Keychain, ledger keychain.Ledger, ledgerIndices []uint32) *Keychain {
	usesLedger := len(ledgerIndices) > 0
	return &Keychain{
		Network:       network,
		Keychain:      keychain,
		Ledger:        ledger,
		UsesLedger:    usesLedger,
		LedgerIndices: ledgerIndices,
	}
}

func (kc *Keychain) HasOnlyOneKey() bool {
	return len(kc.Keychain.Addresses()) == 1
}

func (kc *Keychain) Addresses() set.Set[ids.ShortID] {
	return kc.Keychain.Addresses()
}

func (kc *Keychain) OChainFormattedStrAddresses() ([]string, error) {
	addrs := kc.Addresses().List()
	if len(addrs) == 0 {
		return nil, fmt.Errorf("no addresses in keychain")
	}
	hrp := key.GetHRP(kc.Network.ID)
	addrsStr := []string{}
	for _, addr := range addrs {
		addrStr, err := address.Format("O", hrp, addr[:])
		if err != nil {
			return nil, err
		}
		addrsStr = append(addrsStr, addrStr)
	}

	return addrsStr, nil
}

func (kc *Keychain) AddAddresses(addresses []string) error {
	if kc.UsesLedger {
		prevNumIndices := len(kc.LedgerIndices)
		ledgerIndicesAux, err := getLedgerIndices(kc.Ledger, addresses)
		if err != nil {
			return err
		}
		kc.LedgerIndices = append(kc.LedgerIndices, ledgerIndicesAux...)
		ledgerIndicesSet := set.Set[uint32]{}
		ledgerIndicesSet.Add(kc.LedgerIndices...)
		kc.LedgerIndices = ledgerIndicesSet.List()
		utils.SortUint32(kc.LedgerIndices)
		if len(kc.LedgerIndices) != prevNumIndices {
			if err := showLedgerAddresses(kc.Network, kc.Ledger, kc.LedgerIndices); err != nil {
				return err
			}
		}
		odygoKc, err := keychain.NewLedgerKeychainFromIndices(kc.Ledger, kc.LedgerIndices)
		if err != nil {
			return err
		}
		kc.Keychain = odygoKc
	}
	return nil
}

func GetKeychainFromCmdLineFlags(
	app *application.Odyssey,
	keychainGoal string,
	network models.Network,
	keyName string,
	useEwoq bool,
	useLedger bool,
	ledgerAddresses []string,
	requiredFunds uint64,
) (*Keychain, error) {
	// set ledger usage flag if ledger addresses are given
	if len(ledgerAddresses) > 0 {
		useLedger = true
	}

	// check mutually exclusive flags
	if !flags.EnsureMutuallyExclusive([]bool{useLedger, useEwoq, keyName != ""}) {
		return nil, ErrMutuallyExclusiveKeySource
	}

	switch {
	case network.Kind == models.Devnet:
		// going to just use ewoq atm
		useEwoq = true
		if keyName != "" || useLedger {
			return nil, ErrNonEwoqKeyOnDevnet
		}
	case network.Kind == models.Testnet:
		if useEwoq {
			return nil, ErrEwoqKeyOnTestnet
		}
		// prompt the user if no key source was provided
		if !useLedger && keyName == "" {
			var err error
			useLedger, keyName, err = prompts.GetTestnetKeyOrLedger(app.Prompt, keychainGoal, app.GetKeyDir())
			if err != nil {
				return nil, err
			}
		}
	case network.Kind == models.Mainnet:
		// mainnet requires ledger usage
		if keyName != "" || useEwoq {
			return nil, ErrStoredKeyOrEwoqOnMainnet
		}
		useLedger = true
	}

	network.HandlePublicNetworkSimulation()

	// get keychain accessor
	return GetKeychain(app, useEwoq, useLedger, ledgerAddresses, keyName, network, requiredFunds)
}

func GetKeychain(
	app *application.Odyssey,
	useEwoq bool,
	useLedger bool,
	ledgerAddresses []string,
	keyName string,
	network models.Network,
	requiredFunds uint64,
) (*Keychain, error) {
	// get keychain accessor
	if useLedger {
		ledgerDevice, err := ledger.New()
		if err != nil {
			return nil, err
		}
		// always have index 0, for change
		ledgerIndices := []uint32{0}
		if requiredFunds > 0 {
			ledgerIndicesAux, err := searchForFundedLedgerIndices(network, ledgerDevice, requiredFunds)
			if err != nil {
				return nil, err
			}
			ledgerIndices = append(ledgerIndices, ledgerIndicesAux...)
		}
		if len(ledgerAddresses) > 0 {
			ledgerIndicesAux, err := getLedgerIndices(ledgerDevice, ledgerAddresses)
			if err != nil {
				return nil, err
			}
			ledgerIndices = append(ledgerIndices, ledgerIndicesAux...)
		}
		ledgerIndicesSet := set.Set[uint32]{}
		ledgerIndicesSet.Add(ledgerIndices...)
		ledgerIndices = ledgerIndicesSet.List()
		utils.SortUint32(ledgerIndices)
		if err := showLedgerAddresses(network, ledgerDevice, ledgerIndices); err != nil {
			return nil, err
		}
		kc, err := keychain.NewLedgerKeychainFromIndices(ledgerDevice, ledgerIndices)
		if err != nil {
			return nil, err
		}
		return NewKeychain(network, kc, ledgerDevice, ledgerIndices), nil
	}
	if useEwoq {
		sf, err := key.LoadEwoq(network.ID)
		if err != nil {
			return nil, err
		}
		kc := sf.KeyChain()
		return NewKeychain(network, kc, nil, nil), nil
	}
	sf, err := key.LoadSoft(network.ID, app.GetKeyPath(keyName))
	if err != nil {
		return nil, err
	}
	kc := sf.KeyChain()
	return NewKeychain(network, kc, nil, nil), nil
}

func getLedgerIndices(ledgerDevice keychain.Ledger, addressesStr []string) ([]uint32, error) {
	addresses, err := address.ParseToIDs(addressesStr)
	if err != nil {
		return []uint32{}, fmt.Errorf("failure parsing ledger addresses: %w", err)
	}
	// maps the indices of addresses to their corresponding ledger indices
	indexMap := map[int]uint32{}
	// for all ledger indices to search for, find if the ledger address belongs to the input
	// addresses and, if so, add the index pair to indexMap, breaking the loop if
	// all addresses were found
	for ledgerIndex := uint32(0); ledgerIndex < numLedgerIndicesToSearch; ledgerIndex++ {
		ledgerAddress, err := ledgerDevice.Addresses([]uint32{ledgerIndex})
		if err != nil {
			return []uint32{}, err
		}
		for addressesIndex, addr := range addresses {
			if addr == ledgerAddress[0] {
				ux.Logger.PrintToUser("  Found index %d for address %s", ledgerIndex, addressesStr[addressesIndex])
				indexMap[addressesIndex] = ledgerIndex
			}
		}
		if len(indexMap) == len(addresses) {
			break
		}
	}
	// create ledgerIndices from indexMap
	ledgerIndices := []uint32{}
	for addressesIndex := range addresses {
		ledgerIndex, ok := indexMap[addressesIndex]
		if !ok {
			continue
		}
		ledgerIndices = append(ledgerIndices, ledgerIndex)
	}
	return ledgerIndices, nil
}

// search for a set of indices that pay a given amount
func searchForFundedLedgerIndices(network models.Network, ledgerDevice keychain.Ledger, amount uint64) ([]uint32, error) {
	ux.Logger.PrintToUser("Looking for ledger indices to pay for %.9f DIONE...", float64(amount)/float64(units.Dione))
	oClient := omegavm.NewClient(network.Endpoint)
	totalBalance := uint64(0)
	ledgerIndices := []uint32{}
	for ledgerIndex := uint32(0); ledgerIndex < numLedgerIndicesToSearchForBalance; ledgerIndex++ {
		ledgerAddress, err := ledgerDevice.Addresses([]uint32{ledgerIndex})
		if err != nil {
			return []uint32{}, err
		}
		ctx, cancel := utils.GetAPIContext()
		resp, err := oClient.GetBalance(ctx, ledgerAddress)
		cancel()
		if err != nil {
			return nil, err
		}
		if resp.Balance > 0 {
			ux.Logger.PrintToUser("  Found index %d with %.9f DIONE", ledgerIndex, float64(resp.Balance)/float64(units.Dione))
			totalBalance += uint64(resp.Balance)
			ledgerIndices = append(ledgerIndices, ledgerIndex)
		}
		if totalBalance >= amount {
			break
		}
	}
	if totalBalance < amount {
		ux.Logger.PrintToUser(logging.Yellow.Wrap("Not enough funds in the first %d indices of Ledger"), numLedgerIndicesToSearchForBalance)
		return nil, fmt.Errorf("not enough funds on ledger")
	}
	return ledgerIndices, nil
}

func showLedgerAddresses(network models.Network, ledgerDevice keychain.Ledger, ledgerIndices []uint32) error {
	// get formatted addresses for ux
	addresses, err := ledgerDevice.Addresses(ledgerIndices)
	if err != nil {
		return err
	}
	addrStrs := []string{}
	for _, addr := range addresses {
		addrStr, err := address.Format("O", key.GetHRP(network.ID), addr[:])
		if err != nil {
			return err
		}
		addrStrs = append(addrStrs, addrStr)
	}
	ux.Logger.PrintToUser(logging.Yellow.Wrap("Ledger addresses: "))
	for _, addrStr := range addrStrs {
		ux.Logger.PrintToUser(logging.Yellow.Wrap(fmt.Sprintf("  %s", addrStr)))
	}
	return nil
}
