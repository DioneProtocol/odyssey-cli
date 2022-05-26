// Copyright (C) 2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package cmd

import (
	"errors"

	"github.com/ava-labs/avalanche-cli/cmd/prompts"
	"github.com/ava-labs/avalanche-cli/pkg/models"
	"github.com/ava-labs/avalanche-cli/pkg/vm"
	"github.com/ava-labs/avalanche-cli/ux"
	"github.com/spf13/cobra"
)

var filename string

var (
	forceCreate  bool
	useSubnetEvm bool
)

// var useSpaces *bool
// var useBlob *bool
// var useTimestamp *bool
var useCustom bool

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create [subnetName]",
	Short: "Create a new subnet configuration",
	Long: `The subnet create command builds a new genesis file to configure your subnet.
The command is structured as an interactive wizard. It will walk you through
all the steps you need to create your first subnet.

Currently, the tool supports using the Subnet-EVM as your base genesis
template. You can also provide a custom, user-generated genesis by inputing
a file path with the --file flag. As more subnets reach maturity, you'll be
able to use this tool to generate additional VM templates, such as the
SpacesVM.

By default, running the command with a subnetName that already exists will
cause the command to fail. If you’d like to overwrite an existing
configuration, pass the -f flag.`,
	Args: cobra.ExactArgs(1),
	RunE: createGenesis,
}

func moreThanOneVmSelected() bool {
	vmVars := []bool{useSubnetEvm, useCustom}
	firstSelect := false
	for _, val := range vmVars {
		if firstSelect && val {
			return true
		} else if val {
			firstSelect = true
		}
	}
	return false
}

func getVmFromFlag() models.VmType {
	if useSubnetEvm {
		return models.SubnetEvm
	}
	if useCustom {
		return models.CustomVm
	}
	return ""
}

func createGenesis(cmd *cobra.Command, args []string) error {
	if moreThanOneVmSelected() {
		return errors.New("Too many VMs selected. Provide at most one VM selection flag.")
	}

	if filename == "" {

		var subnetType models.VmType
		var err error
		subnetType = getVmFromFlag()

		if subnetType == "" {

			subnetTypeStr, err := prompts.CaptureList(
				"Choose your VM",
				[]string{subnetEvm, customVm},
			)
			if err != nil {
				return err
			}
			subnetType = models.VmTypeFromString(subnetTypeStr)
		}

		var genesisBytes []byte

		switch subnetType {
		case subnetEvm:
			genesisBytes, err = vm.CreateEvmGenesis(args[0], log)
			if err != nil {
				return err
			}

			err = createSidecar(args[0], models.SubnetEvm)
			if err != nil {
				return err
			}
		case customVm:
			genesisBytes, err = vm.CreateCustomGenesis(args[0], log)
			if err != nil {
				return err
			}
			err = createSidecar(args[0], models.CustomVm)
			if err != nil {
				return err
			}
		default:
			return errors.New("Not implemented")
		}

		err = writeGenesisFile(args[0], genesisBytes)
		if err != nil {
			return err
		}
		ux.Logger.PrintToUser("Successfully created genesis")
	} else {
		ux.Logger.PrintToUser("Using specified genesis")
		err := copyGenesisFile(filename, args[0])
		if err != nil {
			return err
		}

		var subnetType models.VmType
		subnetType = getVmFromFlag()

		if subnetType == "" {
			subnetTypeStr, err := prompts.CaptureList(
				"What VM does your genesis use?",
				[]string{subnetEvm, customVm},
			)
			if err != nil {
				return err
			}
			subnetType = models.VmTypeFromString(subnetTypeStr)
		}
		err = createSidecar(args[0], subnetType)
		if err != nil {
			return err
		}
		ux.Logger.PrintToUser("Successfully created genesis")
	}
	return nil
}
