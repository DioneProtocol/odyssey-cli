// Copyright (C) 2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package opmintegration

import (
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/DioneProtocol/odyssey-cli/pkg/application"
	"github.com/DioneProtocol/odyssey-cli/pkg/constants"
	"github.com/DioneProtocol/odyssey-cli/pkg/ux"
)

const gitExtension = ".git"

// Returns alias
func AddRepo(app *application.Odyssey, repoURL *url.URL, branch string) (string, error) {
	alias, err := getAlias(repoURL)
	if err != nil {
		return "", err
	}

	if alias == constants.DefaultDioneProtocolPackage {
		ux.Logger.PrintToUser("Odyssey Plugins Core already installed, skipping...")
		return "", nil
	}

	repoStr := repoURL.String()

	if path.Ext(repoStr) != gitExtension {
		repoStr += gitExtension
	}

	fmt.Println("Installing repo")

	return alias, app.Opm.AddRepository(alias, repoStr, branch)
}

func UpdateRepos(app *application.Odyssey) error {
	return app.Opm.Update()
}

func InstallVM(app *application.Odyssey, subnetKey string) error {
	vms, err := getVMsInSubnet(app, subnetKey)
	if err != nil {
		return err
	}

	splitKey := strings.Split(subnetKey, ":")
	if len(splitKey) != 2 {
		return fmt.Errorf("invalid key: %s", subnetKey)
	}

	repo := splitKey[0]

	for _, vm := range vms {
		toInstall := repo + ":" + vm
		fmt.Println("Installing vm:", toInstall)
		err = app.Opm.Install(toInstall)
		if err != nil {
			return err
		}
	}

	return nil
}
