package common

import "encoding/json"

type Vote int

const (
	VoteNOP    Vote = -1
	VoteNONE   Vote = 0
	VoteYES    Vote = 1
	VoteEXPIRE Vote = 2
)

func (v Vote) String() string {
	switch v {
	case VoteNOP:
		return "nop"
	case VoteYES:
		return "yes"
	case VoteEXPIRE:
		return "exp"
	default:
		return ""
	}
}

func (v Vote) MarshalText() ([]byte, error) {
	if v == VoteNONE {
		return nil, InvalidVoteError
	}

	return json.Marshal(v.String())
}

func (v *Vote) UnmarshalText(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}

	switch s {
	case "nop":
		*v = VoteNOP
	case "yes":
		*v = VoteYES
	case "exp":
		*v = VoteEXPIRE
	default:
		return InvalidVoteError
	}

	return nil
}
