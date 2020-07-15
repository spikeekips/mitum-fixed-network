package contestlib

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

var ExitHooks *exitHooks

type exitHooks struct {
	sync.RWMutex
	hooks []func()
}

func (eh *exitHooks) Add(f ...func()) *exitHooks {
	eh.Lock()
	defer eh.Unlock()

	eh.hooks = append(eh.hooks, f...)

	return eh
}

func (eh *exitHooks) Hooks() []func() {
	eh.RLock()
	defer eh.RUnlock()

	return eh.hooks
}

func (eh *exitHooks) Run() {
	eh.RLock()
	defer eh.RUnlock()

	for _, h := range eh.hooks {
		h()
	}
}

func init() {
	ExitHooks = &exitHooks{}
}

func ConnectSignal() {
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
		syscall.SIGHUP,
	)

	go func() {
		s := <-sigc

		defer func() {
			os.Exit(1)
		}()

		defer func() {
			ExitHooks.Run()

			_, _ = fmt.Fprintf(os.Stderr, "stopped by force: %v\n", s)
		}()
	}()
}
