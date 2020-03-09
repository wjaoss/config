package config

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/coreos/etcd/clientv3"
	etcd "github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/mvcc/mvccpb"
)

type EtcdOption struct {
	Prefix   string
	Username string
	Password string

	DialTimeout time.Duration
}

type sourceEtcd struct {
	host   string
	prefix string
	sync.RWMutex

	client *etcd.Client

	// decoder
	decoder Decoder

	// current changeset
	current *Snapshot
}

func (s *sourceEtcd) Load() (*Snapshot, error) {
	s.RLock()
	current := s.current
	s.RUnlock()

	if current != nil {
		return current, nil
	}

	snap, err := s.readConfig()
	if err != nil {
		return nil, err
	}

	s.Lock()
	s.current = snap
	s.Unlock()

	return s.current, nil
}

func (s *sourceEtcd) SetDecoder(decoder Decoder) {
	s.decoder = decoder
}

func (s *sourceEtcd) handleEvent(evs []*etcd.Event) (*Snapshot, error) {
	s.RLock()
	data := s.current.Data
	s.RUnlock()

	var vals map[string]interface{}
	if err := json.Unmarshal(data, &vals); err != nil {
		return nil, err
	}

	d := makeEvMap(vals, evs, s.prefix)

	// pack the changeset
	b, err := json.Marshal(d)
	if err != nil {
		return nil, err
	}

	return &Snapshot{
		Data: b,
	}, nil
}

func (s *sourceEtcd) Watch(ctx context.Context) {
	ch := s.client.Watch(ctx, s.prefix, etcd.WithPrefix())

	for {
		select {
		case <-ctx.Done():
			return
		case rsp, ok := <-ch:
			if !ok {
				return
			}

			if snap, err := s.handleEvent(rsp.Events); err == nil {
				s.Lock()
				s.current = snap
				s.Unlock()
			}
		}
	}
}

func (s *sourceEtcd) readConfig() (*Snapshot, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()

	rsp, err := s.client.Get(ctx, s.prefix, etcd.WithPrefix())
	if err != nil {
		return nil, err
	}

	if rsp == nil || len(rsp.Kvs) == 0 {
		return nil, fmt.Errorf("source not found: %s", s.prefix)
	}

	kvs := make([]*mvccpb.KeyValue, 0, len(rsp.Kvs))
	for _, v := range rsp.Kvs {
		kvs = append(kvs, (*mvccpb.KeyValue)(v))
	}

	data := makeMap(kvs, s.prefix)
	b, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	return &Snapshot{
		Data: b,
	}, nil
}

// Etcd create etcd source loader
func Etcd(endpoints string, vars ...EtcdOption) Loader {
	// default config
	var username, password, prefix string
	dialTimeout := time.Second * 5

	if len(vars) > 0 {
		username = vars[0].Username
		password = vars[0].Password
		prefix = vars[0].Prefix

		if vars[0].DialTimeout > 0 {
			dialTimeout = vars[0].DialTimeout
		}
	}

	addrs := strings.Split(endpoints, ",")
	if len(addrs) == 0 {
		addrs = []string{"localhost:2379"}
	}

	clientConfig := etcd.Config{
		Endpoints:   addrs,
		DialTimeout: dialTimeout,
		Username:    username,
		Password:    password,
	}

	c, err := etcd.New(clientConfig)
	if err != nil {
		log.Fatal(err)
	}

	return &sourceEtcd{
		client: c,
		prefix: prefix,
	}
}

func makeEvMap(data map[string]interface{}, kv []*clientv3.Event, stripPrefix string) map[string]interface{} {
	if data == nil {
		data = make(map[string]interface{})
	}

	for _, v := range kv {
		switch mvccpb.Event_EventType(v.Type) {
		case mvccpb.DELETE:
			data = update(data, (*mvccpb.KeyValue)(v.Kv), "delete", stripPrefix)
		default:
			data = update(data, (*mvccpb.KeyValue)(v.Kv), "insert", stripPrefix)
		}
	}

	return data
}

func makeMap(kv []*mvccpb.KeyValue, stripPrefix string) map[string]interface{} {
	data := make(map[string]interface{})

	for _, v := range kv {
		data = update(data, v, "put", stripPrefix)
	}

	return data
}

func update(data map[string]interface{}, v *mvccpb.KeyValue, action, stripPrefix string) map[string]interface{} {
	// remove prefix if non empty, and ensure leading / is removed as well
	vkey := strings.TrimPrefix(strings.TrimPrefix(string(v.Key), stripPrefix), "/")
	// split on prefix
	haveSplit := strings.Contains(vkey, "/")
	keys := strings.Split(vkey, "/")

	var vals interface{}
	if err := json.Unmarshal(v.Value, &vals); err != nil {
		vals = string(v.Value)
	}

	if !haveSplit && len(keys) == 1 {
		key := keys[0]

		switch action {
		case "delete":
			data = make(map[string]interface{})
		default:
			if key == "" {
				v, ok := vals.(map[string]interface{})
				if ok {
					data = v
				}
			} else {
				data[key] = vals
			}

		}
		return data
	}

	// set data for first iteration
	kvals := data
	// iterate the keys and make maps
	for i, k := range keys {
		kval, ok := kvals[k].(map[string]interface{})
		if !ok {
			// create next map
			kval = make(map[string]interface{})
			// set it
			kvals[k] = kval
		}

		// last key: write vals
		if l := len(keys) - 1; i == l {
			switch action {
			case "delete":
				delete(kvals, k)
			default:
				kvals[k] = vals
			}
			break
		}

		// set kvals for next iterator
		kvals = kval
	}

	return data
}
