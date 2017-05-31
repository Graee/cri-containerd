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
// Source: github.com/containerd/containerd/api/services/version (interfaces: VersionClient)

package testing

import (
	version "github.com/containerd/containerd/api/services/version"
	gomock "github.com/golang/mock/gomock"
	empty "github.com/golang/protobuf/ptypes/empty"
	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
)

// Mock of VersionClient interface
type MockVersionClient struct {
	ctrl     *gomock.Controller
	recorder *_MockVersionClientRecorder
}

// Recorder for MockVersionClient (not exported)
type _MockVersionClientRecorder struct {
	mock *MockVersionClient
}

func NewMockVersionClient(ctrl *gomock.Controller) *MockVersionClient {
	mock := &MockVersionClient{ctrl: ctrl}
	mock.recorder = &_MockVersionClientRecorder{mock}
	return mock
}

func (_m *MockVersionClient) EXPECT() *_MockVersionClientRecorder {
	return _m.recorder
}

func (_m *MockVersionClient) Version(_param0 context.Context, _param1 *empty.Empty, _param2 ...grpc.CallOption) (*version.VersionResponse, error) {
	_s := []interface{}{_param0, _param1}
	for _, _x := range _param2 {
		_s = append(_s, _x)
	}
	ret := _m.ctrl.Call(_m, "Version", _s...)
	ret0, _ := ret[0].(*version.VersionResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockVersionClientRecorder) Version(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	_s := append([]interface{}{arg0, arg1}, arg2...)
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Version", _s...)
}
