package localtime

import (
	"sync"
	"time"

	"github.com/beevik/ntp"

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
	*util.FunctionDaemon
	server   string
	offset   time.Duration
	interval time.Duration
}

// NewTimeSyncer creates new TimeSyncer
func NewTimeSyncer(server string, checkInterval time.Duration) (*TimeSyncer, error) {
	_, err := ntp.Query(server)
	if err != nil {
		return nil, err
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

	if checkInterval < minTimeSyncCheckInterval {
		ts.Log().Warn().
			Dur("checkInterval", checkInterval).
			Dur("minCeckInterval", minTimeSyncCheckInterval).
			Msg("checkInterval too short")
	}

	ts.FunctionDaemon = util.NewFunctionDaemon(ts.schedule, true)

	return ts, nil
}

// Start starts TimeSyncer
func (ts *TimeSyncer) Start() error {
	ts.Log().Debug().Msg("started")

	return ts.FunctionDaemon.Start()
}

func (ts *TimeSyncer) schedule(stopChan chan struct{}) error {
	ticker := time.NewTicker(ts.interval)
	defer ticker.Stop()

end:
	for {
		select {
		case <-stopChan:
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
