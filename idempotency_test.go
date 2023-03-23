package idempotency

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

type incompleteStorage struct {
	*memoryStorage
}

func newIncompleteStorage() Storage {
	return &incompleteStorage{
		NewMemoryStorage(),
	}
}

// Complete is not set so that all requests are InProgress.
func (f *incompleteStorage) Complete(ctx context.Context, key string) error {
	return nil
}

func TestVerify(t *testing.T) {
	testRestorer := WithRestorer(func(idempotencyKey string, w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	tests := []struct {
		name           string
		have           *state
		wantHTTPStatus int
		unsetHeader    bool
		repeated       int
	}{
		{
			// If the key has not been seen before, perform the request.
			name:           "First request, pass through to handler",
			have:           New(NewMemoryStorage(), testRestorer),
			wantHTTPStatus: http.StatusOK,
			repeated:       1,
		},
		{
			// If the "Idempotency-Key" request header is missing
			// for a documented idempotent operation requiring it,
			// it should return a 400 Bad Request.
			name:           "No Idempotency-Key header, bad request error",
			have:           New(NewMemoryStorage(), testRestorer),
			wantHTTPStatus: http.StatusBadRequest,
			unsetHeader:    true,
			repeated:       1,
		},
		{
			// If a request with the key is completed, then return the prior result.
			name:           "Repeated requests with same Idempotency-Key ends up in restorer",
			have:           New(NewMemoryStorage(), testRestorer),
			wantHTTPStatus: http.StatusNoContent,
			repeated:       2,
		},
		{
			// If a request with the key is in process, then return a 409 Conflict.
			name:           "Repeated requests on incomplete requests renders a conflict",
			have:           New(newIncompleteStorage(), testRestorer),
			wantHTTPStatus: http.StatusConflict,
			repeated:       2,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler := func() http.Handler {
				fn := func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				}
				return http.HandlerFunc(fn)
			}

			var resp *http.Response
			for i := 0; i < test.repeated; i++ {
				req := httptest.NewRequest("GET", "http://example.com/foo", nil)
				if !test.unsetHeader {
					req.Header.Set("Idempotency-Key", "deadbeef")
				}

				w := httptest.NewRecorder()
				test.have.Verify(handler()).ServeHTTP(w, req)
				resp = w.Result()
			}

			if resp.StatusCode != test.wantHTTPStatus {
				t.Errorf("want status code %v, got %v", test.wantHTTPStatus, resp.StatusCode)
			}
		})
	}
}
