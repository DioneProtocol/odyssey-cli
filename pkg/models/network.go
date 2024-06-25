// Copyright (C) 2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package models

import (
	"fmt"
	"os"

	"github.com/DioneProtocol/odyssey-cli/pkg/constants"
	"github.com/DioneProtocol/odysseygo/genesis"
	odygoconstants "github.com/DioneProtocol/odysseygo/utils/constants"
)

type NetworkKind int64

const (
	Undefined NetworkKind = iota
	Mainnet
	Testnet
	Local
	Devnet
)

func (nk NetworkKind) String() string {
	switch nk {
	case Mainnet:
		return "Mainnet"
	case Testnet:
		return "Testnet"
	case Local:
		return "Local Network"
	case Devnet:
		return "Devnet"
	}
	return "invalid network"
}

type Network struct {
	Kind     NetworkKind
	ID       uint32
	Endpoint string
}

var (
	UndefinedNetwork = NewNetwork(Undefined, 0, "")
	LocalNetwork     = NewNetwork(Local, constants.LocalNetworkID, constants.LocalAPIEndpoint)
	DevnetNetwork    = NewNetwork(Devnet, constants.DevnetNetworkID, constants.DevnetAPIEndpoint)
	TestnetNetwork   = NewNetwork(Testnet, odygoconstants.TestnetID, constants.TestnetAPIEndpoint)
	MainnetNetwork   = NewNetwork(Mainnet, odygoconstants.MainnetID, constants.MainnetAPIEndpoint)
)

func NewNetwork(kind NetworkKind, id uint32, endpoint string) Network {
	return Network{
		Kind:     kind,
		ID:       id,
		Endpoint: endpoint,
	}
}

func NewDevnetNetwork(ip string, port int) Network {
	endpoint := fmt.Sprintf("http://%s:%d", ip, port)
	return NewNetwork(Devnet, constants.DevnetNetworkID, endpoint)
}

func NetworkFromString(s string) Network {
	switch s {
	case Mainnet.String():
		return MainnetNetwork
	case Testnet.String():
		return TestnetNetwork
	case Local.String():
		return LocalNetwork
	case Devnet.String():
		return DevnetNetwork
	}
	return UndefinedNetwork
}

func NetworkFromNetworkID(networkID uint32) Network {
	switch networkID {
	case odygoconstants.MainnetID:
		return MainnetNetwork
	case odygoconstants.TestnetID:
		return TestnetNetwork
	case constants.LocalNetworkID:
		return LocalNetwork
	case constants.DevnetNetworkID:
		return DevnetNetwork
	}
	return UndefinedNetwork
}

func (n Network) Name() string {
	return n.Kind.String()
}

func (n Network) DChainEndpoint() string {
	return fmt.Sprintf("%s/ext/bc/%s/rpc", n.Endpoint, "D")
}

func (n Network) NetworkIDFlagValue() string {
	switch n.Kind {
	case Local:
		return fmt.Sprintf("network-%d", n.ID)
	case Devnet:
		return fmt.Sprintf("network-%d", n.ID)
	case Testnet:
		return "testnet"
	case Mainnet:
		return "mainnet"
	}
	return "invalid-network"
}

func (n Network) GenesisParams() *genesis.Params {
	switch n.Kind {
	case Local:
		return &genesis.LocalParams
	case Devnet:
		return &genesis.LocalParams
	case Testnet:
		return &genesis.TestnetParams
	case Mainnet:
		return &genesis.MainnetParams
	}
	return nil
}

func (n *Network) HandlePublicNetworkSimulation() {
	// used in E2E to simulate public network execution paths on a local network
	if os.Getenv(constants.SimulatePublicNetwork) != "" {
		n.Kind = Local
		n.ID = constants.LocalNetworkID
		n.Endpoint = constants.LocalAPIEndpoint
	}
}
