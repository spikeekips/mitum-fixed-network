package localtime

import (
	"sync"
	"time"

	"github.com/beevik/ntp"
	"github.com/rs/zerolog"

	"github.com/spikeekips/mitum/logging"
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
	server   string
	offset   time.Duration
	stopChan chan bool
	interval time.Duration
}

// NewTimeSyncer creates new TimeSyncer
func NewTimeSyncer(server string, checkInterval time.Duration) (*TimeSyncer, error) {
	_, err := ntp.Query(server)
	if err != nil {
		return nil, err
	}

	ts := &TimeSyncer{
		Logging: logging.NewLogging(func(c zerolog.Context) zerolog.Context {
			return c.
				Str("module", "time-syncer").
				Str("server", server).
				Dur("interval", checkInterval)
		}),
		server:   server,
		interval: checkInterval,
		stopChan: make(chan bool),
	}

	if checkInterval < minTimeSyncCheckInterval {
		ts.Log().Warn().
			Dur("checkInterval", checkInterval).
			Dur("minCeckInterval", minTimeSyncCheckInterval).
			Msg("checkInterval too short")
	}

	return ts, nil
}

// Start starts TimeSyncer
func (ts *TimeSyncer) Start() error {
	go ts.schedule()

	ts.Log().Debug().Msg("started")

	return nil
}

// Stop stops TimeSyncer
func (ts *TimeSyncer) Stop() error {
	ts.Lock()
	defer ts.Unlock()

	ts.Log().Debug().Msg("trying to stop")

	if ts.stopChan != nil {
		ts.stopChan <- true
		close(ts.stopChan)
		ts.stopChan = nil
	}

	return nil
}

func (ts *TimeSyncer) schedule() {
	ticker := time.NewTicker(ts.interval)

end:
	for {
		select {
		case <-ts.stopChan:
			ticker.Stop()
			ts.Log().Debug().Msg("stopped")
			break end
		case <-ticker.C:
			ts.check()
		}
	}
}

// Offset returns the latest time offset.
func (ts *TimeSyncer) Offset() time.Duration {
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
