// Copyright (C) 2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package binutils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/DioneProtocol/odyssey-cli/internal/mocks"
	"github.com/DioneProtocol/odyssey-cli/internal/testutils"
	"github.com/DioneProtocol/odyssey-cli/pkg/application"
	"github.com/DioneProtocol/odyssey-cli/pkg/config"
	"github.com/DioneProtocol/odyssey-cli/pkg/constants"
	"github.com/DioneProtocol/odyssey-cli/pkg/prompts"
	"github.com/DioneProtocol/odysseygo/utils/logging"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	version1 = "v1.17.1"
	version2 = "v1.18.1"

	odysseygoBin = "odysseygo"
)

var (
	binary1 = []byte{0xde, 0xad, 0xbe, 0xef}
	binary2 = []byte{0xfe, 0xed, 0xc0, 0xde}
)

func setupInstallDir(require *require.Assertions) *application.Odyssey {
	rootDir, err := os.MkdirTemp(os.TempDir(), "binutils-tests")
	require.NoError(err)
	defer os.RemoveAll(rootDir)

	app := application.New()
	app.Setup(rootDir, logging.NoLog{}, &config.Config{}, prompts.NewPrompter(), application.NewDownloader())
	return app
}

func Test_installOdysseyGoWithVersion_Zip(t *testing.T) {
	require := testutils.SetupTest(t)

	zipBytes := testutils.CreateDummyOdygoZip(require, binary1)
	app := setupInstallDir(require)

	mockInstaller := &mocks.Installer{}
	mockInstaller.On("GetArch").Return("amd64", "darwin")

	githubDownloader := NewOdygoDownloader()

	mockAppDownloader := mocks.Downloader{}
	mockAppDownloader.On("Download", mock.Anything).Return(zipBytes, nil)
	app.Downloader = &mockAppDownloader

	expectedDir := filepath.Join(app.GetOdysseygoBinDir(), odysseygoBinPrefix+version1)

	binDir, err := installBinaryWithVersion(app, version1, app.GetOdysseygoBinDir(), odysseygoBinPrefix, githubDownloader, mockInstaller)
	require.Equal(expectedDir, binDir)
	require.NoError(err)

	// Check the installed binary
	installedBin, err := os.ReadFile(filepath.Join(binDir, odysseygoBin))
	require.NoError(err)
	require.Equal(binary1, installedBin)
}

func Test_installOdysseyGoWithVersion_Tar(t *testing.T) {
	require := testutils.SetupTest(t)

	tarBytes := testutils.CreateDummyOdygoTar(require, binary1, version1)

	app := setupInstallDir(require)

	mockInstaller := &mocks.Installer{}
	mockInstaller.On("GetArch").Return("amd64", "linux")

	downloader := NewOdygoDownloader()

	mockAppDownloader := mocks.Downloader{}
	mockAppDownloader.On("Download", mock.Anything).Return(tarBytes, nil)
	app.Downloader = &mockAppDownloader

	expectedDir := filepath.Join(app.GetOdysseygoBinDir(), odysseygoBinPrefix+version1)

	binDir, err := installBinaryWithVersion(app, version1, app.GetOdysseygoBinDir(), odysseygoBinPrefix, downloader, mockInstaller)
	require.Equal(expectedDir, binDir)
	require.NoError(err)

	// Check the installed binary
	installedBin, err := os.ReadFile(filepath.Join(binDir, odysseygoBin))
	require.NoError(err)
	require.Equal(binary1, installedBin)
}

func Test_installOdysseyGoWithVersion_MultipleCoinstalls(t *testing.T) {
	require := testutils.SetupTest(t)

	zipBytes1 := testutils.CreateDummyOdygoZip(require, binary1)
	zipBytes2 := testutils.CreateDummyOdygoZip(require, binary2)
	app := setupInstallDir(require)

	mockInstaller := &mocks.Installer{}
	mockInstaller.On("GetArch").Return("amd64", "darwin")

	downloader := NewOdygoDownloader()
	url1, _, err := downloader.GetDownloadURL(version1, mockInstaller)
	require.NoError(err)
	url2, _, err := downloader.GetDownloadURL(version2, mockInstaller)
	require.NoError(err)
	mockInstaller.On("DownloadRelease", url1).Return(zipBytes1, nil)
	mockInstaller.On("DownloadRelease", url2).Return(zipBytes2, nil)

	mockAppDownloader := mocks.Downloader{}
	mockAppDownloader.On("Download", url1).Return(zipBytes1, nil)
	mockAppDownloader.On("Download", url2).Return(zipBytes2, nil)
	app.Downloader = &mockAppDownloader

	expectedDir1 := filepath.Join(app.GetOdysseygoBinDir(), odysseygoBinPrefix+version1)
	expectedDir2 := filepath.Join(app.GetOdysseygoBinDir(), odysseygoBinPrefix+version2)

	binDir1, err := installBinaryWithVersion(app, version1, app.GetOdysseygoBinDir(), odysseygoBinPrefix, downloader, mockInstaller)
	require.Equal(expectedDir1, binDir1)
	require.NoError(err)

	binDir2, err := installBinaryWithVersion(app, version2, app.GetOdysseygoBinDir(), odysseygoBinPrefix, downloader, mockInstaller)
	require.Equal(expectedDir2, binDir2)
	require.NoError(err)

	require.NotEqual(binDir1, binDir2)

	// Check the installed binary
	installedBin1, err := os.ReadFile(filepath.Join(binDir1, odysseygoBin))
	require.NoError(err)
	require.Equal(binary1, installedBin1)

	installedBin2, err := os.ReadFile(filepath.Join(binDir2, odysseygoBin))
	require.NoError(err)
	require.Equal(binary2, installedBin2)
}

func Test_installSubnetEVMWithVersion(t *testing.T) {
	require := testutils.SetupTest(t)

	tarBytes := testutils.CreateDummySubnetEVMTar(require, binary1)
	app := setupInstallDir(require)

	mockInstaller := &mocks.Installer{}
	mockInstaller.On("GetArch").Return("amd64", "darwin")

	downloader := NewSubnetEVMDownloader()

	mockAppDownloader := mocks.Downloader{}
	mockAppDownloader.On("Download", mock.Anything).Return(tarBytes, nil)
	app.Downloader = &mockAppDownloader

	expectedDir := filepath.Join(app.GetSubnetEVMBinDir(), subnetEVMBinPrefix+version1)

	subDir := filepath.Join(app.GetSubnetEVMBinDir(), subnetEVMBinPrefix+version1)

	binDir, err := installBinaryWithVersion(app, version1, subDir, subnetEVMBinPrefix, downloader, mockInstaller)
	require.Equal(expectedDir, binDir)
	require.NoError(err)

	// Check the installed binary
	installedBin, err := os.ReadFile(filepath.Join(binDir, constants.SubnetEVMBin))
	require.NoError(err)
	require.Equal(binary1, installedBin)
}

func Test_installSubnetEVMWithVersion_MultipleCoinstalls(t *testing.T) {
	require := testutils.SetupTest(t)

	tarBytes1 := testutils.CreateDummySubnetEVMTar(require, binary1)
	tarBytes2 := testutils.CreateDummySubnetEVMTar(require, binary2)
	app := setupInstallDir(require)

	mockInstaller := &mocks.Installer{}
	mockInstaller.On("GetArch").Return("arm64", "linux")

	downloader := NewSubnetEVMDownloader()
	url1, _, err := downloader.GetDownloadURL(version1, mockInstaller)
	require.NoError(err)
	url2, _, err := downloader.GetDownloadURL(version2, mockInstaller)
	require.NoError(err)

	mockAppDownloader := mocks.Downloader{}
	mockAppDownloader.On("Download", url1).Return(tarBytes1, nil)
	mockAppDownloader.On("Download", url2).Return(tarBytes2, nil)
	app.Downloader = &mockAppDownloader

	expectedDir1 := filepath.Join(app.GetSubnetEVMBinDir(), subnetEVMBinPrefix+version1)
	expectedDir2 := filepath.Join(app.GetSubnetEVMBinDir(), subnetEVMBinPrefix+version2)

	subDir1 := filepath.Join(app.GetSubnetEVMBinDir(), subnetEVMBinPrefix+version1)
	subDir2 := filepath.Join(app.GetSubnetEVMBinDir(), subnetEVMBinPrefix+version2)

	binDir1, err := installBinaryWithVersion(app, version1, subDir1, subnetEVMBinPrefix, downloader, mockInstaller)
	require.Equal(expectedDir1, binDir1)
	require.NoError(err)

	binDir2, err := installBinaryWithVersion(app, version2, subDir2, subnetEVMBinPrefix, downloader, mockInstaller)
	require.Equal(expectedDir2, binDir2)
	require.NoError(err)

	require.NotEqual(binDir1, binDir2)

	// Check the installed binary
	installedBin1, err := os.ReadFile(filepath.Join(binDir1, constants.SubnetEVMBin))
	require.NoError(err)
	require.Equal(binary1, installedBin1)

	installedBin2, err := os.ReadFile(filepath.Join(binDir2, constants.SubnetEVMBin))
	require.NoError(err)
	require.Equal(binary2, installedBin2)
}
