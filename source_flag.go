package config

import (
	"encoding/json"
	"errors"
	"flag"
	"log"
	"strings"
	"sync"

	"github.com/imdario/mergo"
)

type flagSource struct {
	sync.RWMutex

	// current changeset
	current *Snapshot
}

func (s *flagSource) Load() (*Snapshot, error) {
	s.RLock()
	defer s.RUnlock()

	return s.current, nil
}

func (s *flagSource) SetDecoder(decoder Decoder) {
	// cli arguments don't support decoding
}

func (s *flagSource) readFlags() (*Snapshot, error) {
	if !flag.Parsed() {
		return nil, errors.New("flag.Parse() must be called before")
	}

	var d map[string]interface{}
	visitFn := func(f *flag.Flag) {
		n := strings.ToLower(f.Name)
		keys := strings.FieldsFunc(n, split)
		reverse(keys)

		tmp := make(map[string]interface{})
		for i, k := range keys {
			if i == 0 {
				tmp[k] = f.Value
				continue
			}

			tmp = map[string]interface{}{k: tmp}
		}

		mergo.Map(&d, tmp) // need to sort error handling
		return
	}

	flag.VisitAll(visitFn)

	b, err := json.Marshal(d)
	if err != nil {
		return nil, err
	}

	return &Snapshot{Data: b}, nil
}

// Cli create config source from command line arguments
func Cli() Loader {

	s := &flagSource{}

	snap, err := s.readFlags()
	if err != nil {
		log.Fatal(err)
	}
	return &flagSource{
		current: snap,
	}
}

func split(r rune) bool {
	return r == '.' || r == '_'
}

func reverse(ss []string) {
	for i := len(ss)/2 - 1; i >= 0; i-- {
		opp := len(ss) - 1 - i
		ss[i], ss[opp] = ss[opp], ss[i]
	}
}
