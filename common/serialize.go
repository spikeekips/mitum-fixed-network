package common

type Serializable interface {
	Hash() (Hash, error)
	Serialize() ([]byte, error)
	Unserialize([]byte, interface{}) error
}
