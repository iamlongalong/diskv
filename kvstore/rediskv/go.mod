module github.com/iamlongalong/diskv/kvstore/rediskv

go 1.18

require (
	github.com/alicebob/miniredis/v2 v2.33.0
	github.com/go-redis/redis/v8 v8.11.5
	github.com/iamlongalong/diskv v0.0.0-20241025172318-af7cb836d4ed
)

require (
	github.com/alicebob/gopher-json v0.0.0-20200520072559-a9ecdc9d1d3a // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/yuin/gopher-lua v1.1.1 // indirect
	golang.org/x/net v0.28.0 // indirect
	golang.org/x/sys v0.24.0 // indirect
)

replace google.golang.org/grpc/naming => google.golang.org/grpc v1.29.1
