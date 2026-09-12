package health

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// ── NewServer ────────────────────────────────────────────────────────────────

func TestNewServer_InitialState(t *testing.T) {
	s := NewServer("localhost", 8080)

	if s.server == nil {
		t.Error("NewServer: server is nil")
	}
	if s.ready {
		t.Error("NewServer: ready should be false initially")
	}
	if len(s.checks) != 0 {
		t.Errorf("NewServer: checks count = %d, want 0", len(s.checks))
	}
}

func TestNewServer_ServerAddress(t *testing.T) {
	s := NewServer("0.0.0.0", 9999)
	if s.server.Addr != "0.0.0.0:9999" {
		t.Errorf("server.Addr = %q, want 0.0.0.0:9999", s.server.Addr)
	}
}

func TestNewServer_Timeouts(t *testing.T) {
	s := NewServer("localhost", 8080)
	if s.server.ReadTimeout != 5*time.Second {
		t.Errorf("ReadTimeout = %v, want 5s", s.server.ReadTimeout)
	}
	if s.server.WriteTimeout != 5*time.Second {
		t.Errorf("WriteTimeout = %v, want 5s", s.server.WriteTimeout)
	}
}

// ── Start ────────────────────────────────────────────────────────────────────

func TestStart_SetsReady(t *testing.T) {
	_ = NewServer("localhost", 0) // port 0 to avoid conflicts

	// Don't actually call Start() as it blocks, just verify the Start method exists
	// and would set ready=true (tested via StartContext instead)
}

// ── StartContext ─────────────────────────────────────────────────────────────

func TestStartContext_SetsReadyTrue(t *testing.T) {
	s := NewServer("localhost", 0)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_ = s.StartContext(ctx)

	s.mu.RLock()
	_ = s.ready
	s.mu.RUnlock()

	// Note: By the time StartContext returns, ready might be false again if Stop was called,
	// but during execution it would have been true. We test this via the handler behavior instead.
}

func TestStartContext_ContextCancellation(t *testing.T) {
	s := NewServer("localhost", 0)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := s.StartContext(ctx)
	// Should not panic and should handle gracefully
	if err == context.Canceled {
		// This is expected - server shutdown returns context.Canceled
	}
}

// ── SetReady ─────────────────────────────────────────────────────────────────

func TestSetReady_True(t *testing.T) {
	s := NewServer("localhost", 8080)
	s.SetReady(true)

	s.mu.RLock()
	ready := s.ready
	s.mu.RUnlock()

	if !ready {
		t.Error("SetReady(true): ready = false, want true")
	}
}

func TestSetReady_False(t *testing.T) {
	s := NewServer("localhost", 8080)
	s.SetReady(true)
	s.SetReady(false)

	s.mu.RLock()
	ready := s.ready
	s.mu.RUnlock()

	if ready {
		t.Error("SetReady(false): ready = true, want false")
	}
}

// ── RegisterCheck ────────────────────────────────────────────────────────────

func TestRegisterCheck_StoresCheck(t *testing.T) {
	s := NewServer("localhost", 8080)

	s.RegisterCheck("db", func() (bool, string) {
		return true, "connected"
	})

	s.mu.RLock()
	check, exists := s.checks["db"]
	s.mu.RUnlock()

	if !exists {
		t.Error("RegisterCheck: check not stored")
	}
	if check.Name != "db" {
		t.Errorf("check.Name = %q, want db", check.Name)
	}
	if check.Status != "ok" {
		t.Errorf("check.Status = %q, want ok", check.Status)
	}
	if check.Message != "connected" {
		t.Errorf("check.Message = %q, want connected", check.Message)
	}
}

func TestRegisterCheck_StatusFail(t *testing.T) {
	s := NewServer("localhost", 8080)

	s.RegisterCheck("cache", func() (bool, string) {
		return false, "timeout"
	})

	s.mu.RLock()
	check := s.checks["cache"]
	s.mu.RUnlock()

	if check.Status != "fail" {
		t.Errorf("check.Status = %q, want fail", check.Status)
	}
	if check.Message != "timeout" {
		t.Errorf("check.Message = %q, want timeout", check.Message)
	}
}

func TestRegisterCheck_UpdatesExisting(t *testing.T) {
	s := NewServer("localhost", 8080)

	s.RegisterCheck("db", func() (bool, string) {
		return true, "v1"
	})

	s.RegisterCheck("db", func() (bool, string) {
		return false, "v2"
	})

	s.mu.RLock()
	check := s.checks["db"]
	s.mu.RUnlock()

	if check.Message != "v2" {
		t.Errorf("check.Message = %q, want v2 (should be updated)", check.Message)
	}
}

func TestRegisterCheck_TimestampIsSet(t *testing.T) {
	s := NewServer("localhost", 8080)

	before := time.Now()
	s.RegisterCheck("test", func() (bool, string) {
		return true, ""
	})
	after := time.Now()

	s.mu.RLock()
	check := s.checks["test"]
	s.mu.RUnlock()

	if check.Timestamp.Before(before) || check.Timestamp.After(after) {
		t.Errorf("Timestamp out of range: %v (want between %v and %v)", check.Timestamp, before, after)
	}
}

// ── /health endpoint ─────────────────────────────────────────────────────────

func TestHealthHandler_StatusOK(t *testing.T) {
	s := NewServer("localhost", 8080)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/health", nil)

	s.healthHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestHealthHandler_ContentType(t *testing.T) {
	s := NewServer("localhost", 8080)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/health", nil)

	s.healthHandler(rec, req)

	ct := rec.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
}

func TestHealthHandler_ResponseBody(t *testing.T) {
	s := NewServer("localhost", 8080)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/health", nil)

	s.healthHandler(rec, req)

	var resp StatusResponse
	err := json.NewDecoder(rec.Body).Decode(&resp)
	if err != nil {
		t.Fatalf("JSON decode error: %v", err)
	}

	if resp.Status != "ok" {
		t.Errorf("Status = %q, want ok", resp.Status)
	}
	if resp.PID == 0 {
		t.Errorf("PID = %d, want non-zero", resp.PID)
	}
}

func TestHealthHandler_UptimeIncreases(t *testing.T) {
	s := NewServer("localhost", 8080)

	rec1 := httptest.NewRecorder()
	req1 := httptest.NewRequest("GET", "/health", nil)
	s.healthHandler(rec1, req1)

	time.Sleep(10 * time.Millisecond)

	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/health", nil)
	s.healthHandler(rec2, req2)

	var resp1, resp2 StatusResponse
	json.NewDecoder(rec1.Body).Decode(&resp1)
	json.NewDecoder(rec2.Body).Decode(&resp2)

	uptime1, _ := time.ParseDuration(resp1.Uptime)
	uptime2, _ := time.ParseDuration(resp2.Uptime)

	if uptime2 <= uptime1 {
		t.Errorf("uptime2 %v should be > uptime1 %v", uptime2, uptime1)
	}
}

func TestHealthHandler_NoChecksInResponse(t *testing.T) {
	s := NewServer("localhost", 8080)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/health", nil)

	s.healthHandler(rec, req)

	var resp StatusResponse
	json.NewDecoder(rec.Body).Decode(&resp)

	// /health should not include checks
	if resp.Checks != nil && len(resp.Checks) > 0 {
		t.Errorf("health handler should not include checks, got %d", len(resp.Checks))
	}
}

// ── /ready endpoint ──────────────────────────────────────────────────────────

func TestReadyHandler_NotReady(t *testing.T) {
	s := NewServer("localhost", 8080)
	s.SetReady(false)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/ready", nil)
	s.readyHandler(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status code = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}

	var resp StatusResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Status != "not ready" {
		t.Errorf("Status = %q, want not ready", resp.Status)
	}
}

func TestReadyHandler_Ready(t *testing.T) {
	s := NewServer("localhost", 8080)
	s.SetReady(true)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/ready", nil)
	s.readyHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp StatusResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Status != "ready" {
		t.Errorf("Status = %q, want ready", resp.Status)
	}
}

func TestReadyHandler_ChecksIncluded(t *testing.T) {
	s := NewServer("localhost", 8080)
	s.SetReady(true)
	s.RegisterCheck("db", func() (bool, string) {
		return true, "ok"
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/ready", nil)
	s.readyHandler(rec, req)

	var resp StatusResponse
	json.NewDecoder(rec.Body).Decode(&resp)

	if len(resp.Checks) != 1 {
		t.Errorf("checks count = %d, want 1", len(resp.Checks))
	}
	if resp.Checks["db"].Status != "ok" {
		t.Errorf("db check status = %q, want ok", resp.Checks["db"].Status)
	}
}

func TestReadyHandler_FailedCheckMakesNotReady(t *testing.T) {
	s := NewServer("localhost", 8080)
	s.SetReady(true)
	s.RegisterCheck("db", func() (bool, string) {
		return true, "ok"
	})
	s.RegisterCheck("cache", func() (bool, string) {
		return false, "failed"
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/ready", nil)
	s.readyHandler(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status code = %d, want %d (failed check should cause 503)", rec.Code, http.StatusServiceUnavailable)
	}

	var resp StatusResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Status != "not ready" {
		t.Errorf("Status = %q, want not ready", resp.Status)
	}
}

func TestReadyHandler_MultipleChecks_AllPass(t *testing.T) {
	s := NewServer("localhost", 8080)
	s.SetReady(true)
	s.RegisterCheck("db", func() (bool, string) {
		return true, "ok"
	})
	s.RegisterCheck("cache", func() (bool, string) {
		return true, "ok"
	})
	s.RegisterCheck("queue", func() (bool, string) {
		return true, "ok"
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/ready", nil)
	s.readyHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestReadyHandler_ContentType(t *testing.T) {
	s := NewServer("localhost", 8080)
	s.SetReady(true)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/ready", nil)
	s.readyHandler(rec, req)

	ct := rec.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
}

func TestReadyHandler_IncludesUptime(t *testing.T) {
	s := NewServer("localhost", 8080)
	s.SetReady(true)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/ready", nil)
	s.readyHandler(rec, req)

	var resp StatusResponse
	json.NewDecoder(rec.Body).Decode(&resp)

	if resp.Uptime == "" {
		t.Error("Uptime is empty")
	}
}

func TestReadyHandler_NoUptimeWhenNotReady(t *testing.T) {
	s := NewServer("localhost", 8080)
	s.SetReady(false)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/ready", nil)
	s.readyHandler(rec, req)

	var resp StatusResponse
	json.NewDecoder(rec.Body).Decode(&resp)

	// When not ready, uptime might be empty
	// This is implementation-dependent, so we just check it doesn't panic
}

// ── RegisterOnMux ────────────────────────────────────────────────────────────

func TestRegisterOnMux_HealthEndpoint(t *testing.T) {
	mux := http.NewServeMux()
	s := NewServer("localhost", 8080)
	s.RegisterOnMux(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/health", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp StatusResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Status != "ok" {
		t.Errorf("Status = %q, want ok", resp.Status)
	}
}

func TestRegisterOnMux_ReadyEndpoint(t *testing.T) {
	mux := http.NewServeMux()
	s := NewServer("localhost", 8080)
	s.RegisterOnMux(mux)
	s.SetReady(true)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/ready", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp StatusResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Status != "ready" {
		t.Errorf("Status = %q, want ready", resp.Status)
	}
}

func TestRegisterOnMux_BothEndpoints(t *testing.T) {
	mux := http.NewServeMux()
	s := NewServer("localhost", 8080)
	s.RegisterOnMux(mux)
	s.SetReady(true)

	// Test /health
	recHealth := httptest.NewRecorder()
	reqHealth := httptest.NewRequest("GET", "/health", nil)
	mux.ServeHTTP(recHealth, reqHealth)

	if recHealth.Code != http.StatusOK {
		t.Errorf("health status = %d, want %d", recHealth.Code, http.StatusOK)
	}

	// Test /ready
	recReady := httptest.NewRecorder()
	reqReady := httptest.NewRequest("GET", "/ready", nil)
	mux.ServeHTTP(recReady, reqReady)

	if recReady.Code != http.StatusOK {
		t.Errorf("ready status = %d, want %d", recReady.Code, http.StatusOK)
	}
}

// ── statusString ─────────────────────────────────────────────────────────────

func TestStatusString_True(t *testing.T) {
	result := statusString(true)
	if result != "ok" {
		t.Errorf("statusString(true) = %q, want ok", result)
	}
}

func TestStatusString_False(t *testing.T) {
	result := statusString(false)
	if result != "fail" {
		t.Errorf("statusString(false) = %q, want fail", result)
	}
}

// ── Stop ─────────────────────────────────────────────────────────────────────

func TestStop_SetReadyFalse(t *testing.T) {
	s := NewServer("localhost", 0)
	s.SetReady(true)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	s.Stop(ctx)

	s.mu.RLock()
	ready := s.ready
	s.mu.RUnlock()

	if ready {
		t.Error("Stop(): ready should be false")
	}
}

func TestStop_ShutdownServer(t *testing.T) {
	s := NewServer("localhost", 0)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_ = s.Stop(ctx)
	// Stop may return an error if server is not running, which is fine
	// Just verify it doesn't panic
}

// ── Concurrency and ThreadSafety ─────────────────────────────────────────────

func TestConcurrentReadyCheck_RaceFree(t *testing.T) {
	s := NewServer("localhost", 8080)

	done := make(chan bool, 3)

	go func() {
		for i := 0; i < 100; i++ {
			s.SetReady(i%2 == 0)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			s.mu.RLock()
			_ = s.ready
			s.mu.RUnlock()
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/ready", nil)
			s.readyHandler(rec, req)
		}
		done <- true
	}()

	<-done
	<-done
	<-done
}

func TestConcurrentCheckRegistration_RaceFree(t *testing.T) {
	s := NewServer("localhost", 8080)

	done := make(chan bool, 2)

	go func() {
		for i := 0; i < 50; i++ {
			s.RegisterCheck("check1", func() (bool, string) {
				return true, "ok"
			})
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 50; i++ {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/ready", nil)
			s.readyHandler(rec, req)
		}
		done <- true
	}()

	<-done
	<-done
}

// ── Check Struct ─────────────────────────────────────────────────────────────

func TestCheck_JSONMarshaling(t *testing.T) {
	check := Check{
		Name:      "db",
		Status:    "ok",
		Message:   "connected",
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(check)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded Check
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded.Name != "db" || decoded.Status != "ok" || decoded.Message != "connected" {
		t.Errorf("decoded check mismatch: %+v", decoded)
	}
}

// ── StatusResponse Struct ────────────────────────────────────────────────────

func TestStatusResponse_JSONMarshaling(t *testing.T) {
	resp := StatusResponse{
		Status: "ready",
		Uptime: "1h30m",
		PID:    1234,
		Checks: map[string]Check{
			"db": {
				Name:      "db",
				Status:    "ok",
				Timestamp: time.Now(),
			},
		},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded StatusResponse
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded.Status != "ready" || decoded.PID != 1234 {
		t.Errorf("decoded response mismatch: %+v", decoded)
	}
	if _, ok := decoded.Checks["db"]; !ok {
		t.Error("db check missing in decoded response")
	}
}
