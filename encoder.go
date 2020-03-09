package config

type Encoder interface {
	Encode(interface{}) ([]byte, error)
	Decode([]byte, interface{}) error
}
