package idempotency

import (
	"fmt"
	"net/http"
)

// RequestStatus keeps track of requests that are in process and what body sum
// they have, this is to check wether to return a Conflict or a Unprocessable
// Entity.
type RequestStatus struct {
	InProcess bool
}

// Option is the functional option signature for configuring idempotency.
type Option func(*state)

type state struct {
	storage      Storage
	restorer     func(idempotencyKey string, w http.ResponseWriter, r *http.Request)
	errResponder func(err error, status int, w http.ResponseWriter, r *http.Request)
}

// WithRestorer configures the function that restores a previous payload from
// storage.
func WithRestorer(f func(idempotencyKey string, w http.ResponseWriter, r *http.Request)) Option {
	return func(s *state) {
		s.restorer = f
	}
}

// WithErrorResponder configures a function that responds to the client
// whenever an error occurs.
func WithErrorResponder(f func(err error, status int, w http.ResponseWriter, r *http.Request)) Option {
	return func(s *state) {
		s.errResponder = f
	}
}

// New creates a new idempotency state.
func New(storage Storage, opts ...Option) *state {
	s := &state{
		storage: storage,
		restorer: func(idempotencyKey string, w http.ResponseWriter, r *http.Request) {
		},
		errResponder: func(err error, status int, w http.ResponseWriter, r *http.Request) {
			http.Error(w, err.Error(), status)
		},
	}

	for _, opt := range opts {
		if opt != nil {
			opt(s)
		}
	}

	return s
}

// Verify verifies the contents of the Idempotency-Key to make sure the
// request has not been seen before. The RFC defines the following
// functionality:
// * If the key has not been seen before, perform the request.
// * If a request with the key is in process, then return a 409 Conflict.
// * If a request with the key is completed, then return the prior result.
// * TODO: If a request has a different request payload, it should return a
// 422 Unprocessable Entity.
// * TODO: Implement Link: <https://developer.example.com/idempotency>; rel="describedby"; type="text/html"
func (s *state) Verify(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		idempotencyKey := r.Header.Get("Idempotency-Key")

		if idempotencyKey == "" {
			s.errResponder(fmt.Errorf("no Idempotency-Key set"), http.StatusBadRequest, w, r)
			return
		}

		status, err := s.storage.Get(ctx, idempotencyKey)
		if err != nil {
			s.errResponder(fmt.Errorf("could not process request to get Idempotency-Key: %w", err), http.StatusInternalServerError, w, r)
			return
		}

		// If the idempotency key does not exist, we do not need to
		// process further.
		if status == nil {
			// Add the key right away.
			err = s.storage.Add(ctx, idempotencyKey)
			if err != nil {
				s.errResponder(fmt.Errorf("could not process request to save Idempotency-Key: %w", err), http.StatusInternalServerError, w, r)
				return
			}

			// Run the handlers that has the actual functionality.
			next.ServeHTTP(w, r)

			// Complete the request.
			err = s.storage.Complete(ctx, idempotencyKey)
			if err != nil {
				s.errResponder(fmt.Errorf("could not complete request: %w", err), http.StatusInternalServerError, w, r)
			}
			return
		}

		// Conflict if it is in process.
		if status.InProcess {
			s.errResponder(fmt.Errorf("request already in progress"), http.StatusConflict, w, r)
			return
		}

		// Return the previous data if the request has been completed
		// previously.
		s.restorer(idempotencyKey, w, r)
	}

	return http.HandlerFunc(fn)
}
