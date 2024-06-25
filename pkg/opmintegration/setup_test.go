// Copyright (C) 2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package opmintegration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/DioneProtocol/odyssey-cli/pkg/constants"
	"github.com/stretchr/testify/require"
)

func TestSetupOPM(t *testing.T) {
	require := require.New(t)
	testDir := t.TempDir()
	app := newTestApp(t, testDir)

	err := os.MkdirAll(filepath.Dir(app.GetOPMLog()), constants.DefaultPerms755)
	require.NoError(err)

	err = SetupOpm(app, testDir)
	require.NoError(err)
	require.NotEqual(nil, app.Opm)
	require.Equal(testDir, app.OpmDir)
}
