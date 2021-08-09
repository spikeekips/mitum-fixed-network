package localtime

import (
	"context"
	"sync"
	"time"

	"github.com/beevik/ntp"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/logging"
)

var (
	allowedTimeSyncOffset     = time.Millisecond * 500
	minTimeSyncCheckInterval  = time.Minute
	timeServerQueryingTimeout = time.Second * 5
	timeSyncer                *TimeSyncer
)

// TimeSyncer tries to sync time to time server.
type TimeSyncer struct {
	sync.RWMutex
	*logging.Logging
	*util.ContextDaemon
	server   string
	offset   time.Duration
	interval time.Duration
}

// NewTimeSyncer creates new TimeSyncer
func NewTimeSyncer(server string, checkInterval time.Duration) (*TimeSyncer, error) {
	if checkInterval < timeServerQueryingTimeout {
		return nil, errors.Errorf("too narrow checking interval; should be over %v", timeServerQueryingTimeout)
	}

	if err := util.Retry(3, time.Second*2, func(int) error {
		if _, err := ntp.Query(server); err != nil {
			return errors.Wrapf(err, "failed to query ntp server, %q", server)
		}

		return nil
	}); err != nil {
		return nil, err
	}

	ts := &TimeSyncer{
		Logging: logging.NewLogging(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "time-syncer").
				Str("server", server).
				Dur("interval", checkInterval)
		}),
		server:   server,
		interval: checkInterval,
	}

	ts.ContextDaemon = util.NewContextDaemon("time-syncer", ts.schedule)

	ts.check()

	return ts, nil
}

// Start starts TimeSyncer
func (ts *TimeSyncer) Start() error {
	ts.Log().Debug().Msg("started")

	if ts.interval < minTimeSyncCheckInterval {
		ts.Log().Warn().
			Dur("check_interval", ts.interval).
			Dur("min_ceck_interval", minTimeSyncCheckInterval).
			Msg("interval too short")
	}

	return ts.ContextDaemon.Start()
}

func (ts *TimeSyncer) schedule(ctx context.Context) error {
	ticker := time.NewTicker(ts.interval)
	defer ticker.Stop()

end:
	for {
		select {
		case <-ctx.Done():
			ts.Log().Debug().Msg("stopped")

			break end
		case <-ticker.C:
			started := time.Now()
			ts.check()
			ts.Log().Debug().Dur("elapsed", time.Since(started)).Msg("time queried")
		}
	}

	return nil
}

// Offset returns the latest time offset.
func (ts *TimeSyncer) Offset() time.Duration {
	ts.RLock()
	defer ts.RUnlock()

	return ts.offset
}

func (ts *TimeSyncer) setOffset(d time.Duration) {
	ts.Lock()
	defer ts.Unlock()

	ts.offset = d
}

func (ts *TimeSyncer) check() {
	response, err := ntp.QueryWithOptions(ts.server, ntp.QueryOptions{Timeout: timeServerQueryingTimeout})
	if err != nil {
		ts.Log().Error().Err(err).Msg("failed to query")

		return
	}

	offset := ts.Offset()

	if err := response.Validate(); err != nil {
		ts.Log().Error().
			Err(err).
			Interface("response", response).
			Msg("invalid response")

		return
	}

	defer func() {
		ts.Log().Debug().
			Interface("response", response).
			Dur("offset", offset).
			Msg("time checked")
	}()

	if offset < 1 {
		ts.setOffset(response.ClockOffset)

		return
	}

	switch diff := offset - response.ClockOffset; {
	case diff == 0:
		return
	case diff > 0:
		if diff < allowedTimeSyncOffset {
			return
		}
	case diff < 0:
		if diff > allowedTimeSyncOffset*-1 {
			return
		}
	}

	ts.setOffset(response.ClockOffset)
}

// SetTimeSyncer sets the global TimeSyncer.
func SetTimeSyncer(syncer *TimeSyncer) {
	timeSyncer = syncer
}

// Now returns the tuned Time with TimeSyncer.Offset().
func Now() time.Time {
	if timeSyncer == nil {
		return time.Now()
	}

	return time.Now().Add(timeSyncer.Offset())
}

func UTCNow() time.Time {
	return Now().UTC()
}
