package config

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util/isvalid"
)

type ErrorType string

const (
	ErrorTypeError          ErrorType = "error"
	ErrorTypeWrongBlockHash ErrorType = "wrong-block"
)

func (et ErrorType) IsValid([]byte) error {
	switch et {
	case ErrorTypeError, ErrorTypeWrongBlockHash:
		return nil
	default:
		return isvalid.InvalidError.Errorf("unknown ErrorType, %q", et)
	}
}

type ProposalProcessor interface {
	ProposalProcessorType() string
}

type DefaultProposalProcessor struct{}

func (DefaultProposalProcessor) ProposalProcessorType() string {
	return "default"
}

type ErrorProposalProcessor struct {
	WhenPreparePoints []ErrorPoint
	WhenSavePoints    []ErrorPoint
}

func (ErrorProposalProcessor) ProposalProcessorType() string {
	return "error"
}

type ErrorPoint struct {
	Type   ErrorType   `json:"type"`
	Height base.Height `json:"height"`
	Round  base.Round  `json:"round"`
}
