// Code generated by mockery v2.24.0. DO NOT EDIT.

package mocks

import (
	application "github.com/ava-labs/avalanche-cli/pkg/application"

	mock "github.com/stretchr/testify/mock"
)

// ProcessChecker is an autogenerated mock type for the ProcessChecker type
type ProcessChecker struct {
	mock.Mock
}

// IsServerProcessRunning provides a mock function with given fields: app
func (_m *ProcessChecker) IsServerProcessRunning(app *application.Avalanche) (bool, error) {
	ret := _m.Called(app)

	var r0 bool
	var r1 error
	if rf, ok := ret.Get(0).(func(*application.Avalanche) (bool, error)); ok {
		return rf(app)
	}
	if rf, ok := ret.Get(0).(func(*application.Avalanche) bool); ok {
		r0 = rf(app)
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func(*application.Avalanche) error); ok {
		r1 = rf(app)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type mockConstructorTestingTNewProcessChecker interface {
	mock.TestingT
	Cleanup(func())
}

// NewProcessChecker creates a new instance of ProcessChecker. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewProcessChecker(t mockConstructorTestingTNewProcessChecker) *ProcessChecker {
	mock := &ProcessChecker{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
