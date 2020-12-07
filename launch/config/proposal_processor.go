package config

type ProposalProcessor interface {
	ProposalProcessorType() string
}

type DefaultProposalProcessor struct{}

func (no DefaultProposalProcessor) ProposalProcessorType() string {
	return "default"
}
