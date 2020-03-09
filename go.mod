module github.com/wjaoss/config

go 1.13

replace (
	github.com/coreos/go-systemd => github.com/coreos/go-systemd/v22 v22.0.0
	google.golang.org/grpc => google.golang.org/grpc v1.26.0
)

require (
	github.com/bitly/go-simplejson v0.5.0
	github.com/coreos/etcd v3.3.18+incompatible
	github.com/coreos/go-systemd v0.0.0-00010101000000-000000000000 // indirect
	github.com/coreos/pkg v0.0.0-20180928190104-399ea9e2e55f // indirect
	github.com/fsnotify/fsnotify v1.4.7
	github.com/gogo/protobuf v1.3.1 // indirect
	github.com/google/uuid v1.1.1 // indirect
	github.com/imdario/mergo v0.3.8
	github.com/wjaoss/x v0.0.0-20200309071043-647477a4c0ad
	go.uber.org/zap v1.14.0 // indirect
	google.golang.org/genproto v0.0.0-20200306153348-d950eab6f860 // indirect
	gopkg.in/yaml.v3 v3.0.0-20200121175148-a6ecf24a6d71
)
