// Copyright (C) 2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package subnetcmd

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/DioneProtocol/odyssey-cli/pkg/models"
	"github.com/spf13/cobra"
)

// odyssey subnet delete
func newDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete",
		Short: "Delete a subnet configuration",
		Long:  "The subnet delete command deletes an existing subnet configuration.",
		RunE:  deleteSubnet,
		Args:  cobra.ExactArgs(1),
	}
}

func deleteSubnet(_ *cobra.Command, args []string) error {
	// TODO sanitize this input
	subnetName := args[0]
	subnetDir := filepath.Join(app.GetSubnetDir(), subnetName)

	customVMPath := app.GetCustomVMPath(subnetName)

	sidecar, err := app.LoadSidecar(subnetName)
	if err != nil {
		return err
	}

	if sidecar.VM == models.CustomVM {
		if _, err := os.Stat(customVMPath); err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				return err
			}
			app.Log.Warn("tried to remove custom VM path but it actually does not exist. Ignoring")
			return nil
		}

		// exists
		if err := os.Remove(customVMPath); err != nil {
			return err
		}
	}

	// TODO this method does not delete the imported VM binary if this
	// is an OPM subnet. We can't naively delete the binary because it
	// may be used by multiple subnets. We should delete this binary,
	// but only if no other subnet is using it.

	if _, err := os.Stat(subnetDir); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		app.Log.Warn("tried to remove the Subnet dir path but it actually does not exist. Ignoring")
		return nil
	}

	// exists
	if err := os.RemoveAll(subnetDir); err != nil {
		return err
	}
	return nil
}
