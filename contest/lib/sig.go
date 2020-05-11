package contestlib

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/spikeekips/mitum/util/logging"
)

func ConnectSignal(exitHooks *[]func(), log logging.Logger) {
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)

	go func() {
		s := <-sigc

		log.Warn().
			Str("sig", s.String()).
			Msg("contest stopped by force")

		for _, h := range *exitHooks {
			h()
		}

		os.Exit(1)
	}()
}
