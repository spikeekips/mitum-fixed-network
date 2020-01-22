package isvalid

type IsValider interface {
	IsValid([]byte) error
}
