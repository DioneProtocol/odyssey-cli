// Copyright (C) 2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package txutils

import (
	"context"
	"fmt"

	"github.com/DioneProtocol/odyssey-cli/pkg/key"
	"github.com/DioneProtocol/odyssey-cli/pkg/models"
	"github.com/DioneProtocol/odysseygo/ids"
	"github.com/DioneProtocol/odysseygo/utils/formatting/address"
	"github.com/DioneProtocol/odysseygo/vms/omegavm"
	"github.com/DioneProtocol/odysseygo/vms/omegavm/txs"
	"github.com/DioneProtocol/odysseygo/vms/secp256k1fx"
)

// get network model associated to tx
func GetNetwork(tx *txs.Tx) (models.Network, error) {
	unsignedTx := tx.Unsigned
	var networkID uint32
	switch unsignedTx := unsignedTx.(type) {
	case *txs.RemoveSubnetValidatorTx:
		networkID = unsignedTx.NetworkID
	case *txs.AddSubnetValidatorTx:
		networkID = unsignedTx.NetworkID
	case *txs.CreateChainTx:
		networkID = unsignedTx.NetworkID
	case *txs.TransformSubnetTx:
		networkID = unsignedTx.NetworkID
	case *txs.AddPermissionlessValidatorTx:
		networkID = unsignedTx.NetworkID
	default:
		return models.UndefinedNetwork, fmt.Errorf("unexpected unsigned tx type %T", unsignedTx)
	}
	network := models.NetworkFromNetworkID(networkID)
	if network.Kind == models.Undefined {
		return models.UndefinedNetwork, fmt.Errorf("undefined network model for tx")
	}
	return network, nil
}

// get subnet id associated to tx
func GetSubnetID(tx *txs.Tx) (ids.ID, error) {
	unsignedTx := tx.Unsigned
	var subnetID ids.ID
	switch unsignedTx := unsignedTx.(type) {
	case *txs.RemoveSubnetValidatorTx:
		subnetID = unsignedTx.Subnet
	case *txs.AddSubnetValidatorTx:
		subnetID = unsignedTx.SubnetValidator.Subnet
	case *txs.CreateChainTx:
		subnetID = unsignedTx.SubnetID
	case *txs.TransformSubnetTx:
		subnetID = unsignedTx.Subnet
	case *txs.AddPermissionlessValidatorTx:
		subnetID = unsignedTx.Subnet
	default:
		return ids.Empty, fmt.Errorf("unexpected unsigned tx type %T", unsignedTx)
	}
	return subnetID, nil
}

func GetLedgerDisplayName(tx *txs.Tx) string {
	unsignedTx := tx.Unsigned
	switch unsignedTx.(type) {
	case *txs.AddSubnetValidatorTx:
		return "SubnetValidator"
	case *txs.CreateChainTx:
		return "CreateChain"
	default:
		return ""
	}
}

func IsCreateChainTx(tx *txs.Tx) bool {
	_, ok := tx.Unsigned.(*txs.CreateChainTx)
	return ok
}

func GetOwners(network models.Network, subnetID ids.ID) ([]string, uint32, error) {
	oClient := omegavm.NewClient(network.Endpoint)
	ctx := context.Background()
	txBytes, err := oClient.GetTx(ctx, subnetID)
	if err != nil {
		return nil, 0, fmt.Errorf("subnet tx %s query error: %w", subnetID, err)
	}
	var tx txs.Tx
	if _, err := txs.Codec.Unmarshal(txBytes, &tx); err != nil {
		return nil, 0, fmt.Errorf("couldn't unmarshal tx %s: %w", subnetID, err)
	}
	createSubnetTx, ok := tx.Unsigned.(*txs.CreateSubnetTx)
	if !ok {
		return nil, 0, fmt.Errorf("got unexpected type %T for subnet tx %s", tx.Unsigned, subnetID)
	}
	owner, ok := createSubnetTx.Owner.(*secp256k1fx.OutputOwners)
	if !ok {
		return nil, 0, fmt.Errorf("got unexpected type %T for subnet owners tx %s", createSubnetTx.Owner, subnetID)
	}
	controlKeys := owner.Addrs
	threshold := owner.Threshold
	hrp := key.GetHRP(network.ID)
	controlKeysStrs := []string{}
	for _, addr := range controlKeys {
		addrStr, err := address.Format("O", hrp, addr[:])
		if err != nil {
			return nil, 0, err
		}
		controlKeysStrs = append(controlKeysStrs, addrStr)
	}
	return controlKeysStrs, threshold, nil
}
