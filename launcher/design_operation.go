package launcher

type OperationDesign map[string]interface{}

func NewOperationDesign() OperationDesign {
	return OperationDesign{}
}

func (cd *OperationDesign) IsValid([]byte) error {
	return nil
}
