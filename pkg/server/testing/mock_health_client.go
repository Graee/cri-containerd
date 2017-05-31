/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Automatically generated by MockGen. DO NOT EDIT!
// Source: google.golang.org/grpc/health/grpc_health_v1 (interfaces: HealthClient)

package testing

import (
	gomock "github.com/golang/mock/gomock"
	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
)

// Mock of HealthClient interface
type MockHealthClient struct {
	ctrl     *gomock.Controller
	recorder *_MockHealthClientRecorder
}

// Recorder for MockHealthClient (not exported)
type _MockHealthClientRecorder struct {
	mock *MockHealthClient
}

func NewMockHealthClient(ctrl *gomock.Controller) *MockHealthClient {
	mock := &MockHealthClient{ctrl: ctrl}
	mock.recorder = &_MockHealthClientRecorder{mock}
	return mock
}

func (_m *MockHealthClient) EXPECT() *_MockHealthClientRecorder {
	return _m.recorder
}

func (_m *MockHealthClient) Check(_param0 context.Context, _param1 *grpc_health_v1.HealthCheckRequest, _param2 ...grpc.CallOption) (*grpc_health_v1.HealthCheckResponse, error) {
	_s := []interface{}{_param0, _param1}
	for _, _x := range _param2 {
		_s = append(_s, _x)
	}
	ret := _m.ctrl.Call(_m, "Check", _s...)
	ret0, _ := ret[0].(*grpc_health_v1.HealthCheckResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockHealthClientRecorder) Check(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	_s := append([]interface{}{arg0, arg1}, arg2...)
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Check", _s...)
}
