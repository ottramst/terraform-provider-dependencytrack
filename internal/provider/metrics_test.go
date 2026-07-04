package provider

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"
)

// fastMetricsPolling shrinks the refresh polling knobs for the duration of a
// test so the io.EOF paths don't slow the unit test suite down.
func fastMetricsPolling(t *testing.T) {
	t.Helper()

	origInterval, origTimeout := metricsRefreshPollInterval, metricsRefreshPollTimeout
	metricsRefreshPollInterval = time.Millisecond
	metricsRefreshPollTimeout = 20 * time.Millisecond
	t.Cleanup(func() {
		metricsRefreshPollInterval, metricsRefreshPollTimeout = origInterval, origTimeout
	})
}

func TestCurrentMetricsWithRefreshImmediate(t *testing.T) {
	refreshed := false

	got, found, err := currentMetricsWithRefresh(context.Background(),
		func(context.Context) (int, error) { return 42, nil },
		func(context.Context) error { refreshed = true; return nil },
	)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if !found || got != 42 {
		t.Fatalf("got (%v, %v), want (42, true)", got, found)
	}
	if refreshed {
		t.Fatal("refresh must not be triggered when metrics already exist")
	}
}

func TestCurrentMetricsWithRefreshAfterRefresh(t *testing.T) {
	fastMetricsPolling(t)

	calls := 0
	refreshed := false

	got, found, err := currentMetricsWithRefresh(context.Background(),
		func(context.Context) (int, error) {
			calls++
			if calls < 3 {
				return 0, io.EOF
			}
			return 7, nil
		},
		func(context.Context) error { refreshed = true; return nil },
	)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if !found || got != 7 {
		t.Fatalf("got (%v, %v), want (7, true)", got, found)
	}
	if !refreshed {
		t.Fatal("refresh should have been triggered")
	}
}

func TestCurrentMetricsWithRefreshNeverAppears(t *testing.T) {
	fastMetricsPolling(t)

	_, found, err := currentMetricsWithRefresh(context.Background(),
		func(context.Context) (int, error) { return 0, io.EOF },
		func(context.Context) error { return nil },
	)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if found {
		t.Fatal("found should be false when metrics never appear")
	}
}

func TestCurrentMetricsWithRefreshErrorPassthrough(t *testing.T) {
	wantErr := errors.New("boom")

	_, _, err := currentMetricsWithRefresh(context.Background(),
		func(context.Context) (int, error) { return 0, wantErr },
		func(context.Context) error { return nil },
	)
	if !errors.Is(err, wantErr) {
		t.Fatalf("got error %v, want %v", err, wantErr)
	}
}

func TestCurrentMetricsWithRefreshRefreshError(t *testing.T) {
	wantErr := errors.New("refresh failed")

	_, _, err := currentMetricsWithRefresh(context.Background(),
		func(context.Context) (int, error) { return 0, io.EOF },
		func(context.Context) error { return wantErr },
	)
	if !errors.Is(err, wantErr) {
		t.Fatalf("got error %v, want %v", err, wantErr)
	}
}

func TestCurrentMetricsWithRefreshContextCanceled(t *testing.T) {
	fastMetricsPolling(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, _, err := currentMetricsWithRefresh(ctx,
		func(context.Context) (int, error) { return 0, io.EOF },
		func(context.Context) error { return nil },
	)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got error %v, want context.Canceled", err)
	}
}
