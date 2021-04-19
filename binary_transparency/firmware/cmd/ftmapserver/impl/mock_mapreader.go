// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/google/trillian-examples/binary_transparency/firmware/cmd/ftmapserver/impl (interfaces: MapReader)

package impl

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	api "github.com/google/trillian-examples/binary_transparency/firmware/api"
	batchmap "github.com/google/trillian/experimental/batchmap"
	types "github.com/google/trillian/types"
)

// MockMapReader is a mock of MapReader interface.
type MockMapReader struct {
	ctrl     *gomock.Controller
	recorder *MockMapReaderMockRecorder
}

// MockMapReaderMockRecorder is the mock recorder for MockMapReader.
type MockMapReaderMockRecorder struct {
	mock *MockMapReader
}

// NewMockMapReader creates a new mock instance.
func NewMockMapReader(ctrl *gomock.Controller) *MockMapReader {
	mock := &MockMapReader{ctrl: ctrl}
	mock.recorder = &MockMapReaderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockMapReader) EXPECT() *MockMapReaderMockRecorder {
	return m.recorder
}

// Aggregation mocks base method.
func (m *MockMapReader) Aggregation(arg0 int, arg1 uint64) (api.AggregatedFirmware, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Aggregation", arg0, arg1)
	ret0, _ := ret[0].(api.AggregatedFirmware)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Aggregation indicates an expected call of Aggregation.
func (mr *MockMapReaderMockRecorder) Aggregation(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Aggregation", reflect.TypeOf((*MockMapReader)(nil).Aggregation), arg0, arg1)
}

// LatestRevision mocks base method.
func (m *MockMapReader) LatestRevision() (int, types.LogRootV1, int64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "LatestRevision")
	ret0, _ := ret[0].(int)
	ret1, _ := ret[1].(types.LogRootV1)
	ret2, _ := ret[2].(int64)
	ret3, _ := ret[3].(error)
	return ret0, ret1, ret2, ret3
}

// LatestRevision indicates an expected call of LatestRevision.
func (mr *MockMapReaderMockRecorder) LatestRevision() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "LatestRevision", reflect.TypeOf((*MockMapReader)(nil).LatestRevision))
}

// Tile mocks base method.
func (m *MockMapReader) Tile(arg0 int, arg1 []byte) (*batchmap.Tile, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Tile", arg0, arg1)
	ret0, _ := ret[0].(*batchmap.Tile)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Tile indicates an expected call of Tile.
func (mr *MockMapReaderMockRecorder) Tile(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Tile", reflect.TypeOf((*MockMapReader)(nil).Tile), arg0, arg1)
}
