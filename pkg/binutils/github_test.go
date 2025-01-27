// Copyright (C) 2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package binutils

import (
	"errors"
	"testing"

	"github.com/DioneProtocol/odyssey-cli/internal/mocks"
	"github.com/DioneProtocol/odyssey-cli/pkg/constants"
	"github.com/stretchr/testify/require"
)

type urlTest struct {
	version     string
	goarch      string
	goos        string
	expectedURL string
	expectedExt string
	expectedErr error
}

func TestGetGithubLatestReleaseURL(t *testing.T) {
	require := require.New(t)
	expected := "https://api.github.com/repos/DioneProtocol/odysseygo/releases/latest"
	url := GetGithubLatestReleaseURL(constants.DioneProtocolOrg, constants.OdysseyGoRepoName)
	require.Equal(expected, url)
}

func TestGetDownloadURL_OdysseyGo(t *testing.T) {
	tests := []urlTest{
		{
			version:     "v1.17.1",
			goarch:      "amd64",
			goos:        "linux",
			expectedURL: "https://github.com/DioneProtocol/odysseygo/releases/download/v1.17.1/odysseygo-linux-amd64-v1.17.1.tar.gz",
			expectedExt: tarExtension,
			expectedErr: nil,
		},
		{
			version:     "v1.18.5",
			goarch:      "arm64",
			goos:        "darwin",
			expectedURL: "https://github.com/DioneProtocol/odysseygo/releases/download/v1.18.5/odysseygo-macos-v1.18.5.zip",
			expectedExt: zipExtension,
			expectedErr: nil,
		},
		{
			version:     "v2.1.4",
			goarch:      "amd64",
			goos:        "windows",
			expectedURL: "https://github.com/DioneProtocol/odysseygo/releases/download/v2.1.4/odysseygo-win-v2.1.4-experimental.zip",
			expectedExt: zipExtension,
			expectedErr: nil,
		},
		{
			version:     "v1.2.3",
			goarch:      "riscv",
			goos:        "solaris",
			expectedURL: "",
			expectedExt: "",
			expectedErr: errors.New("OS not supported: solaris"),
		},
	}

	for _, tt := range tests {
		require := require.New(t)
		mockInstaller := &mocks.Installer{}
		mockInstaller.On("GetArch").Return(tt.goarch, tt.goos)

		downloader := NewOdygoDownloader()

		url, ext, err := downloader.GetDownloadURL(tt.version, mockInstaller)
		require.Equal(tt.expectedURL, url)
		require.Equal(tt.expectedExt, ext)
		require.Equal(tt.expectedErr, err)
	}
}

func TestGetDownloadURL_SubnetEVM(t *testing.T) {
	tests := []urlTest{
		{
			version:     "v1.17.1",
			goarch:      "amd64",
			goos:        "linux",
			expectedURL: "https://github.com/DioneProtocol/subnet-evm/releases/download/v1.17.1/subnet-evm_1.17.1_linux_amd64.tar.gz",
			expectedExt: tarExtension,
			expectedErr: nil,
		},
		{
			version:     "v1.18.5",
			goarch:      "arm64",
			goos:        "darwin",
			expectedURL: "https://github.com/DioneProtocol/subnet-evm/releases/download/v1.18.5/subnet-evm_1.18.5_darwin_arm64.tar.gz",
			expectedExt: tarExtension,
			expectedErr: nil,
		},
		{
			version:     "v1.2.3",
			goarch:      "riscv",
			goos:        "solaris",
			expectedURL: "",
			expectedExt: "",
			expectedErr: errors.New("OS not supported: solaris"),
		},
	}

	for _, tt := range tests {
		require := require.New(t)
		mockInstaller := &mocks.Installer{}
		mockInstaller.On("GetArch").Return(tt.goarch, tt.goos)

		downloader := NewSubnetEVMDownloader()

		url, ext, err := downloader.GetDownloadURL(tt.version, mockInstaller)
		require.Equal(tt.expectedURL, url)
		require.Equal(tt.expectedExt, ext)
		require.Equal(tt.expectedErr, err)
	}
}
