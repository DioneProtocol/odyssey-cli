// Copyright (C) 2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package models

import (
	"testing"

	"github.com/DioneProtocol/odyssey-network-runner/utils"
	"github.com/stretchr/testify/require"
)

func TestGetVMID_imported(t *testing.T) {
	assert := require.New(t)
	testVMID := "abcd"
	sc := Sidecar{
		ImportedFromOPM: true,
		ImportedVMID:    testVMID,
	}

	vmid, err := sc.GetVMID()
	assert.NoError(err)
	assert.Equal(testVMID, vmid)
}

func TestGetVMID_derived(t *testing.T) {
	assert := require.New(t)
	testVMName := "subnet"
	sc := Sidecar{
		ImportedFromOPM: false,
		Name:            testVMName,
	}

	expectedVMID, err := utils.VMID(testVMName)
	assert.NoError(err)

	vmid, err := sc.GetVMID()
	assert.NoError(err)
	assert.Equal(expectedVMID.String(), vmid)
}
