package common

import (
	"sync"
	"time"

	"github.com/beevik/ntp"
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
		Logger: NewLogger(
			log,
			"module", "time-syncer",
			"server", server,
			"interval", checkInterval,
		),
		server:   server,
		interval: checkInterval,
		stopChan: make(chan bool),
	}, nil
}

func SetTimeSyncer(syncer *TimeSyncer) {
	timeSyncer = syncer
	log.Debug("common.timeSyncer is set")
}

func (s *TimeSyncer) Start() error {
	s.Log().Debug("trying to start time-syncer")

	go s.schedule()

	s.Log().Debug("time-syncer started")
	return nil
}

func (s *TimeSyncer) Stop() error {
	s.Lock()
	defer s.Unlock()

	s.Log().Debug("trying to stop time-syncer")
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
			s.Log().Debug("time-syncer stopped")
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
		s.Log().Error("failed to query", "error", err)
		return
	}

	if err := response.Validate(); err != nil {
		s.Log().Error("failed to validate response", "response", response, "error", err)
		return
	}
	defer func() {
		s.Log().Debug("time checked", "response", response, "offset", s.offset)
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
