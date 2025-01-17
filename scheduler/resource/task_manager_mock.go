// Code generated by MockGen. DO NOT EDIT.
// Source: scheduler/resource/task_manager.go

// Package resource is a generated GoMock package.
package resource

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
)

// MockTaskManager is a mock of TaskManager interface.
type MockTaskManager struct {
	ctrl     *gomock.Controller
	recorder *MockTaskManagerMockRecorder
}

// MockTaskManagerMockRecorder is the mock recorder for MockTaskManager.
type MockTaskManagerMockRecorder struct {
	mock *MockTaskManager
}

// NewMockTaskManager creates a new mock instance.
func NewMockTaskManager(ctrl *gomock.Controller) *MockTaskManager {
	mock := &MockTaskManager{ctrl: ctrl}
	mock.recorder = &MockTaskManagerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockTaskManager) EXPECT() *MockTaskManagerMockRecorder {
	return m.recorder
}

// Delete mocks base method.
func (m *MockTaskManager) Delete(arg0 string) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Delete", arg0)
}

// Delete indicates an expected call of Delete.
func (mr *MockTaskManagerMockRecorder) Delete(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockTaskManager)(nil).Delete), arg0)
}

// Load mocks base method.
func (m *MockTaskManager) Load(arg0 string) (*Task, bool) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Load", arg0)
	ret0, _ := ret[0].(*Task)
	ret1, _ := ret[1].(bool)
	return ret0, ret1
}

// Load indicates an expected call of Load.
func (mr *MockTaskManagerMockRecorder) Load(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Load", reflect.TypeOf((*MockTaskManager)(nil).Load), arg0)
}

// LoadOrStore mocks base method.
func (m *MockTaskManager) LoadOrStore(arg0 *Task) (*Task, bool) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "LoadOrStore", arg0)
	ret0, _ := ret[0].(*Task)
	ret1, _ := ret[1].(bool)
	return ret0, ret1
}

// LoadOrStore indicates an expected call of LoadOrStore.
func (mr *MockTaskManagerMockRecorder) LoadOrStore(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "LoadOrStore", reflect.TypeOf((*MockTaskManager)(nil).LoadOrStore), arg0)
}

// RunGC mocks base method.
func (m *MockTaskManager) RunGC() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RunGC")
	ret0, _ := ret[0].(error)
	return ret0
}

// RunGC indicates an expected call of RunGC.
func (mr *MockTaskManagerMockRecorder) RunGC() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RunGC", reflect.TypeOf((*MockTaskManager)(nil).RunGC))
}

// Store mocks base method.
func (m *MockTaskManager) Store(arg0 *Task) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Store", arg0)
}

// Store indicates an expected call of Store.
func (mr *MockTaskManagerMockRecorder) Store(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Store", reflect.TypeOf((*MockTaskManager)(nil).Store), arg0)
}
