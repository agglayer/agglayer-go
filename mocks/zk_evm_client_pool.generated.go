// Code generated by mockery v2.38.0. DO NOT EDIT.

package mocks

import (
	types "github.com/0xPolygon/agglayer/types"
	mock "github.com/stretchr/testify/mock"
)

// ZkEVMClientCacheMock is an autogenerated mock type for the IZkEVMClientClientCreator type
type ZkEVMClientCacheMock struct {
	mock.Mock
}

type ZkEVMClientClientCreatorMock_Expecter struct {
	mock *mock.Mock
}

func (_m *ZkEVMClientCacheMock) EXPECT() *ZkEVMClientClientCreatorMock_Expecter {
	return &ZkEVMClientClientCreatorMock_Expecter{mock: &_m.Mock}
}

// GetClient provides a mock function with given fields: rpc
func (_m *ZkEVMClientCacheMock) GetClient(rpc string) types.IZkEVMClient {
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

// ZkEVMClientClientCreatorMock_NewClient_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'NewClient'
type ZkEVMClientClientCreatorMock_NewClient_Call struct {
	*mock.Call
}

// NewClient is a helper method to define mock.On call
//   - rpc string
func (_e *ZkEVMClientClientCreatorMock_Expecter) NewClient(rpc interface{}) *ZkEVMClientClientCreatorMock_NewClient_Call {
	return &ZkEVMClientClientCreatorMock_NewClient_Call{Call: _e.mock.On("NewClient", rpc)}
}

func (_c *ZkEVMClientClientCreatorMock_NewClient_Call) Run(run func(rpc string)) *ZkEVMClientClientCreatorMock_NewClient_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string))
	})
	return _c
}

func (_c *ZkEVMClientClientCreatorMock_NewClient_Call) Return(_a0 types.IZkEVMClient) *ZkEVMClientClientCreatorMock_NewClient_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *ZkEVMClientClientCreatorMock_NewClient_Call) RunAndReturn(run func(string) types.IZkEVMClient) *ZkEVMClientClientCreatorMock_NewClient_Call {
	_c.Call.Return(run)
	return _c
}

// NewZkEVMClientClientCreatorMock creates a new instance of ZkEVMClientClientCreatorMock. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewZkEVMClientClientCreatorMock(t interface {
	mock.TestingT
	Cleanup(func())
}) *ZkEVMClientCacheMock {
	mock := &ZkEVMClientCacheMock{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
