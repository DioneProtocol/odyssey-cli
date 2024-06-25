// Copyright (C) 2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package models

import (
	"time"

	"github.com/DioneProtocol/odysseygo/ids"
)

type ElasticSubnetConfig struct {
	SubnetID                  ids.ID
	AssetID                   ids.ID
	InitialSupply             uint64
	MaxSupply                 uint64
	MinConsumptionRate        uint64
	MaxConsumptionRate        uint64
	MinValidatorStake         uint64
	MaxValidatorStake         uint64
	MinValidatorStakeDuration time.Duration
	MaxValidatorStakeDuration time.Duration
	MinDelegatorStakeDuration time.Duration
	MaxDelegatorStakeDuration time.Duration
	MinDelegationFee          uint32
	MinDelegatorStake         uint64
	MaxValidatorWeightFactor  byte
	UptimeRequirement         uint32
}
