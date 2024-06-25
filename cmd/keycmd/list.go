// Copyright (C) 2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package keycmd

import (
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"

	"github.com/DioneProtocol/coreth/ethclient"
	"github.com/DioneProtocol/odyssey-cli/pkg/constants"
	"github.com/DioneProtocol/odyssey-cli/pkg/key"
	"github.com/DioneProtocol/odyssey-cli/pkg/models"
	"github.com/DioneProtocol/odyssey-cli/pkg/utils"
	"github.com/DioneProtocol/odysseygo/ids"
	ledger "github.com/DioneProtocol/odysseygo/utils/crypto/ledger"
	"github.com/DioneProtocol/odysseygo/utils/formatting/address"
	"github.com/DioneProtocol/odysseygo/utils/units"
	"github.com/DioneProtocol/odysseygo/vms/omegavm"
	"github.com/ethereum/go-ethereum/common"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

const (
	localFlag         = "local"
	testnetFlag       = "testnet"
	mainnetFlag       = "mainnet"
	allFlag           = "all-networks"
	dchainFlag        = "dchain"
	ledgerIndicesFlag = "ledger"
	useNanoDioneFlag  = "use-nano-dione"
)

var (
	local         bool
	testnet       bool
	mainnet       bool
	all           bool
	dchain        bool
	useNanoDione  bool
	ledgerIndices []uint
)

// odyssey subnet list
func newListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List stored signing keys or ledger addresses",
		Long: `The key list command prints information for all stored signing
keys or for the ledger addresses associated to certain indices.`,
		RunE:         listKeys,
		SilenceUsage: true,
	}
	cmd.Flags().BoolVarP(
		&local,
		localFlag,
		"l",
		false,
		"list local network addresses",
	)
	cmd.Flags().BoolVarP(
		&testnet,
		testnetFlag,
		"t",
		false,
		"list testnet network addresses",
	)
	cmd.Flags().BoolVarP(
		&mainnet,
		mainnetFlag,
		"m",
		false,
		"list mainnet network addresses",
	)
	cmd.Flags().BoolVarP(
		&all,
		allFlag,
		"a",
		false,
		"list all network addresses",
	)
	cmd.Flags().BoolVarP(
		&dchain,
		dchainFlag,
		"d",
		true,
		"list D-Chain addresses",
	)
	cmd.Flags().BoolVarP(
		&useNanoDione,
		useNanoDioneFlag,
		"n",
		false,
		"use nano Dione for balances",
	)
	cmd.Flags().UintSliceVarP(
		&ledgerIndices,
		ledgerIndicesFlag,
		"g",
		[]uint{},
		"list ledger addresses for the given indices",
	)
	return cmd
}

func getClients(networks []models.Network, dchain bool) (
	map[models.Network]omegavm.Client,
	map[models.Network]ethclient.Client,
	error,
) {
	var err error
	oClients := map[models.Network]omegavm.Client{}
	dClients := map[models.Network]ethclient.Client{}
	for _, network := range networks {
		oClients[network] = omegavm.NewClient(network.Endpoint)
		if dchain {
			dClients[network], err = ethclient.Dial(network.DChainEndpoint())
			if err != nil {
				return nil, nil, err
			}
		}
	}
	return oClients, dClients, nil
}

type addressInfo struct {
	kind    string
	name    string
	chain   string
	address string
	balance string
	network string
}

func listKeys(*cobra.Command, []string) error {
	var addrInfos []addressInfo
	networks := []models.Network{}
	if local || all {
		networks = append(networks, models.LocalNetwork)
	}
	if testnet || all {
		networks = append(networks, models.TestnetNetwork)
	}
	if mainnet || all {
		networks = append(networks, models.MainnetNetwork)
	}
	if len(networks) == 0 {
		// no flag was set, prompt user
		networkStr, err := app.Prompt.CaptureList(
			"Choose network for which to list addresses",
			[]string{models.Mainnet.String(), models.Testnet.String(), models.Local.String()},
		)
		if err != nil {
			return err
		}
		network := models.NetworkFromString(networkStr)
		networks = append(networks, network)
	}
	queryLedger := len(ledgerIndices) > 0
	if queryLedger {
		dchain = false
	}
	oClients, dClients, err := getClients(networks, dchain)
	if err != nil {
		return err
	}
	if queryLedger {
		ledgerIndicesU32 := []uint32{}
		for _, index := range ledgerIndices {
			ledgerIndicesU32 = append(ledgerIndicesU32, uint32(index))
		}
		addrInfos, err = getLedgerIndicesInfo(oClients, ledgerIndicesU32, networks)
		if err != nil {
			return err
		}
	} else {
		addrInfos, err = getStoredKeysInfo(oClients, dClients, networks, dchain)
		if err != nil {
			return err
		}
	}
	printAddrInfos(addrInfos)
	return nil
}

func getStoredKeysInfo(
	oClients map[models.Network]omegavm.Client,
	dClients map[models.Network]ethclient.Client,
	networks []models.Network,
	dchain bool,
) ([]addressInfo, error) {
	files, err := os.ReadDir(app.GetKeyDir())
	if err != nil {
		return nil, err
	}
	keyPaths := make([]string, len(files))
	for i, f := range files {
		if strings.HasSuffix(f.Name(), constants.KeySuffix) {
			keyPaths[i] = filepath.Join(app.GetKeyDir(), f.Name())
		}
	}
	addrInfos := []addressInfo{}
	for _, keyPath := range keyPaths {
		keyAddrInfos, err := getStoredKeyInfo(oClients, dClients, networks, keyPath, dchain)
		if err != nil {
			return nil, err
		}
		addrInfos = append(addrInfos, keyAddrInfos...)
	}
	return addrInfos, nil
}

func getStoredKeyInfo(
	oClients map[models.Network]omegavm.Client,
	dClients map[models.Network]ethclient.Client,
	networks []models.Network,
	keyPath string,
	dchain bool,
) ([]addressInfo, error) {
	addrInfos := []addressInfo{}
	for _, network := range networks {
		keyName := strings.TrimSuffix(filepath.Base(keyPath), constants.KeySuffix)
		sk, err := key.LoadSoft(network.ID, keyPath)
		if err != nil {
			return nil, err
		}
		if dchain {
			dChainAddr := sk.D()
			addrInfo, err := getDChainAddrInfo(dClients, network, dChainAddr, "stored", keyName)
			if err != nil {
				return nil, err
			}
			addrInfos = append(addrInfos, addrInfo)
		}
		oChainAddrs := sk.O()
		for _, oChainAddr := range oChainAddrs {
			addrInfo, err := getOChainAddrInfo(oClients, network, oChainAddr, "stored", keyName)
			if err != nil {
				return nil, err
			}
			addrInfos = append(addrInfos, addrInfo)
		}
	}
	return addrInfos, nil
}

func getLedgerIndicesInfo(
	oClients map[models.Network]omegavm.Client,
	ledgerIndices []uint32,
	networks []models.Network,
) ([]addressInfo, error) {
	ledgerDevice, err := ledger.New()
	if err != nil {
		return nil, err
	}
	addresses, err := ledgerDevice.Addresses(ledgerIndices)
	if err != nil {
		return nil, err
	}
	if len(addresses) != len(ledgerIndices) {
		return nil, fmt.Errorf("derived addresses length %d differs from expected %d", len(addresses), len(ledgerIndices))
	}
	addrInfos := []addressInfo{}
	for i, index := range ledgerIndices {
		addr := addresses[i]
		ledgerAddrInfos, err := getLedgerIndexInfo(oClients, index, networks, addr)
		if err != nil {
			return []addressInfo{}, err
		}
		addrInfos = append(addrInfos, ledgerAddrInfos...)
	}
	return addrInfos, nil
}

func getLedgerIndexInfo(
	oClients map[models.Network]omegavm.Client,
	index uint32,
	networks []models.Network,
	addr ids.ShortID,
) ([]addressInfo, error) {
	addrInfos := []addressInfo{}
	for _, network := range networks {
		oChainAddr, err := address.Format("O", key.GetHRP(network.ID), addr[:])
		if err != nil {
			return nil, err
		}
		addrInfo, err := getOChainAddrInfo(
			oClients,
			network,
			oChainAddr,
			"ledger",
			fmt.Sprintf("index %d", index),
		)
		if err != nil {
			return nil, err
		}
		addrInfos = append(addrInfos, addrInfo)
	}
	return addrInfos, nil
}

func getOChainAddrInfo(
	oClients map[models.Network]omegavm.Client,
	network models.Network,
	oChainAddr string,
	kind string,
	name string,
) (addressInfo, error) {
	balance, err := getOChainBalanceStr(oClients[network], oChainAddr)
	if err != nil {
		// just ignore local network errors
		if network.Kind != models.Local {
			return addressInfo{}, err
		}
	}
	return addressInfo{
		kind:    kind,
		name:    name,
		chain:   "O-Chain (Bech32 format)",
		address: oChainAddr,
		balance: balance,
		network: network.Name(),
	}, nil
}

func getDChainAddrInfo(
	dClients map[models.Network]ethclient.Client,
	network models.Network,
	dChainAddr string,
	kind string,
	name string,
) (addressInfo, error) {
	dChainBalance, err := getDChainBalanceStr(dClients[network], dChainAddr)
	if err != nil {
		// just ignore local network errors
		if network.Kind != models.Local {
			return addressInfo{}, err
		}
	}
	return addressInfo{
		kind:    kind,
		name:    name,
		chain:   "D-Chain (Ethereum hex format)",
		address: dChainAddr,
		balance: dChainBalance,
		network: network.Name(),
	}, nil
}

func printAddrInfos(addrInfos []addressInfo) {
	header := []string{"Kind", "Name", "Chain", "Address", "Balance", "Network"}
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(header)
	table.SetRowLine(true)
	table.SetAutoMergeCellsByColumnIndex([]int{0, 1, 2})
	for _, addrInfo := range addrInfos {
		table.Append([]string{
			addrInfo.kind,
			addrInfo.name,
			addrInfo.chain,
			addrInfo.address,
			addrInfo.balance,
			addrInfo.network,
		})
	}
	table.Render()
}

func getDChainBalanceStr(dClient ethclient.Client, addrStr string) (string, error) {
	addr := common.HexToAddress(addrStr)
	ctx, cancel := utils.GetAPIContext()
	balance, err := dClient.BalanceAt(ctx, addr, nil)
	cancel()
	if err != nil {
		return "", err
	}
	// convert to nDione
	balance = balance.Div(balance, big.NewInt(int64(units.Dione)))
	if balance.Cmp(big.NewInt(0)) == 0 {
		return "0", nil
	}
	balanceStr := ""
	if useNanoDione {
		balanceStr = fmt.Sprintf("%9d", balance.Uint64())
	} else {
		balanceStr = fmt.Sprintf("%.9f", float64(balance.Uint64())/float64(units.Dione))
	}
	return balanceStr, nil
}

func getOChainBalanceStr(oClient omegavm.Client, addr string) (string, error) {
	pID, err := address.ParseToID(addr)
	if err != nil {
		return "", err
	}
	ctx, cancel := utils.GetAPIContext()
	resp, err := oClient.GetBalance(ctx, []ids.ShortID{pID})
	cancel()
	if err != nil {
		return "", err
	}
	if resp.Balance == 0 {
		return "0", nil
	}
	balanceStr := ""
	if useNanoDione {
		balanceStr = fmt.Sprintf("%9d", resp.Balance)
	} else {
		balanceStr = fmt.Sprintf("%.9f", float64(resp.Balance)/float64(units.Dione))
	}
	return balanceStr, nil
}
