package config

import "time"

// Value represent config field value of any type
type Value interface {
	Bool(def bool) bool
	Int(def int) int
	String(def string) string
	Float64(def float64) float64
	Duration(def time.Duration) time.Duration
	StringSlice(def []string) []string
	StringMap(def map[string]string) map[string]string
	Scan(val interface{}) error
	Bytes() []byte
}

// Values contains collection of config value
type Values interface {
	Bytes() []byte
	Get(path ...string) Value
	Set(val interface{}, path ...string)
	Del(path ...string)
	Map() map[string]interface{}
	Scan(v interface{}) error
}

// Reader reads snapshot into config values
type Reader interface {
	Read(*Snapshot) (Values, error)
}

// Merger concatenate multiple config config snapshot
type Merger interface {
	Merge(...*Snapshot) (*Snapshot, error)
}
