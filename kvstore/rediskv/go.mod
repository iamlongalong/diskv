module github.com/iamlongalong/diskv/kvstore/rediskv

go 1.22

toolchain go1.22.0

require github.com/go-redis/redis/v8 v8.11.5

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	golang.org/x/net v0.28.0 // indirect
	golang.org/x/sys v0.24.0 // indirect
)

replace google.golang.org/grpc/naming => google.golang.org/grpc v1.29.1
