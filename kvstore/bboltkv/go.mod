module github.com/iamlongalong/diskv/kvstore/bboltkv

go 1.22

toolchain go1.22.0

require go.etcd.io/bbolt v1.3.11

require github.com/stretchr/testify v1.9.0 // indirect

require (
	golang.org/x/sync v0.8.0 // indirect
	golang.org/x/sys v0.24.0 // indirect
)

replace google.golang.org/grpc/naming => google.golang.org/grpc v1.29.1
