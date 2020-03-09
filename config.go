package config

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// Config define configuration functionality
type Config interface {
	Values

	Subscribe() <-chan struct{}
}

type config struct {
	sync.RWMutex

	options Options

	// latest snapshots
	snaps []*Snapshot

	// current merged snapshot
	snap *Snapshot

	// loaded snapshot
	values Values

	// subscriber of config changes
	subscribers []chan struct{}
}

// New initialize configuration with customizable options
// merge ordering start from first argument and last argument as final source
func New(opts ...Option) Config {
	init := Options{
		sources: make([]Loader, 0),
		reader:  &jsonReader{},
		merger:  &jsonMerger{},
	}

	// merge options
	options := mergeOptions(init, opts...)

	c := &config{
		options: options,
	}

	// read initial values
	if err := c.readAndMergeConfigs(); err != nil {
		log.Fatal("error read initial value: ", err)
	}

	if options.watch {
		// create subscribers
		c.subscribers = make([]chan struct{}, 0)

		fmt.Println("watch changes...")
		go c.watchChanges()
	}

	return c
}

func (c *config) watchChanges() {
	// run watchers
	for _, source := range c.options.sources {
		watcher, ok := source.(Watchable)
		if ok {
			go watcher.Watch(c.options.ctx)
		}
	}

	// periodically get snapshot of config sources
	for {
		select {
		case <-c.options.ctx.Done():
			// close subscriber
			c.RLock()
			for _, subscriber := range c.subscribers {
				close(subscriber)
			}
			c.RUnlock()

			return
		case <-time.After(c.options.watchDuration):
			if err := c.readAndMergeConfigs(); err != nil {
				log.Println("error update and merge config, err:", err)
			}

			break
		}
	}
}

func (c *config) readAndMergeConfigs() error {
	// collect all config snapshots
	snaps := make([]*Snapshot, len(c.options.sources))

	for i, source := range c.options.sources {
		snap, err := source.Load()
		if err != nil {
			return err
		}

		snaps[i] = snap
	}

	// compare with last snapshots
	var changed bool
	if c.snaps == nil || len(c.snaps) == 0 || len(c.snaps) != len(snaps) {
		changed = true
	} else {
		for i := 0; i < len(c.snaps); i++ {
			if c.snaps[i].Checksum() != snaps[i].Checksum() {
				changed = true
				break
			}
		}
	}

	if changed {
		// merge snapshot
		snap, err := c.options.merger.Merge(snaps...)
		if err != nil {
			return err
		}

		// read values
		values, err := c.options.reader.Read(snap)
		if err != nil {
			return err
		}

		// update current value
		c.Lock()
		c.snaps = snaps
		c.snap = snap
		c.values = values
		c.Unlock()

		// notify all subsribers
		c.RLock()
		for _, subscriber := range c.subscribers {
			select {
			case subscriber <- struct{}{}:
			default:
				// overflow
			}
		}
		c.RUnlock()
	}

	return nil
}

func (c *config) Subscribe() <-chan struct{} {
	c.Lock()
	defer c.Unlock()

	// buffered channel to prevent blocking
	s := make(chan struct{}, 1)
	c.subscribers = append(c.subscribers, s)

	return s
}

func (c *config) Bytes() []byte {
	c.RLock()
	defer c.RUnlock()

	return c.values.Bytes()
}

func (c *config) Get(path ...string) Value {
	c.RLock()
	defer c.RUnlock()

	return c.values.Get(path...)
}

func (c *config) Set(val interface{}, path ...string) {
	c.RLock()
	defer c.RUnlock()

	c.values.Set(val, path...)
}

func (c *config) Del(path ...string) {
	c.RLock()
	defer c.RUnlock()

	c.values.Del(path...)
}

func (c *config) Map() map[string]interface{} {
	c.RLock()
	defer c.RUnlock()

	return c.values.Map()
}

func (c *config) Scan(v interface{}) error {
	c.RLock()
	defer c.RUnlock()

	return c.values.Scan(v)
}
