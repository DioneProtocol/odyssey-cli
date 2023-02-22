// Code generated by mockery v2.20.0. DO NOT EDIT.

package mocks

import mock "github.com/stretchr/testify/mock"

// Installer is an autogenerated mock type for the Installer type
type Installer struct {
	mock.Mock
}

// GetArch provides a mock function with given fields:
func (_m *Installer) GetArch() (string, string) {
	ret := _m.Called()

	var r0 string
	var r1 string
	if rf, ok := ret.Get(0).(func() (string, string)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	if rf, ok := ret.Get(1).(func() string); ok {
		r1 = rf()
	} else {
		r1 = ret.Get(1).(string)
	}

	return r0, r1
}

type mockConstructorTestingTNewInstaller interface {
	mock.TestingT
	Cleanup(func())
}

// NewInstaller creates a new instance of Installer. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewInstaller(t mockConstructorTestingTNewInstaller) *Installer {
	mock := &Installer{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
