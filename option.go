package config

import "context"

import "time"

// Option define method to modify config options
type Option func(o *Options)

// Options of configuration package
type Options struct {
	// config reader
	reader Reader

	// config merger
	merger Merger

	// source loaders
	sources []Loader

	// watcher should be configured along with running context
	watch         bool
	watchDuration time.Duration
	ctx           context.Context
}

// EnableWatcher set configuration should watch configuration changes
func EnableWatcher(ctx context.Context, d ...time.Duration) Option {
	def := time.Second * 30
	if len(d) > 0 && d[0] > 0 {
		def = d[0]
	}

	return func(o *Options) {
		o.watch = true
		o.watchDuration = def
		o.ctx = ctx
	}
}

// WithSource add new configuration source
func WithSource(loader Loader, decoder ...Decoder) Option {
	return func(o *Options) {
		if len(decoder) > 0 {
			loader.SetDecoder(decoder[0])
		}

		o.sources = append(o.sources, loader)
	}
}

// WithReader overrides default config reader
func WithReader(reader Reader) Option {
	return func(o *Options) {
		o.reader = reader
	}
}

// WithMerger overrides default config merger
func WithMerger(merger Merger) Option {
	return func(o *Options) {
		o.merger = merger
	}
}

func mergeOptions(dest Options, opts ...Option) Options {
	for _, opt := range opts {
		opt(&dest)
	}

	return dest
}
