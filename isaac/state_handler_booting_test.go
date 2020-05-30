package isaac

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
)

type testStateBootingHandler struct {
	baseTestStateHandler
}

func (t *testStateBootingHandler) TestWithBlock() {
	cs, err := NewStateBootingHandler(t.localstate)
	t.NoError(err)

	stateChan := make(chan StateChangeContext)
	cs.SetStateChan(stateChan)

	doneChan := make(chan struct{})
	go func() {
	end:
		for {
			select {
			case <-time.After(time.Second):
				break end
			case ctx := <-stateChan:
				if ctx.To() == base.StateJoining {
					doneChan <- struct{}{}

					break end
				}
			}
		}
	}()

	t.NoError(cs.Activate(NewStateChangeContext(base.StateStopped, base.StateBooting, nil, nil)))
	defer func() {
		_ = cs.Deactivate(StateChangeContext{})
	}()

	select {
	case <-time.After(time.Second):
		t.NoError(xerrors.Errorf("timeout to wait to be finished"))
	case <-doneChan:
		break
	}
}

func (t *testStateBootingHandler) TestWithoutBlock() {
	cs, err := NewStateBootingHandler(t.localstate)
	t.NoError(err)
	t.NoError(t.localstate.Storage().Clean())

	stateChan := make(chan StateChangeContext)
	cs.SetStateChan(stateChan)

	doneChan := make(chan struct{})
	go func() {
	end:
		for {
			select {
			case <-time.After(time.Second):
				break end
			case ctx := <-stateChan:
				if ctx.To() == base.StateSyncing {
					doneChan <- struct{}{}

					break end
				}
			}
		}
	}()

	t.NoError(cs.Activate(NewStateChangeContext(base.StateStopped, base.StateBooting, nil, nil)))
	defer func() {
		_ = cs.Deactivate(StateChangeContext{})
	}()

	select {
	case <-time.After(time.Second):
		t.NoError(xerrors.Errorf("timeout to wait to be finished"))
	case <-doneChan:
		break
	}
}

func TestStateBootingHandler(t *testing.T) {
	suite.Run(t, new(testStateBootingHandler))
}
