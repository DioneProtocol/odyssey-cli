// Copyright (C) 2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package binutils

import "github.com/DioneProtocol/odyssey-cli/pkg/application"

func SetupCustomBin(app *application.Odyssey, subnetName string) string {
	// Just need to get the path of the vm
	return app.GetCustomVMPath(subnetName)
}

func SetupOPMBin(app *application.Odyssey, vmid string) string {
	// Just need to get the path of the vm
	return app.GetOPMVMPath(vmid)
}
