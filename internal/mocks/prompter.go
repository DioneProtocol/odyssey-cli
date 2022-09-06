// Code generated by mockery v2.14.0. DO NOT EDIT.

package mocks

import (
	big "math/big"

	ids "github.com/ava-labs/avalanchego/ids"
	common "github.com/ethereum/go-ethereum/common"

	mock "github.com/stretchr/testify/mock"

	models "github.com/ava-labs/avalanche-cli/pkg/models"

	time "time"

	url "net/url"
)

// Prompter is an autogenerated mock type for the Prompter type
type Prompter struct {
	mock.Mock
}

// CaptureAddress provides a mock function with given fields: promptStr
func (_m *Prompter) CaptureAddress(promptStr string) (common.Address, error) {
	ret := _m.Called(promptStr)

	var r0 common.Address
	if rf, ok := ret.Get(0).(func(string) common.Address); ok {
		r0 = rf(promptStr)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(common.Address)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(promptStr)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CaptureAnyList provides a mock function with given fields: promptStr, options
func (_m *Prompter) CaptureAnyList(promptStr string, options interface{}) (interface{}, error) {
	ret := _m.Called(promptStr, options)

	var r0 interface{}
	if rf, ok := ret.Get(0).(func(string, interface{}) interface{}); ok {
		r0 = rf(promptStr, options)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(interface{})
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, interface{}) error); ok {
		r1 = rf(promptStr, options)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CaptureDate provides a mock function with given fields: promptStr
func (_m *Prompter) CaptureDate(promptStr string) (time.Time, error) {
	ret := _m.Called(promptStr)

	var r0 time.Time
	if rf, ok := ret.Get(0).(func(string) time.Time); ok {
		r0 = rf(promptStr)
	} else {
		r0 = ret.Get(0).(time.Time)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(promptStr)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CaptureDuration provides a mock function with given fields: promptStr
func (_m *Prompter) CaptureDuration(promptStr string) (time.Duration, error) {
	ret := _m.Called(promptStr)

	var r0 time.Duration
	if rf, ok := ret.Get(0).(func(string) time.Duration); ok {
		r0 = rf(promptStr)
	} else {
		r0 = ret.Get(0).(time.Duration)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(promptStr)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CaptureExistingFilepath provides a mock function with given fields: promptStr
func (_m *Prompter) CaptureExistingFilepath(promptStr string) (string, error) {
	ret := _m.Called(promptStr)

	var r0 string
	if rf, ok := ret.Get(0).(func(string) string); ok {
		r0 = rf(promptStr)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(promptStr)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CaptureGitURL provides a mock function with given fields: promptStr
func (_m *Prompter) CaptureGitURL(promptStr string) (url.URL, error) {
	ret := _m.Called(promptStr)

	var r0 url.URL
	if rf, ok := ret.Get(0).(func(string) url.URL); ok {
		r0 = rf(promptStr)
	} else {
		r0 = ret.Get(0).(url.URL)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(promptStr)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CaptureIndex provides a mock function with given fields: promptStr, options
func (_m *Prompter) CaptureIndex(promptStr string, options []interface{}) (int, error) {
	ret := _m.Called(promptStr, options)

	var r0 int
	if rf, ok := ret.Get(0).(func(string, []interface{}) int); ok {
		r0 = rf(promptStr, options)
	} else {
		r0 = ret.Get(0).(int)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, []interface{}) error); ok {
		r1 = rf(promptStr, options)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CaptureList provides a mock function with given fields: promptStr, options
func (_m *Prompter) CaptureList(promptStr string, options []string) (string, error) {
	ret := _m.Called(promptStr, options)

	var r0 string
	if rf, ok := ret.Get(0).(func(string, []string) string); ok {
		r0 = rf(promptStr, options)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, []string) error); ok {
		r1 = rf(promptStr, options)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CaptureNoYes provides a mock function with given fields: promptStr
func (_m *Prompter) CaptureNoYes(promptStr string) (bool, error) {
	ret := _m.Called(promptStr)

	var r0 bool
	if rf, ok := ret.Get(0).(func(string) bool); ok {
		r0 = rf(promptStr)
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(promptStr)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CaptureNodeID provides a mock function with given fields: promptStr
func (_m *Prompter) CaptureNodeID(promptStr string) (ids.NodeID, error) {
	ret := _m.Called(promptStr)

	var r0 ids.NodeID
	if rf, ok := ret.Get(0).(func(string) ids.NodeID); ok {
		r0 = rf(promptStr)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(ids.NodeID)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(promptStr)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CapturePChainAddress provides a mock function with given fields: promptStr, network
func (_m *Prompter) CapturePChainAddress(promptStr string, network models.Network) (string, error) {
	ret := _m.Called(promptStr, network)

	var r0 string
	if rf, ok := ret.Get(0).(func(string, models.Network) string); ok {
		r0 = rf(promptStr, network)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, models.Network) error); ok {
		r1 = rf(promptStr, network)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CapturePositiveBigInt provides a mock function with given fields: promptStr
func (_m *Prompter) CapturePositiveBigInt(promptStr string) (*big.Int, error) {
	ret := _m.Called(promptStr)

	var r0 *big.Int
	if rf, ok := ret.Get(0).(func(string) *big.Int); ok {
		r0 = rf(promptStr)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*big.Int)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(promptStr)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CaptureString provides a mock function with given fields: promptStr
func (_m *Prompter) CaptureString(promptStr string) (string, error) {
	ret := _m.Called(promptStr)

	var r0 string
	if rf, ok := ret.Get(0).(func(string) string); ok {
		r0 = rf(promptStr)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(promptStr)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CaptureUint64 provides a mock function with given fields: promptStr
func (_m *Prompter) CaptureUint64(promptStr string) (uint64, error) {
	ret := _m.Called(promptStr)

	var r0 uint64
	if rf, ok := ret.Get(0).(func(string) uint64); ok {
		r0 = rf(promptStr)
	} else {
		r0 = ret.Get(0).(uint64)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(promptStr)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CaptureVersion provides a mock function with given fields: promptStr
func (_m *Prompter) CaptureVersion(promptStr string) (string, error) {
	ret := _m.Called(promptStr)

	var r0 string
	if rf, ok := ret.Get(0).(func(string) string); ok {
		r0 = rf(promptStr)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(promptStr)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CaptureWeight provides a mock function with given fields: promptStr
func (_m *Prompter) CaptureWeight(promptStr string) (uint64, error) {
	ret := _m.Called(promptStr)

	var r0 uint64
	if rf, ok := ret.Get(0).(func(string) uint64); ok {
		r0 = rf(promptStr)
	} else {
		r0 = ret.Get(0).(uint64)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(promptStr)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CaptureYesNo provides a mock function with given fields: promptStr
func (_m *Prompter) CaptureYesNo(promptStr string) (bool, error) {
	ret := _m.Called(promptStr)

	var r0 bool
	if rf, ok := ret.Get(0).(func(string) bool); ok {
		r0 = rf(promptStr)
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(promptStr)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type mockConstructorTestingTNewPrompter interface {
	mock.TestingT
	Cleanup(func())
}

// NewPrompter creates a new instance of Prompter. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewPrompter(t mockConstructorTestingTNewPrompter) *Prompter {
	mock := &Prompter{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
