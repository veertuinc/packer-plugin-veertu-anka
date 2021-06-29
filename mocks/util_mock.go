// Code generated by MockGen. DO NOT EDIT.
// Source: util/util.go

// Package mocks is a generated GoMock package.
package mocks

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	multistep "github.com/hashicorp/packer-plugin-sdk/multistep"
	packer "github.com/hashicorp/packer-plugin-sdk/packer"
	util "github.com/veertuinc/packer-plugin-veertu-anka/util"
)

// MockUtil is a mock of Util interface.
type MockUtil struct {
	ctrl     *gomock.Controller
	recorder *MockUtilMockRecorder
}

// MockUtilMockRecorder is the mock recorder for MockUtil.
type MockUtilMockRecorder struct {
	mock *MockUtil
}

// NewMockUtil creates a new mock instance.
func NewMockUtil(ctrl *gomock.Controller) *MockUtil {
	mock := &MockUtil{ctrl: ctrl}
	mock.recorder = &MockUtilMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockUtil) EXPECT() *MockUtilMockRecorder {
	return m.recorder
}

// ConfigTmpDir mocks base method.
func (m *MockUtil) ConfigTmpDir() (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ConfigTmpDir")
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ConfigTmpDir indicates an expected call of ConfigTmpDir.
func (mr *MockUtilMockRecorder) ConfigTmpDir() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ConfigTmpDir", reflect.TypeOf((*MockUtil)(nil).ConfigTmpDir))
}

// ConvertDiskSizeToBytes mocks base method.
func (m *MockUtil) ConvertDiskSizeToBytes(diskSize string) (uint64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ConvertDiskSizeToBytes", diskSize)
	ret0, _ := ret[0].(uint64)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ConvertDiskSizeToBytes indicates an expected call of ConvertDiskSizeToBytes.
func (mr *MockUtilMockRecorder) ConvertDiskSizeToBytes(diskSize interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ConvertDiskSizeToBytes", reflect.TypeOf((*MockUtil)(nil).ConvertDiskSizeToBytes), diskSize)
}

// ObtainMacOSVersionFromInstallerApp mocks base method.
func (m *MockUtil) ObtainMacOSVersionFromInstallerApp(path string) (util.InstallAppPlist, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ObtainMacOSVersionFromInstallerApp", path)
	ret0, _ := ret[0].(util.InstallAppPlist)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ObtainMacOSVersionFromInstallerApp indicates an expected call of ObtainMacOSVersionFromInstallerApp.
func (mr *MockUtilMockRecorder) ObtainMacOSVersionFromInstallerApp(path interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ObtainMacOSVersionFromInstallerApp", reflect.TypeOf((*MockUtil)(nil).ObtainMacOSVersionFromInstallerApp), path)
}

// RandSeq mocks base method.
func (m *MockUtil) RandSeq(n int) string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RandSeq", n)
	ret0, _ := ret[0].(string)
	return ret0
}

// RandSeq indicates an expected call of RandSeq.
func (mr *MockUtilMockRecorder) RandSeq(n interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RandSeq", reflect.TypeOf((*MockUtil)(nil).RandSeq), n)
}

// StepError mocks base method.
func (m *MockUtil) StepError(ui packer.Ui, state multistep.StateBag, err error) multistep.StepAction {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "StepError", ui, state, err)
	ret0, _ := ret[0].(multistep.StepAction)
	return ret0
}

// StepError indicates an expected call of StepError.
func (mr *MockUtilMockRecorder) StepError(ui, state, err interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "StepError", reflect.TypeOf((*MockUtil)(nil).StepError), ui, state, err)
}
