// Copyright (C) 2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package migrations

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/DioneProtocol/odyssey-cli/pkg/application"
	"github.com/DioneProtocol/odyssey-cli/pkg/config"
	"github.com/DioneProtocol/odyssey-cli/pkg/constants"
	"github.com/DioneProtocol/odyssey-cli/pkg/models"
	"github.com/DioneProtocol/odyssey-cli/pkg/prompts"
	"github.com/DioneProtocol/odyssey-cli/pkg/ux"
	"github.com/DioneProtocol/odysseygo/utils/logging"
	"github.com/stretchr/testify/require"
)

func TestSubnetEVMRenameMigration(t *testing.T) {
	type test struct {
		name       string
		sc         *models.Sidecar
		expectedVM string
	}

	subnetName := "test"

	tests := []test{
		{
			name: "Convert SubnetEVM",
			sc: &models.Sidecar{
				Name: subnetName,
				VM:   "SubnetEVM",
			},
			expectedVM: "Subnet-EVM",
		},
		{
			name: "Preserve Subnet-EVM",
			sc: &models.Sidecar{
				Name: subnetName,
				VM:   "Subnet-EVM",
			},
			expectedVM: "Subnet-EVM",
		},
		{
			name: "Ignore unknown",
			sc: &models.Sidecar{
				Name: subnetName,
				VM:   "unknown",
			},
			expectedVM: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ux.NewUserLog(logging.NoLog{}, io.Discard)
			require := require.New(t)
			testDir := t.TempDir()

			app := &application.Odyssey{}
			app.Setup(testDir, logging.NoLog{}, config.New(), prompts.NewPrompter(), application.NewDownloader())

			err := app.CreateSidecar(tt.sc)
			require.NoError(err)

			runner := migrationRunner{
				showMsg: true,
				running: false,
				migrations: map[int]migrationFunc{
					0: migrateSubnetEVMNames,
				},
			}
			// run the migration
			err = runner.run(app)
			require.NoError(err)

			loadedSC, err := app.LoadSidecar(tt.sc.Name)
			require.NoError(err)
			require.Equal(tt.expectedVM, string(loadedSC.VM))
		})
	}
}

func TestSubnetEVMRenameMigration_EmptyDir(t *testing.T) {
	ux.NewUserLog(logging.NoLog{}, io.Discard)
	require := require.New(t)
	testDir := t.TempDir()

	app := &application.Odyssey{}
	app.Setup(testDir, logging.NoLog{}, config.New(), prompts.NewPrompter(), application.NewDownloader())

	emptySubnetName := "emptySubnet"

	subnetDir := filepath.Join(app.GetSubnetDir(), emptySubnetName)
	err := os.MkdirAll(subnetDir, constants.DefaultPerms755)
	require.NoError(err)

	runner := migrationRunner{
		showMsg: true,
		running: false,
		migrations: map[int]migrationFunc{
			0: migrateSubnetEVMNames,
		},
	}
	// run the migration
	err = runner.run(app)
	require.NoError(err)
}
