package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	simple "github.com/bitly/go-simplejson"
	"github.com/imdario/mergo"
)

type jsonValue struct {
	*simple.Json
}

type jsonValues struct {
	sj   *simple.Json
	snap *Snapshot
}

type jsonReader struct{}

func (j *jsonReader) Read(snap *Snapshot) (Values, error) {
	if snap == nil {
		return nil, errors.New("snapshot is nil")
	}

	return newJSONValues(snap)
}

type jsonMerger struct{}

func (j *jsonMerger) Merge(snaps ...*Snapshot) (*Snapshot, error) {
	var merged map[string]interface{}

	for _, m := range snaps {
		if m == nil {
			continue
		}

		if m.Data == nil || len(m.Data) == 0 {
			continue
		}

		var data map[string]interface{}
		if err := json.Unmarshal(m.Data, &data); err != nil {
			return nil, err
		}

		if err := mergo.Map(&merged, data, mergo.WithOverride); err != nil {
			return nil, err
		}
	}

	b, err := json.Marshal(merged)
	if err != nil {
		return nil, err
	}

	snap := &Snapshot{
		Data: b,
	}

	return snap, nil
}

func newJSONValues(snap *Snapshot) (Values, error) {
	j := simple.New()
	if err := j.UnmarshalJSON(snap.Data); err != nil {
		j.SetPath(nil, string(snap.Data))
	}

	return &jsonValues{snap: snap, sj: j}, nil
}

func (j *jsonValue) Bool(def bool) bool {
	b, err := j.Json.Bool()
	if err == nil {
		return b
	}

	str, ok := j.Interface().(string)
	if !ok {
		return def
	}

	b, err = strconv.ParseBool(str)
	if err != nil {
		return def
	}

	return b
}

func (j *jsonValue) Int(def int) int {
	i, err := j.Json.Int()
	if err == nil {
		return i
	}

	str, ok := j.Interface().(string)
	if !ok {
		return def
	}

	i, err = strconv.Atoi(str)
	if err != nil {
		return def
	}

	return i
}

func (j *jsonValue) String(def string) string {
	return j.Json.MustString(def)
}

func (j *jsonValue) Float64(def float64) float64 {
	f, err := j.Json.Float64()
	if err == nil {
		return f
	}

	str, ok := j.Interface().(string)
	if !ok {
		return def
	}

	f, err = strconv.ParseFloat(str, 64)
	if err != nil {
		return def
	}

	return f
}

func (j *jsonValue) Duration(def time.Duration) time.Duration {
	v, err := j.Json.String()
	if err != nil {
		return def
	}

	value, err := time.ParseDuration(v)
	if err != nil {
		return def
	}

	return value
}

func (j *jsonValue) StringSlice(def []string) []string {
	v, err := j.Json.String()
	if err == nil {
		sl := strings.Split(v, ",")
		if len(sl) > 1 {
			return sl
		}
	}
	return j.Json.MustStringArray(def)
}

func (j *jsonValue) StringMap(def map[string]string) map[string]string {
	m, err := j.Json.Map()
	if err != nil {
		return def
	}

	res := map[string]string{}

	for k, v := range m {
		res[k] = fmt.Sprintf("%v", v)
	}

	return res
}

func (j *jsonValue) Scan(v interface{}) error {
	b, err := j.Json.MarshalJSON()
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

func (j *jsonValue) Bytes() []byte {
	b, err := j.Json.Bytes()
	if err != nil {
		// try return marshalled
		b, err = j.Json.MarshalJSON()
		if err != nil {
			return []byte{}
		}
		return b
	}
	return b
}

func (j *jsonValues) Get(path ...string) Value {
	return &jsonValue{j.sj.GetPath(resolvePath(path)...)}
}

func (j *jsonValues) Del(path ...string) {
	path = resolvePath(path)

	// delete the tree?
	if len(path) == 0 {
		j.sj = simple.New()
		return
	}

	if len(path) == 1 {
		j.sj.Del(path[0])
		return
	}

	vals := j.sj.GetPath(path[:len(path)-1]...)
	vals.Del(path[len(path)-1])
	j.sj.SetPath(path[:len(path)-1], vals.Interface())
	return
}

func (j *jsonValues) Set(val interface{}, path ...string) {
	j.sj.SetPath(resolvePath(path), val)
}

func (j *jsonValues) Bytes() []byte {
	b, _ := j.sj.MarshalJSON()
	return b
}

func (j *jsonValues) Map() map[string]interface{} {
	m, _ := j.sj.Map()
	return m
}

func (j *jsonValues) Scan(v interface{}) error {
	b, err := j.sj.MarshalJSON()
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

func resolvePath(path []string) []string {
	if path == nil || len(path) == 0 {

		return []string{}
	}

	if len(path) == 1 {
		return strings.Split(path[0], ".")
	}

	return path
}
