// Copyright (C) 2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package networkcmd

import (
	"testing"

	"github.com/DioneProtocol/odyssey-cli/internal/mocks"
	"github.com/DioneProtocol/odyssey-cli/internal/testutils"
	"github.com/DioneProtocol/odyssey-cli/pkg/models"
	"github.com/DioneProtocol/odysseygo/ids"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var testOdygoCompat = []byte("{\"19\": [\"v1.9.2\"],\"18\": [\"v1.9.1\"],\"17\": [\"v1.9.0\",\"v1.8.0\"]}")

func Test_determineOdygoVersion(t *testing.T) {
	subnetName1 := "test1"
	subnetName2 := "test2"
	subnetName3 := "test3"
	subnetName4 := "test4"

	dummySlice := ids.ID{1, 2, 3, 4}

	sc1 := models.Sidecar{
		Name: subnetName1,
		Networks: map[string]models.NetworkData{
			models.Local.String(): {
				SubnetID:     dummySlice,
				BlockchainID: dummySlice,
				RPCVersion:   18,
			},
		},
		VM: models.SubnetEvm,
	}

	sc2 := models.Sidecar{
		Name: subnetName2,
		Networks: map[string]models.NetworkData{
			models.Local.String(): {
				SubnetID:     dummySlice,
				BlockchainID: dummySlice,
				RPCVersion:   18,
			},
		},
		VM: models.SubnetEvm,
	}

	sc3 := models.Sidecar{
		Name: subnetName3,
		Networks: map[string]models.NetworkData{
			models.Local.String(): {
				SubnetID:     dummySlice,
				BlockchainID: dummySlice,
				RPCVersion:   19,
			},
		},
		VM: models.SubnetEvm,
	}

	scCustom := models.Sidecar{
		Name: subnetName4,
		Networks: map[string]models.NetworkData{
			models.Local.String(): {
				SubnetID:     dummySlice,
				BlockchainID: dummySlice,
				RPCVersion:   0,
			},
		},
		VM: models.CustomVM,
	}

	type test struct {
		name          string
		userOdygo     string
		sidecars      []models.Sidecar
		expectedOdygo string
		expectedErr   bool
	}

	tests := []test{
		{
			name:          "user not latest",
			userOdygo:     "v1.9.5",
			sidecars:      []models.Sidecar{sc1},
			expectedOdygo: "v1.9.5",
			expectedErr:   false,
		},
		{
			name:          "single sc",
			userOdygo:     "latest",
			sidecars:      []models.Sidecar{sc1},
			expectedOdygo: "v1.9.1",
			expectedErr:   false,
		},
		{
			name:          "multi sc matching",
			userOdygo:     "latest",
			sidecars:      []models.Sidecar{sc1, sc2},
			expectedOdygo: "v1.9.1",
			expectedErr:   false,
		},
		{
			name:          "multi sc mismatch",
			userOdygo:     "latest",
			sidecars:      []models.Sidecar{sc1, sc3},
			expectedOdygo: "",
			expectedErr:   true,
		},
		{
			name:          "single custom",
			userOdygo:     "latest",
			sidecars:      []models.Sidecar{scCustom},
			expectedOdygo: "latest",
			expectedErr:   false,
		},
		{
			name:          "custom plus user selected",
			userOdygo:     "v1.9.1",
			sidecars:      []models.Sidecar{scCustom},
			expectedOdygo: "v1.9.1",
			expectedErr:   false,
		},
		{
			name:          "multi sc matching plus custom",
			userOdygo:     "latest",
			sidecars:      []models.Sidecar{sc1, sc2, scCustom},
			expectedOdygo: "v1.9.1",
			expectedErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app = testutils.SetupTestInTempDir(t)
			mockDownloader := &mocks.Downloader{}
			mockDownloader.On("Download", mock.Anything).Return(testOdygoCompat, nil)
			mockDownloader.On("GetLatestReleaseVersion", mock.Anything).Return("v1.9.2", nil)

			app.Downloader = mockDownloader

			for i := range tt.sidecars {
				err := app.CreateSidecar(&tt.sidecars[i])
				require.NoError(t, err)
			}

			odygoVersion, err := determineOdygoVersion(tt.userOdygo)
			if tt.expectedErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tt.expectedOdygo, odygoVersion)
		})
	}
}
