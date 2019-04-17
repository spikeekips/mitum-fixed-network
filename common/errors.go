package common

const (
	_ uint = iota
	OverflowErrorCode
	UnknownSealTypeCode
	InvalidVoteCode
	JSONUnmarshalCode
	NotImplementedCode
	InvalidHashCode
)

var (
	OverflowError        Error = NewError("common", OverflowErrorCode, "overflow number")
	UnknownSealTypeError Error = NewError("common", UnknownSealTypeCode, "unknown seal type found")
	InvalidVoteError     Error = NewError("common", InvalidVoteCode, "invalid vote found")
	JSONUnmarshalError   Error = NewError("common", JSONUnmarshalCode, "failed json unmarshal")
	NotImplementedError  Error = NewError("common", NotImplementedCode, "not implemented")
	InvalidHashError     Error = NewError("common", InvalidHashCode, "invalid has found")
)
