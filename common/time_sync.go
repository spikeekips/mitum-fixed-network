package common

import (
	"sync"
	"time"

	"github.com/beevik/ntp"
	"github.com/rs/zerolog"
)

var (
	allowedTimeSyncOffset = time.Duration(time.Millisecond * 500)
	timeSyncer            *TimeSyncer
)

type TimeSyncer struct {
	sync.RWMutex
	*Logger
	server   string
	offset   time.Duration
	stopChan chan bool
	interval time.Duration
}

func NewTimeSyncer(server string, checkInterval time.Duration) (*TimeSyncer, error) {
	_, err := ntp.Query(server)
	if err != nil {
		return nil, err
	}

	return &TimeSyncer{
		Logger: NewLogger(func(c zerolog.Context) zerolog.Context {
			return c.
				Str("module", "time-syncer").
				Str("server", server).
				Dur("interval", checkInterval)
		}),
		server:   server,
		interval: checkInterval,
		stopChan: make(chan bool),
	}, nil
}

func SetTimeSyncer(syncer *TimeSyncer) {
	timeSyncer = syncer
}

func (s *TimeSyncer) Start() error {
	s.Log().Debug().Msg("trying to start time-syncer")

	go s.schedule()

	s.Log().Debug().Msg("time-syncer started")
	return nil
}

func (s *TimeSyncer) Stop() error {
	s.Lock()
	defer s.Unlock()

	s.Log().Debug().Msg("trying to stop time-syncer")
	if s.stopChan != nil {
		s.stopChan <- true
		close(s.stopChan)
		s.stopChan = nil
	}

	return nil
}

func (s *TimeSyncer) schedule() {
	ticker := time.NewTicker(s.interval)

end:
	for {
		select {
		case <-s.stopChan:
			ticker.Stop()
			s.Log().Debug().Msg("time-syncer stopped")
			break end
		case <-ticker.C:
			s.check()
		}
	}
}

func (s *TimeSyncer) Offset() time.Duration {
	return s.offset
}

func (s *TimeSyncer) check() {
	s.Lock()
	defer s.Unlock()

	response, err := ntp.Query(s.server)
	if err != nil {
		s.Log().Error().Err(err).Msg("failed to query")
		return
	}

	if err := response.Validate(); err != nil {
		s.Log().Error().
			Err(err).
			Interface("response", response).
			Msg("failed to validate response")
		return
	}
	defer func() {
		s.Log().Debug().
			Interface("response", response).
			Dur("offset", s.offset).
			Msg("time checked")
	}()

	if s.offset < 1 {
		s.offset = response.ClockOffset
		return
	}

	diff := s.offset - response.ClockOffset
	if diff > 0 {
		if diff < allowedTimeSyncOffset {
			return
		}
	} else if diff < 0 {
		if diff > allowedTimeSyncOffset*-1 {
			return
		}
	}

	s.offset = response.ClockOffset
}

func Now() Time {
	if timeSyncer == nil {
		return Time{Time: time.Now()}
	}

	return Time{Time: time.Now().Add(timeSyncer.Offset())}
}
