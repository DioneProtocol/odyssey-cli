// Copyright (C) 2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package subnetcmd

import (
	"errors"
	"os"
	"strconv"
	"time"

	"github.com/DioneProtocol/odyssey-cli/cmd/flags"
	"github.com/DioneProtocol/odyssey-cli/pkg/models"
	"github.com/DioneProtocol/odyssey-cli/pkg/subnet"
	"github.com/DioneProtocol/odysseygo/ids"
	"github.com/DioneProtocol/odysseygo/vms/omegavm"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

var (
	validatorsLocal   bool
	validatorsTestnet bool
	validatorsMainnet bool
)

// odyssey subnet validators
func newValidatorsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validators [subnetName]",
		Short: "List a subnet's validators",
		Long: `The subnet validators command lists the validators of a subnet and provides
severarl statistics about them.`,
		RunE:         printValidators,
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
	}
	cmd.Flags().BoolVarP(&validatorsLocal, "local", "l", false, "deploy to a local network")
	cmd.Flags().BoolVarP(&validatorsTestnet, "testnet", "t", false, "deploy to testnet")
	cmd.Flags().BoolVarP(&validatorsMainnet, "mainnet", "m", false, "deploy to mainnet")
	return cmd
}

func printValidators(_ *cobra.Command, args []string) error {
	if !flags.EnsureMutuallyExclusive([]bool{validatorsLocal, validatorsTestnet, validatorsMainnet}) {
		return errMutuallyExclusiveNetworks
	}

	network := models.UndefinedNetwork
	switch {
	case validatorsLocal:
		network = models.LocalNetwork
	case validatorsTestnet:
		network = models.TestnetNetwork
	case validatorsMainnet:
		network = models.MainnetNetwork
	}

	if network.Kind == models.Undefined {
		// no flag was set, prompt user
		networkStr, err := app.Prompt.CaptureList(
			"Choose a network to list validators from",
			[]string{models.Local.String(), models.Testnet.String(), models.Mainnet.String()},
		)
		if err != nil {
			return err
		}
		network = models.NetworkFromString(networkStr)
	}

	// get the subnetID
	sc, err := app.LoadSidecar(args[0])
	if err != nil {
		return err
	}

	deployInfo, ok := sc.Networks[network.Name()]
	if !ok {
		return errors.New("no deployment found for subnet")
	}

	subnetID := deployInfo.SubnetID

	if network.Kind == models.Local {
		return printLocalValidators(subnetID)
	} else {
		return printPublicValidators(subnetID, network)
	}
}

func printLocalValidators(subnetID ids.ID) error {
	validators, err := subnet.GetSubnetValidators(subnetID)
	if err != nil {
		return err
	}

	return printValidatorsFromList(validators)
}

func printPublicValidators(subnetID ids.ID, network models.Network) error {
	validators, err := subnet.GetPublicSubnetValidators(subnetID, network)
	if err != nil {
		return err
	}

	return printValidatorsFromList(validators)
}

func printValidatorsFromList(validators []omegavm.ClientPermissionlessValidator) error {
	header := []string{"NodeID", "Stake Amount", "Delegator Weight", "Start Time", "End Time", "Type"}
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(header)
	table.SetRowLine(true)

	for _, validator := range validators {
		var delegatorWeight uint64
		if validator.DelegatorWeight != nil {
			delegatorWeight = *validator.DelegatorWeight
		}

		validatorType := "permissioned"
		if validator.PotentialReward != nil && *validator.PotentialReward > 0 {
			validatorType = "elastic"
		}

		table.Append([]string{
			validator.NodeID.String(),
			strconv.FormatUint(*validator.StakeAmount, 10),
			strconv.FormatUint(delegatorWeight, 10),
			formatUnixTime(validator.StartTime),
			formatUnixTime(validator.EndTime),
			validatorType,
		})
	}

	table.Render()

	return nil
}

func formatUnixTime(unixTime uint64) string {
	return time.Unix(int64(unixTime), 0).Format(time.RFC3339)
}
