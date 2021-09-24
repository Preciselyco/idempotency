package idempotency

import (
	"context"
	"testing"
)

func TestContext(t *testing.T) {
	have := "b2ab44c6-ed51-4453-ab00-90779453f2b3"
	ctx := context.Background()

	withKey := NewContext(ctx, have)

	got, ok := FromContext(withKey)
	if !ok {
		t.Errorf("want ok = true, got false")
	}

	if got != have {
		t.Errorf("want idempotency key = %v, got %v", have, got)
	}
}
