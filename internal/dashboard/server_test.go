package dashboard

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/iheanyi/grove/internal/registry"
)

func TestBackgroundUpdatesStops(t *testing.T) {
	s := &Server{
		wsHub:    NewHub(),
		registry: registry.New(),
		stopCh:   make(chan struct{}),
		bgDone:   make(chan struct{}),
	}

	go s.backgroundUpdates(time.Millisecond)

	if err := s.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	select {
	case <-s.bgDone:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("backgroundUpdates did not stop")
	}
}

func TestHandleStaticServesIndexAndSPAFallback(t *testing.T) {
	s := &Server{}

	for _, path := range []string{"/", "/workspace/missing"} {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			rec := httptest.NewRecorder()

			s.handleStatic(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
			}
			if body := rec.Body.String(); body == "" {
				t.Fatal("expected embedded index response body")
			}
		})
	}
}
