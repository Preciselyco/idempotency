# Idempotency - A Go middleware for idempotency in HTTP

This is a middleware that implements an RFC draft for
[idempotency-key-header](https://datatracker.ietf.org/doc/html/draft-ietf-httpapi-idempotency-key-header).


### Example code:

	idempotencyMiddleware := idempotency.New(idempotency.NewMemoryStorage(),
		func(idempotencyKey string, w http.ResponseWriter, r *http.Request) {
			req, err := _state.FindByKey(idempotencyKey)
			if err != nil {
				http.Error(w, "No such key", http.StatusBadRequest)
				return
			}

			// Return content previously returned by myHandler.
		})

	http.Handle("/", idempotencyMiddleware.Verify(myHandler))
