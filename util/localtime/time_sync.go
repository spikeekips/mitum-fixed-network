package localtime

import (
	"context"
	"sync"
	"time"

	"github.com/beevik/ntp"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/logging"
)

var (
	allowedTimeSyncOffset    = time.Millisecond * 500
	minTimeSyncCheckInterval = time.Second * 5
	timeSyncer               *TimeSyncer
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
	if err := util.Retry(3, time.Second*2, func(int) error {
		if _, err := ntp.Query(server); err != nil {
			return xerrors.Errorf("failed to query ntp server, %q: %w", server, err)
		}

		return nil
	}); err != nil {
		return nil, xerrors.Errorf("failed to query ntp server, %q: %w", server, err)
	}

	ts := &TimeSyncer{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
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
			ts.check()
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

func (ts *TimeSyncer) check() {
	ts.Lock()
	defer ts.Unlock()

	response, err := ntp.Query(ts.server)
	if err != nil {
		ts.Log().Error().Err(err).Msg("failed to query")

		return
	}

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
			Dur("offset", ts.offset).
			Msg("time checked")
	}()

	if ts.offset < 1 {
		ts.offset = response.ClockOffset

		return
	}

	switch diff := ts.offset - response.ClockOffset; {
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

	ts.offset = response.ClockOffset
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
