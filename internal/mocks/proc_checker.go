// Code generated by mockery v2.10.6. DO NOT EDIT.

package mocks

import (
 "github.com/ava-labs/avalanche-cli/pkg/application"

	mock "github.com/stretchr/testify/mock"
)

// ProcessChecker is an autogenerated mock type for the ProcessChecker type
type ProcessChecker struct {
	mock.Mock
}

// IsServerProcessRunning provides a mock function with given fields: _a0
func (_m *ProcessChecker) IsServerProcessRunning(_a0 *application.Avalanche) (bool, error) {
	ret := _m.Called(_a0)

	var r0 bool
	if rf, ok := ret.Get(0).(func(*application.Avalanche) bool); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*application.Avalanche) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
