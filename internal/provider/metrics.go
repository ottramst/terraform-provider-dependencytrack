package provider

import (
	"context"
	"errors"
	"io"
	"time"
)

// Timing knobs for currentMetricsWithRefresh, overridable in unit tests.
var (
	metricsRefreshPollInterval = 2 * time.Second
	metricsRefreshPollTimeout  = 60 * time.Second
)

// currentMetricsWithRefresh reads the latest metrics via latest, working
// around Dependency-Track v4's behavior of answering HTTP 200 with an empty
// body when metrics have never been computed (client-go surfaces that as
// io.EOF). In that case it triggers refresh once (an asynchronous
// recalculation task) and polls latest until metrics appear or the poll
// timeout elapses; if they never appear it reports found=false with a nil
// error so the caller can fall back to zero values. Dependency-Track v5
// synthesizes a zeroed metrics object instead of an empty body, so the
// refresh path is never taken there.
func currentMetricsWithRefresh[T any](
	ctx context.Context,
	latest func(context.Context) (T, error),
	refresh func(context.Context) error,
) (m T, found bool, err error) {
	m, err = latest(ctx)
	if err == nil {
		return m, true, nil
	}
	if !errors.Is(err, io.EOF) {
		return m, false, err
	}

	// No metrics computed yet: kick off a refresh and poll briefly.
	if err := refresh(ctx); err != nil {
		return m, false, err
	}

	deadline := time.Now().Add(metricsRefreshPollTimeout)
	for {
		select {
		case <-ctx.Done():
			return m, false, ctx.Err()
		case <-time.After(metricsRefreshPollInterval):
		}

		m, err = latest(ctx)
		if err == nil {
			return m, true, nil
		}
		if !errors.Is(err, io.EOF) {
			return m, false, err
		}

		if time.Now().After(deadline) {
			return m, false, nil
		}
	}
}
