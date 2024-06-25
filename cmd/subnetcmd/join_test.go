// Copyright (C) 2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package subnetcmd

import (
	"testing"

	"github.com/DioneProtocol/odyssey-cli/internal/mocks"
	"github.com/DioneProtocol/odysseygo/ids"
	"github.com/DioneProtocol/odysseygo/vms/omegavm"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestIsNodeValidatingSubnet(t *testing.T) {
	require := require.New(t)
	nodeID := ids.GenerateTestNodeID()
	nonValidator := ids.GenerateTestNodeID()
	subnetID := ids.GenerateTestID()

	oClient := &mocks.OClient{}
	oClient.On("GetCurrentValidators", mock.Anything, mock.Anything, mock.Anything).Return(
		[]omegavm.ClientPermissionlessValidator{
			{
				ClientStaker: omegavm.ClientStaker{
					NodeID: nodeID,
				},
			},
		}, nil)

	oClient.On("GetPendingValidators", mock.Anything, mock.Anything, mock.Anything).Return(
		[]interface{}{}, nil, nil).Once()

	interfaceReturn := make([]interface{}, 1)
	val := map[string]interface{}{
		"nodeID": nonValidator.String(),
	}
	interfaceReturn[0] = val
	oClient.On("GetPendingValidators", mock.Anything, mock.Anything, mock.Anything).Return(interfaceReturn, nil, nil)

	// first pass: should return true for the GetCurrentValidators
	isValidating, err := checkIsValidating(subnetID, nodeID, oClient)
	require.NoError(err)
	require.True(isValidating)

	// second pass: The nonValidator is not in current nor pending validators, hence false
	isValidating, err = checkIsValidating(subnetID, nonValidator, oClient)
	require.NoError(err)
	require.False(isValidating)

	// third pass: The second mocked GetPendingValidators applies, and this time
	// nonValidator is in the pending set, hence true
	isValidating, err = checkIsValidating(subnetID, nonValidator, oClient)
	require.NoError(err)
	require.True(isValidating)
}
