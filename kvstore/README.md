## 说明

kvstore 定义了基本的接口，目前实现了如下几种存储引擎:

- sqlite3
- redis
- etcd3
- bbolt

gkv 是基于 kvstore 的一个 具体类型 的 kv 存储，详情可见 [gkv](../gkv/README.md)
