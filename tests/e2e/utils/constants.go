// Copyright (C) 2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package utils

const (
	baseDir      = ".odyssey-cli"
	hardhatDir   = "./tests/e2e/hardhat"
	confFilePath = hardhatDir + "/dynamic_conf.json"
	greeterFile  = hardhatDir + "/greeter.json"

	BaseTest               = "./test/index.ts"
	GreeterScript          = "./scripts/deploy.ts"
	GreeterCheck           = "./scripts/checkGreeting.ts"
	SoloSubnetEVMKey1      = "soloSubnetEVMVersion1"
	SoloSubnetEVMKey2      = "soloSubnetEVMVersion2"
	SoloOdygoKey           = "soloOdygoVersion"
	OnlyOdygoKey           = "onlyOdygoVersion"
	MultiOdygoSubnetEVMKey = "multiOdygoSubnetEVMVersion"
	MultiOdygo1Key         = "multiOdygoVersion1"
	MultiOdygo2Key         = "multiOdygoVersion2"
	LatestEVM2OdygoKey     = "latestEVM2Odygo"
	LatestOdygo2EVMKey     = "latestOdygo2EVM"
	OnlyOdygoValue         = "latest"

	SubnetEvmGenesisPath      = "tests/e2e/assets/test_subnet_evm_genesis.json"
	SubnetEvmGenesis2Path     = "tests/e2e/assets/test_subnet_evm_genesis_2.json"
	EwoqKeyPath               = "tests/e2e/assets/ewoq_key.pk"
	SubnetEvmAllowFeeRecpPath = "tests/e2e/assets/test_subnet_evm_allowFeeRecps_genesis.json"
	SubnetEvmGenesisBadPath   = "tests/e2e/assets/test_subnet_evm_genesis_bad.json"

	PluginDirExt = "plugins"

	ledgerSimDir         = "./tests/e2e/ledgerSim"
	basicLedgerSimScript = "./launchAndApproveTxs.ts"
)
