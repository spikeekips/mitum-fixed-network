package contestlib

type ComponentDesign struct {
	Suffrage          *SuffrageDesign          `yaml:",omitempty"`
	ProposalProcessor *ProposalProcessorDesign `yaml:"proposal-processor,omitempty"`
}

func NewComponentDesign(o *ComponentDesign) *ComponentDesign {
	if o != nil {
		return &ComponentDesign{
			Suffrage:          o.Suffrage,
			ProposalProcessor: o.ProposalProcessor,
		}
	}

	return &ComponentDesign{}
}

func (cc *ComponentDesign) IsValid([]byte) error {
	if cc.Suffrage == nil {
		cc.Suffrage = NewSuffrageDesign()
	}

	if err := cc.Suffrage.IsValid(nil); err != nil {
		return err
	}

	if cc.ProposalProcessor == nil {
		cc.ProposalProcessor = NewProposalProcessorDesign()
	}

	if err := cc.ProposalProcessor.IsValid(nil); err != nil {
		return err
	}

	return nil
}

func (cc *ComponentDesign) Merge(b *ComponentDesign) error {
	if cc.Suffrage == nil {
		if b == nil {
			cc.Suffrage = NewSuffrageDesign()
		} else {
			cc.Suffrage = b.Suffrage
		}
	}

	if cc.ProposalProcessor == nil {
		if b == nil {
			cc.ProposalProcessor = NewProposalProcessorDesign()
		} else {
			cc.ProposalProcessor = b.ProposalProcessor
		}
	}

	return nil
}
