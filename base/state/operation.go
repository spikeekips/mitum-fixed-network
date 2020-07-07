package state

type OperationProcesser interface {
	ProcessOperation(
		getState func(key string) (StateUpdater, error),
		setState func(StateUpdater) error,
	) error
}
