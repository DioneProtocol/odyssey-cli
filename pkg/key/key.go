// Copyright (C) 2019-2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package key implements key manager and helper functions.
package key

import (
	"bytes"
	"errors"
	"sort"

	"github.com/DioneProtocol/odysseygo/ids"
	"github.com/DioneProtocol/odysseygo/utils/constants"
	"github.com/DioneProtocol/odysseygo/vms/components/dione"
	"github.com/DioneProtocol/odysseygo/vms/omegavm/txs"
	"github.com/DioneProtocol/odysseygo/vms/secp256k1fx"
)

var (
	ErrInvalidType = errors.New("invalid type")
	ErrCantSpend   = errors.New("can't spend")
)

// Key defines methods for key manager interface.
type Key interface {
	// O returns all formatted O-Chain addresses.
	O() []string
	// D returns the D-Chain address in Ethereum format
	D() string
	// Addresses returns the all raw ids.ShortID address.
	Addresses() []ids.ShortID
	// Match attempts to match a list of addresses up to the provided threshold.
	Match(owners *secp256k1fx.OutputOwners, time uint64) ([]uint32, []ids.ShortID, bool)
	// Spend attempts to spend all specified UTXOs (outputs)
	// and returns the new UTXO inputs.
	//
	// If target amount is specified, it only uses the
	// outputs until the total spending is below the target
	// amount.
	Spends(outputs []*dione.UTXO, opts ...OpOption) (
		totalBalanceToSpend uint64,
		inputs []*dione.TransferableInput,
		signers [][]ids.ShortID,
	)
	// Sign generates [numSigs] signatures and attaches them to [oTx].
	Sign(oTx *txs.Tx, signers [][]ids.ShortID) error
}

type Op struct {
	time         uint64
	targetAmount uint64
	feeDeduct    uint64
}

type OpOption func(*Op)

func (op *Op) applyOpts(opts []OpOption) {
	for _, opt := range opts {
		opt(op)
	}
}

func WithTime(t uint64) OpOption {
	return func(op *Op) {
		op.time = t
	}
}

func WithTargetAmount(ta uint64) OpOption {
	return func(op *Op) {
		op.targetAmount = ta
	}
}

// To deduct transfer fee from total spend (output).
// e.g., "units.MilliDione" for A/O-Chain transfer.
func WithFeeDeduct(fee uint64) OpOption {
	return func(op *Op) {
		op.feeDeduct = fee
	}
}

func GetHRP(networkID uint32) string {
	switch networkID {
	case constants.LocalID:
		return constants.LocalHRP
	case constants.TestnetID:
		return constants.TestnetHRP
	case constants.MainnetID:
		return constants.MainnetHRP
	default:
		return constants.FallbackHRP
	}
}

type innerSortTransferableInputsWithSigners struct {
	ins     []*dione.TransferableInput
	signers [][]ids.ShortID
}

func (ins *innerSortTransferableInputsWithSigners) Less(i, j int) bool {
	iID, iIndex := ins.ins[i].InputSource()
	jID, jIndex := ins.ins[j].InputSource()

	switch bytes.Compare(iID[:], jID[:]) {
	case -1:
		return true
	case 0:
		return iIndex < jIndex
	default:
		return false
	}
}

func (ins *innerSortTransferableInputsWithSigners) Len() int {
	return len(ins.ins)
}

func (ins *innerSortTransferableInputsWithSigners) Swap(i, j int) {
	ins.ins[j], ins.ins[i] = ins.ins[i], ins.ins[j]
	ins.signers[j], ins.signers[i] = ins.signers[i], ins.signers[j]
}

// SortTransferableInputsWithSigners sorts the inputs and signers based on the
// input's utxo ID.
func SortTransferableInputsWithSigners(ins []*dione.TransferableInput, signers [][]ids.ShortID) {
	sort.Sort(&innerSortTransferableInputsWithSigners{ins: ins, signers: signers})
}
