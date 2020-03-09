package config

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	yaml "gopkg.in/yaml.v3"
)

var (
	fileTransformers = map[string]fileTransformer{
		"yaml": &yamlFileTransformer{},
		"json": &jsonFileTransformer{},
	}
)

// fileTransformer transform file to json stream based on its extension
type fileTransformer interface {
	Transform([]byte) ([]byte, error)
}

type jsonFileTransformer struct{}

func (l *jsonFileTransformer) Transform(src []byte) ([]byte, error) {
	// noop
	return src, nil
}

type yamlFileTransformer struct{}

func (l *yamlFileTransformer) Transform(src []byte) ([]byte, error) {
	dest := make(map[string]interface{})
	if err := yaml.Unmarshal(src, &dest); err != nil {
		return nil, err
	}

	return json.Marshal(dest)
}

type fileSource struct {
	file   string
	format string
	watch  bool
	sync.RWMutex

	// decoder
	decoder Decoder

	// current changeset
	current *Snapshot
}

// Load read initial change set
func (s *fileSource) Load() (*Snapshot, error) {
	s.RLock()
	current := s.current
	s.RUnlock()

	if current != nil {
		return current, nil
	}

	snap, err := s.readFile()
	if err != nil {
		return nil, err
	}

	s.Lock()
	s.current = snap
	s.Unlock()

	return snap, nil
}

func (s *fileSource) SetDecoder(decoder Decoder) {
	s.decoder = decoder
}

// readFile read configuration file and put into current snapshot
func (s *fileSource) readFile() (*Snapshot, error) {
	b, err := ioutil.ReadFile(s.file)
	if err != nil {
		return nil, err
	}

	// if decoder assigned, then decode stream before transforming
	if s.decoder != nil {
		b = s.decoder.Decode(b)
	}

	// transform based on format (ext)
	transformer, ok := fileTransformers[s.format]
	if !ok {
		// fallback to json
		transformer = &jsonFileTransformer{}
	}

	b, err = transformer.Transform(b)
	if err != nil {
		return nil, err
	}

	snap := &Snapshot{Data: b}

	return snap, nil
}

func (s *fileSource) Watch(ctx context.Context) {
	if !s.watch {
		return
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}

	watcher.Add(s.file)

	for {
		select {
		case <-ctx.Done():
			watcher.Close()
			return
		case event, ok := <-watcher.Events:
			if !ok {
				break
			}

			if event.Op&fsnotify.Write == fsnotify.Write {
				snap, err := s.readFile()
				if err != nil {
					log.Fatal(err)
				}

				s.Lock()
				if s.current.Checksum() != snap.Checksum() {
					s.current = snap
				}
				s.Unlock()
			}
		}
	}
}

// File create config source from give file
func File(file string, watch ...bool) Loader {
	ext := filepath.Ext(file)
	if len(ext) > 0 {
		ext = ext[1:]
	}

	s := &fileSource{
		file:   file,
		format: strings.ToLower(ext),
		watch:  len(watch) > 0 && watch[0],
	}

	return s
}
