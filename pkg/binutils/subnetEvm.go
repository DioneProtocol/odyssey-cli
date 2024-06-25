// Copyright (C) 2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package binutils

import (
	"path/filepath"

	"github.com/DioneProtocol/odyssey-cli/pkg/application"
	"github.com/DioneProtocol/odyssey-cli/pkg/constants"
)

func SetupSubnetEVM(app *application.Odyssey, subnetEVMVersion string) (string, string, error) {
	// Check if already installed
	binDir := app.GetSubnetEVMBinDir()
	subDir := filepath.Join(binDir, subnetEVMBinPrefix+subnetEVMVersion)

	installer := NewInstaller()
	downloader := NewSubnetEVMDownloader()
	version, vmDir, err := InstallBinary(
		app,
		subnetEVMVersion,
		binDir,
		subDir,
		subnetEVMBinPrefix,
		constants.DioneProtocolOrg,
		constants.SubnetEVMRepoName,
		downloader,
		installer,
	)
	return version, filepath.Join(vmDir, constants.SubnetEVMBin), err
}
