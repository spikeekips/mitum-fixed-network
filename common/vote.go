package common

import "encoding/json"

type Vote int

const (
	VoteNO   Vote = -1
	VoteNONE Vote = 0
	VoteYES  Vote = 1
	VoteEXP  Vote = 2
)

func (v Vote) String() string {
	switch v {
	case VoteNO:
		return "no"
	case VoteYES:
		return "yes"
	case VoteEXP:
		return "expire"
	default:
		return ""
	}
}

func (v Vote) MarshalJSON() ([]byte, error) {
	if v == VoteNONE {
		return nil, InvalidVoteError
	}

	return json.Marshal(v.String())
}

func (v *Vote) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}

	switch s {
	case "no":
		*v = VoteNO
	case "yes":
		*v = VoteYES
	case "expire":
		*v = VoteEXP
	default:
		return InvalidVoteError
	}

	return nil
}
