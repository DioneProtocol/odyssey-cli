// Copyright (C) 2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package binutils

import (
	"fmt"

	"github.com/DioneProtocol/odyssey-cli/pkg/constants"
)

const (
	linux   = "linux"
	darwin  = "darwin"
	windows = "windows"

	zipExtension = "zip"
	tarExtension = "tar.gz"
)

type GithubDownloader interface {
	GetDownloadURL(version string, installer Installer) (string, string, error)
}

type (
	subnetEVMDownloader struct{}
	odysseyGoDownloader struct{}
)

var (
	_ GithubDownloader = (*subnetEVMDownloader)(nil)
	_ GithubDownloader = (*odysseyGoDownloader)(nil)
)

func GetGithubLatestReleaseURL(org, repo string) string {
	return "https://api.github.com/repos/" + org + "/" + repo + "/releases/latest"
}

func NewOdygoDownloader() GithubDownloader {
	return &odysseyGoDownloader{}
}

func (odysseyGoDownloader) GetDownloadURL(version string, installer Installer) (string, string, error) {
	// NOTE: if any of the underlying URLs change (github changes, release file names, etc.) this fails
	goarch, goos := installer.GetArch()

	var odysseygoURL string
	var ext string

	switch goos {
	case linux:
		odysseygoURL = fmt.Sprintf(
			"https://github.com/%s/%s/releases/download/%s/odysseygo-linux-%s-%s.tar.gz",
			constants.DioneProtocolOrg,
			constants.OdysseyGoRepoName,
			version,
			goarch,
			version,
		)
		ext = tarExtension
	case darwin:
		odysseygoURL = fmt.Sprintf(
			"https://github.com/%s/%s/releases/download/%s/odysseygo-macos-%s.zip",
			constants.DioneProtocolOrg,
			constants.OdysseyGoRepoName,
			version,
			version,
		)
		ext = zipExtension
		// EXPERIMENTAL WIN, no support
	case windows:
		odysseygoURL = fmt.Sprintf(
			"https://github.com/%s/%s/releases/download/%s/odysseygo-win-%s-experimental.zip",
			constants.DioneProtocolOrg,
			constants.OdysseyGoRepoName,
			version,
			version,
		)
		ext = zipExtension
	default:
		return "", "", fmt.Errorf("OS not supported: %s", goos)
	}

	return odysseygoURL, ext, nil
}

func NewSubnetEVMDownloader() GithubDownloader {
	return &subnetEVMDownloader{}
}

func (subnetEVMDownloader) GetDownloadURL(version string, installer Installer) (string, string, error) {
	// NOTE: if any of the underlying URLs change (github changes, release file names, etc.) this fails
	goarch, goos := installer.GetArch()

	var subnetEVMURL string
	ext := tarExtension

	switch goos {
	case linux:
		subnetEVMURL = fmt.Sprintf(
			"https://github.com/%s/%s/releases/download/%s/%s_%s_linux_%s.tar.gz",
			constants.DioneProtocolOrg,
			constants.SubnetEVMRepoName,
			version,
			constants.SubnetEVMRepoName,
			version[1:], // WARN subnet-evm isn't consistent in its release naming, it's omitting the v in the file name...
			goarch,
		)
	case darwin:
		subnetEVMURL = fmt.Sprintf(
			"https://github.com/%s/%s/releases/download/%s/%s_%s_darwin_%s.tar.gz",
			constants.DioneProtocolOrg,
			constants.SubnetEVMRepoName,
			version,
			constants.SubnetEVMRepoName,
			version[1:],
			goarch,
		)
	default:
		return "", "", fmt.Errorf("OS not supported: %s", goos)
	}

	return subnetEVMURL, ext, nil
}
