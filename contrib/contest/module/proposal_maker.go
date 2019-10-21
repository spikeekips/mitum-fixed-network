package contest_module

var ProposalMakers []string

func init() {
	ProposalMakers = append(ProposalMakers,
		"DefaultProposalMaker",
		"ConditionProposalMaker",
	)
}
