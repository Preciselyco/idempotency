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

type state struct {
	storage  Storage
	restorer func(idempotencyKey string, w http.ResponseWriter, r *http.Request)
}

// New creates a new idempotency state with a storage and a restorer to be
// able to look up the Idempotency-Keys that has been seen and the restorer to
// get back the previous status and response when a request has been
// completed.
func New(storage Storage, restorer func(idempotencyKey string, w http.ResponseWriter, r *http.Request)) *state {
	return &state{
		storage:  storage,
		restorer: restorer,
	}
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
		idempotencyKey := r.Header.Get("Idempotency-Key")

		if idempotencyKey == "" {
			http.Error(w, fmt.Errorf("no Idempotency-Key set").Error(), http.StatusBadRequest)
			return
		}

		status, err := s.storage.Get(idempotencyKey)
		if err != nil {
			http.Error(w, fmt.Errorf("could not process request to get Idempotency-Key: %w", err).Error(), http.StatusInternalServerError)
			return
		}

		// If the idempotency key does not exist, we do not need to
		// process further.
		if status == nil {
			// Add the key right away.
			err = s.storage.Add(idempotencyKey)
			if err != nil {
				http.Error(w, fmt.Errorf("could not process request to save Idempotency-Key: %w", err).Error(), http.StatusInternalServerError)
				return
			}

			// Run the handlers that has the actual functionality.
			next.ServeHTTP(w, r)

			// Complete the request.
			err = s.storage.Complete(idempotencyKey)
			if err != nil {
				http.Error(w, fmt.Errorf("could not complete request: %w", err).Error(), http.StatusInternalServerError)
			}
			return
		}

		// Conflict if it is in process.
		if status.InProcess {
			http.Error(w, "request already in progress", http.StatusConflict)
			return
		}

		// Return the previous data if the request has been completed
		// previously.
		s.restorer(idempotencyKey, w, r)
	}

	return http.HandlerFunc(fn)
}
