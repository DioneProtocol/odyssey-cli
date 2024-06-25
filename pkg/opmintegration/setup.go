// Copyright (C) 2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package opmintegration

import (
	"os"

	"github.com/DioneProtocol/odyssey-cli/pkg/application"
	"github.com/DioneProtocol/odyssey-cli/pkg/constants"
	"github.com/DioneProtocol/opm/config"
	"github.com/DioneProtocol/opm/opm"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v2"
)

// Note, you can only call this method once per run
func SetupOpm(app *application.Odyssey, opmBaseDir string) error {
	credentials, err := initCredentials(app)
	if err != nil {
		return err
	}

	// Need to initialize a afero filesystem object to run opm
	fs := afero.NewOsFs()

	err = os.MkdirAll(app.GetOPMPluginDir(), constants.DefaultPerms755)
	if err != nil {
		return err
	}

	// The New() function has a lot of prints we'd like to hide from the user,
	// so going to divert stdout to the log temporarily
	stdOutHolder := os.Stdout
	opmLog, err := os.OpenFile(app.GetOPMLog(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, constants.DefaultPerms755)
	if err != nil {
		return err
	}
	defer opmLog.Close()
	os.Stdout = opmLog
	opmConfig := opm.Config{
		Directory:        opmBaseDir,
		Auth:             credentials,
		AdminAPIEndpoint: app.Conf.GetConfigStringValue(constants.ConfigOPMAdminAPIEndpointKey),
		PluginDir:        app.GetOPMPluginDir(),
		Fs:               fs,
	}
	opmInstance, err := opm.New(opmConfig)
	if err != nil {
		return err
	}
	os.Stdout = stdOutHolder
	app.Opm = opmInstance

	app.OpmDir = opmBaseDir
	return err
}

// If we need to use custom git credentials (say for private repos).
// the zero value for credentials is safe to use.
// Stolen from OPM repo
func initCredentials(app *application.Odyssey) (http.BasicAuth, error) {
	result := http.BasicAuth{}

	if app.Conf.ConfigValueIsSet(constants.ConfigOPMCredentialsFileKey) {
		credentials := &config.Credential{}

		bytes, err := os.ReadFile(app.Conf.GetConfigStringValue(constants.ConfigOPMCredentialsFileKey))
		if err != nil {
			return result, err
		}
		if err := yaml.Unmarshal(bytes, credentials); err != nil {
			return result, err
		}

		result.Username = credentials.Username
		result.Password = credentials.Password
	}

	return result, nil
}
