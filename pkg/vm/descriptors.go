// Copyright (C) 2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
	"math/big"

	"github.com/ava-labs/avalanche-cli/pkg/application"
	"github.com/ava-labs/avalanche-cli/pkg/binutils"
	"github.com/ava-labs/avalanche-cli/pkg/constants"
	"github.com/ava-labs/avalanche-cli/pkg/ux"
)

func getChainID(app *application.Avalanche) (*big.Int, error) {
	// TODO check against known chain ids and provide warning
	ux.Logger.PrintToUser("Enter your subnet's ChainId. It can be any positive integer.")

	chainID, err := app.Prompt.CapturePositiveBigInt("ChainId")
	if err != nil {
		return nil, err
	}

	exists, err := app.ChainIDExists(chainID.String())
	if err != nil {
		return nil, err
	}
	if exists {
		ux.Logger.PrintToUser("The provided chain ID %q already exists! Try a different one:", chainID.String())
		return getChainID(app)
	}

	return chainID, nil
}

func getTokenName(app *application.Avalanche) (string, error) {
	ux.Logger.PrintToUser("Select a symbol for your subnet's native token")
	tokenName, err := app.Prompt.CaptureString("Token symbol")
	if err != nil {
		return "", err
	}

	return tokenName, nil
}

func getSubnetEVMVersion(app *application.Avalanche) (string, error) {
	const (
		useLatest     = "Use latest version"
		useCustom     = "Specify custom version"
		defaultPrompt = "What version of subnet-evm would you like?"
	)

	versionOptions := []string{useLatest, useCustom}

	versionOption, err := app.Prompt.CaptureList(
		defaultPrompt,
		versionOptions,
	)
	if err != nil {
		return "", err
	}

	if versionOption == useLatest {
		// Get and return latest version
		return binutils.GetLatestReleaseVersion(binutils.GetGithubLatestReleaseURL(
			constants.AvaLabsOrg,
			constants.SubnetEVMRepoName,
		))
	}

	// prompt for version
	version, err := app.Prompt.CaptureString("Subnet-EVM version")
	if err != nil {
		return "", err
	}

	return version, nil
}

func getDescriptors(app *application.Avalanche, subnetEVMVersion string) (*big.Int, string, string, stateDirection, error) {
	chainID, err := getChainID(app)
	if err != nil {
		return nil, "", "", stop, err
	}

	tokenName, err := getTokenName(app)
	if err != nil {
		return nil, "", "", stop, err
	}

	if subnetEVMVersion == "latest" {
		subnetEVMVersion, err = binutils.GetLatestReleaseVersion(binutils.GetGithubLatestReleaseURL(
			constants.AvaLabsOrg,
			constants.SubnetEVMRepoName,
		))
		if err != nil {
			return nil, "", "", stop, err
		}
	} else if subnetEVMVersion == "" {
		subnetEVMVersion, err = getSubnetEVMVersion(app)
		if err != nil {
			return nil, "", "", stop, err
		}
	}

	return chainID, tokenName, subnetEVMVersion, forward, nil
}
