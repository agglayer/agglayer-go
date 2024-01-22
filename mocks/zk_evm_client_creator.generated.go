// Code generated by mockery v2.38.0. DO NOT EDIT.

package mocks

import (
	types "github.com/0xPolygon/beethoven/types"
	mock "github.com/stretchr/testify/mock"
)

// ZkEVMClientClientCreatorMock is an autogenerated mock type for the IZkEVMClientClientCreator type
type ZkEVMClientClientCreatorMock struct {
	mock.Mock
}

// NewClient provides a mock function with given fields: rpc
func (_m *ZkEVMClientClientCreatorMock) NewClient(rpc string) types.IZkEVMClient {
	ret := _m.Called(rpc)

	if len(ret) == 0 {
		panic("no return value specified for NewClient")
	}

	var r0 types.IZkEVMClient
	if rf, ok := ret.Get(0).(func(string) types.IZkEVMClient); ok {
		r0 = rf(rpc)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(types.IZkEVMClient)
		}
	}

	return r0
}

// NewZkEVMClientClientCreatorMock creates a new instance of ZkEVMClientClientCreatorMock. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewZkEVMClientClientCreatorMock(t interface {
	mock.TestingT
	Cleanup(func())
}) *ZkEVMClientClientCreatorMock {
	mock := &ZkEVMClientClientCreatorMock{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}