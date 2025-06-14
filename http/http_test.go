package http

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// MockStore implements the minimal methods needed for testing.
type MockStore struct {
	ApplyErr          error
	GetValueExists    bool
	GetValue          interface{}
	AddFollowerErr    error
	RemoveFollowerErr error
}

func (m *MockStore) Apply([]byte) error                 { return m.ApplyErr }
func (m *MockStore) Get(key string) (interface{}, bool) { return m.GetValue, m.GetValueExists }
func (m *MockStore) AddFollower(id, addr string) error  { return m.AddFollowerErr }
func (m *MockStore) RemoveFollower(id string) error     { return m.RemoveFollowerErr }

func TestApplyHandler_OnlyPost(t *testing.T) {
	s := &Server{store: &MockStore{}}

	req := httptest.NewRequest(http.MethodGet, "/apply", nil)
	w := httptest.NewRecorder()
	s.applyHandler(w, req)
	if w.Result().StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 Method Not Allowed, got %d", w.Result().StatusCode)
	}

	req = httptest.NewRequest(http.MethodPost, "/apply", nil)
	w = httptest.NewRecorder()
	s.applyHandler(w, req)
	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", w.Result().StatusCode)
	}
}

func TestApplyHandler_Success(t *testing.T) {
	s := &Server{store: &MockStore{}}
	req := httptest.NewRequest(http.MethodPost, "/apply", strings.NewReader("test"))
	w := httptest.NewRecorder()
	s.applyHandler(w, req)
	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", w.Result().StatusCode)
	}
}

func TestApplyHandler_Error(t *testing.T) {
	s := &Server{store: &MockStore{ApplyErr: errors.New("fail")}}
	req := httptest.NewRequest(http.MethodPost, "/apply", strings.NewReader("test"))
	w := httptest.NewRecorder()
	s.applyHandler(w, req)
	if w.Result().StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Result().StatusCode)
	}
}

func TestGetHandler_OnlyGet(t *testing.T) {
	s := &Server{store: &MockStore{GetValueExists: true, GetValue: "bar"}}

	req := httptest.NewRequest(http.MethodPost, "/get", nil)
	w := httptest.NewRecorder()
	s.getHandler(w, req)
	if w.Result().StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 Method Not Allowed, got %d", w.Result().StatusCode)
	}

	req = httptest.NewRequest(http.MethodGet, "/get?key=foo", nil)
	w = httptest.NewRecorder()
	s.getHandler(w, req)
	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", w.Result().StatusCode)
	}
}

func TestGetHandler_KeyQueryParamMustNotEmpty(t *testing.T) {
	s := &Server{store: &MockStore{}}

	req := httptest.NewRequest(http.MethodGet, "/get", nil)
	w := httptest.NewRecorder()
	s.getHandler(w, req)
	if w.Result().StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request, got %d", w.Result().StatusCode)
	}

	req = httptest.NewRequest(http.MethodGet, "/get?key=", nil)
	w = httptest.NewRecorder()
	s.getHandler(w, req)
	if w.Result().StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request, got %d", w.Result().StatusCode)
	}
}

func TestGetHandler_Success(t *testing.T) {
	s := &Server{store: &MockStore{GetValue: "bar", GetValueExists: true}}
	req := httptest.NewRequest(http.MethodGet, "/get?key=foo", nil)
	w := httptest.NewRecorder()
	s.getHandler(w, req)
	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", w.Result().StatusCode)
	}
	if strings.Trim(w.Body.String(), " \n") != `{"data":"bar"}` {
		t.Errorf("expected response body to be `{\"data\":\"bar\"}`, got `%s`", w.Body.String())
	}
}

func TestGetHandler_ErrorIfKeyNotExist(t *testing.T) {
	s := &Server{store: &MockStore{GetValue: nil, GetValueExists: false}}
	req := httptest.NewRequest(http.MethodGet, "/get?key=nonexistent", nil)
	w := httptest.NewRecorder()
	s.getHandler(w, req)
	if w.Result().StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request, got %d", w.Result().StatusCode)
	}
}
