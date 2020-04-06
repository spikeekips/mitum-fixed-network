package isaac

type SyncerState uint8

const (
	_ SyncerState = iota
	SyncerCreated
	SyncerPreparing
	SyncerPrepared
	SyncerSaving
	SyncerSaved
)

func (ss SyncerState) String() string {
	switch ss {
	case SyncerCreated:
		return "syncer-created"
	case SyncerPreparing:
		return "syncer_preparing"
	case SyncerPrepared:
		return "syncer-prepared"
	case SyncerSaving:
		return "syncer-saving"
	case SyncerSaved:
		return "syncer-saved"
	default:
		return "<unknown sync state>"
	}
}

type Syncer interface {
	Prepare(Manifest /* base manifest */) error
	Save() error
	HeightFrom() Height
	HeightTo() Height
	State() SyncerState
	TailManifest() Manifest
	Close() error
}

type syncerFetchBlockError struct {
	err     error
	heights []Height
	node    Address
	missing []Height
	blocks  []Block
}

func (fm *syncerFetchBlockError) Error() string {
	if fm.err == nil {
		return ""
	}

	return fm.err.Error()
}
