// Copyright (C) 2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
	"testing"

	"github.com/DioneProtocol/odyssey-cli/internal/mocks"
	"github.com/DioneProtocol/odyssey-cli/pkg/application"
	"github.com/DioneProtocol/odyssey-cli/pkg/constants"
	"github.com/DioneProtocol/odyssey-cli/pkg/models"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	testOdygoVersion         = "v0.4.2"
	testUnlistedOdygoVersion = "v0.4.3"
)

var (
	testSubnetEVMCompat = []byte("{\"rpcChainVMProtocolVersion\": {\"v0.4.2\": 18,\"v0.4.1\": 18,\"v0.4.0\": 17}}")
	testOdygoCompat     = []byte("{\"19\": [\"v1.9.2\"],\"18\": [\"v1.9.1\"],\"17\": [\"v1.9.0\",\"v1.8.0\"]}")
	testOdygoCompat2    = []byte("{\"19\": [\"v1.9.2\", \"v1.9.1\"],\"18\": [\"v1.9.0\"]}")
	testOdygoCompat3    = []byte("{\"19\": [\"v1.9.1\", \"v1.9.2\"],\"18\": [\"v1.9.0\"]}")
	testOdygoCompat4    = []byte("{\"19\": [\"v1.9.1\", \"v1.9.2\", \"v1.9.11\"],\"18\": [\"v1.9.0\"]}")
)

func TestGetRPCProtocolVersionSubnetEVM(t *testing.T) {
	require := require.New(t)
	expectedRPC := 18
	var vm models.VMType = models.SubnetEvm

	mockDownloader := &mocks.Downloader{}
	mockDownloader.On("Download", mock.Anything).Return(testSubnetEVMCompat, nil)

	app := application.New()
	app.Downloader = mockDownloader

	rpcVersion, err := GetRPCProtocolVersion(app, vm, testOdygoVersion)
	require.NoError(err)
	require.Equal(expectedRPC, rpcVersion)
}

func TestGetRPCProtocolVersionUnknownVM(t *testing.T) {
	require := require.New(t)
	var vm models.VMType = "unknown"

	app := application.New()

	_, err := GetRPCProtocolVersion(app, vm, testOdygoVersion)
	require.ErrorContains(err, "unknown VM type")
}

func TestGetRPCProtocolVersionMissing(t *testing.T) {
	require := require.New(t)

	mockDownloader := &mocks.Downloader{}
	mockDownloader.On("Download", mock.Anything).Return(testSubnetEVMCompat, nil)

	app := application.New()
	app.Downloader = mockDownloader

	_, err := GetRPCProtocolVersion(app, models.SubnetEvm, testUnlistedOdygoVersion)
	require.ErrorContains(err, "no RPC version found")
}

func TestGetLatestOdysseyGoByProtocolVersion(t *testing.T) {
	type versionTest struct {
		name            string
		rpc             int
		testData        []byte
		latestVersion   string
		expectedVersion string
		expectedErr     error
	}

	tests := []versionTest{
		{
			name:            "latest, one entry",
			rpc:             19,
			testData:        testOdygoCompat,
			latestVersion:   "v1.9.2",
			expectedVersion: "v1.9.2",
			expectedErr:     nil,
		},
		{
			name:            "older, one entry",
			rpc:             18,
			testData:        testOdygoCompat,
			latestVersion:   "v1.9.2",
			expectedVersion: "v1.9.1",
			expectedErr:     nil,
		},
		{
			name:            "latest, multiple entry",
			rpc:             19,
			testData:        testOdygoCompat2,
			latestVersion:   "v1.9.2",
			expectedVersion: "v1.9.2",
			expectedErr:     nil,
		},
		{
			name:            "latest, multiple entry, reverse sorted",
			rpc:             19,
			testData:        testOdygoCompat3,
			latestVersion:   "v1.9.2",
			expectedVersion: "v1.9.2",
			expectedErr:     nil,
		},
		{
			name:            "latest, multiple entry, unreleased version",
			rpc:             19,
			testData:        testOdygoCompat2,
			latestVersion:   "v1.9.1",
			expectedVersion: "v1.9.1",
			expectedErr:     nil,
		},
		{
			name:            "no rpc version",
			rpc:             20,
			testData:        testOdygoCompat2,
			latestVersion:   "v1.9.2",
			expectedVersion: "",
			expectedErr:     ErrNoOdygoVersion,
		},
		{
			name:            "existing rpc, but no eligible version",
			rpc:             19,
			testData:        testOdygoCompat,
			latestVersion:   "v1.9.1",
			expectedVersion: "",
			expectedErr:     ErrNoOdygoVersion,
		},
		{
			name:            "string sorting test",
			rpc:             19,
			testData:        testOdygoCompat4,
			latestVersion:   "v1.9.11",
			expectedVersion: "v1.9.11",
			expectedErr:     nil,
		},
		{
			name:            "string sorting test 2",
			rpc:             19,
			testData:        testOdygoCompat4,
			latestVersion:   "v1.9.2",
			expectedVersion: "v1.9.2",
			expectedErr:     nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require := require.New(t)

			mockDownloader := &mocks.Downloader{}
			mockDownloader.On("Download", mock.Anything).Return(tt.testData, nil)
			mockDownloader.On("GetLatestReleaseVersion", mock.Anything).Return(tt.latestVersion, nil)

			app := application.New()
			app.Downloader = mockDownloader

			odygoVersion, err := GetLatestOdysseyGoByProtocolVersion(app, tt.rpc, constants.OdysseyGoCompatibilityURL)
			if tt.expectedErr == nil {
				require.NoError(err)
			} else {
				require.ErrorIs(err, tt.expectedErr)
			}
			require.Equal(tt.expectedVersion, odygoVersion)
		})
	}
}
