// Copyright (C) 2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package binutils

import (
	"github.com/DioneProtocol/odyssey-cli/pkg/application"
	"github.com/DioneProtocol/odyssey-cli/pkg/constants"
)

func SetupOdysseygo(app *application.Odyssey, odygoVersion string) (string, string, error) {
	binDir := app.GetOdysseygoBinDir()

	installer := NewInstaller()
	downloader := NewOdygoDownloader()
	return InstallBinary(
		app,
		odygoVersion,
		binDir,
		binDir,
		odysseygoBinPrefix,
		constants.DioneProtocolOrg,
		constants.OdysseyGoRepoName,
		downloader,
		installer,
	)
}
