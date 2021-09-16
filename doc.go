/*
Package idempotency implements the Idempotency-Key HTTP Header described in
the draft-ietf-httpapi-idempotency-key-header-00 RFC.

See: https://datatracker.ietf.org/doc/html/draft-ietf-httpapi-idempotency-key-header

Note that the RFC is a draft and some assumptions will be made a long the way.

The package aims to implement a way to use a net/http handler that will check
the Idempotency-Key header and determine what action to do. The client is
responsible of sending a unique value of the Idempotency-Key header,
recommended values are UUIDs.
*/
package idempotency
