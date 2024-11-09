# diskv

一般我们要使用 kv 存储时，多用 redis、rocksdb、leveldb 等等，这些存储各有优势，能满足绝大多数的需求。

但这些存储系统有有一个特点： 底层都是基于二进制存储，需要专门的工具才能查看相关数据。
这一操作核心是提升读写的性能，和压缩存储大小。

但有些时候，我们希望用更简单的方式去存储数据，简单到我们肉眼就能读懂的程度。
当我们对 "数据丢失" 和 "数据恢复" 操作不熟练时，心中难免多有不安。

于是，我希望实现一个极简的 kv 存储系统，满足两个基本理念：
- 以明文方式存储数据，让数据可读性高
- 极力减小对内存的占用，所有操作，尽量都通过直接操作文件系统完成
- 极力减少外部依赖 (当前无外部依赖库)

从文件存储上看，如下:
<p align="center">
  <img src="assets/db.png" width="300" >
  <img src="assets/idx.png" width="405" >
</p>

## 快速上手

### 创建一个 db
```go
db, err := diskv.CreateDB(ctx, &diskv.CreateConfig{
    Dir: "/tmp/diskv",
    KeysLen: 100, // 预分配的 key 的数量
    MaxLen: 64, // idx 块的最大长度
})
```

### 数据操作
```go
// 存储
err = db.Set(ctx, key, bytesValue)
err = db.SetString(ctx, key, stringValue)

// 读取
bytesValue, err := db.Get(ctx, key)
stringValue, err := db.GetString(ctx, key)

// 删除
ok, err := db.Delete(ctx, key)

// 检查是否存在
has, err := db.Has(ctx, key)

// 遍历
err = db.ForEach(ctx, func(key string, value []byte) bool {
    return true // 继续遍历
})

```

### 文件迁移

由于 key 的空间大小是预分配的，若 key 的数量逐渐增加，达到预分配大小的 75% 以上时(负载 75%)，性能就会受到影响。
实测的 benchmark 测试表明，50% 负载时为 10w QPS, 80% 负载时为 9.2w QPS, 当负载达到 100% 时，性能从原 10w QPS 降低到了 2.1w QPS，详见 [benchmark](./benchmark.txt) 文件。

```go
db, err := diskv.CreateDB(ctx, &diskv.CreateConfig{
    Dir: "/tmp/diskv",
    MaxLen:  64,
    KeysLen: 1000, // 预分配的 key 的数量
})

// 主动迁移
err := db.MigrateIdx(ctx, &diskv.CreateConfig{
    MaxLen:  64,
    KeysLen: 5000, // 预分配的 key 的数量
})

```

当前的 DB 文件，是以 log 的模式增量写入的，idx 只记录最后一次 key 的位置，因此，当修改、删除等操作频繁发生时，log 文件就会持续增长。可通过 `db.MigrateValue()` 进行主动迁移，这个操作会根据 idx 记录的所有 key 信息，把有效的 value 移动到新的 db 文件中，并删除旧的 db 文件。

```go
err := db.MigrateValue(ctx)
```

上述的迁移都是阻塞进行的，迁移过程中无法读写数据。
迁移后的文件名不会变动，老的文件会以 `*._bak` 的后缀名保存最近一次的迁移文件。

## 带类型存储

详情见 [gkv](./gkv/README.md) 目录. 

## TODO

- [x] 支持 idx 文件迁移
- [x] 支持 db 文件压缩 + 迁移
- [x] 支持存储特定结构体的接口
- [ ] 实现一个 list 存储系统 (glist)
- [ ] 考虑实现一个基于 struct 的存储系统 (gobject)

## 这个项目要做什么

在一个开发语言的生态下，往往有很多内存级别的数据结构工具，但当我们希望持久化数据时，就会变成对复杂的外部系统的搭建和维护。 比如，你想实现一个带存储能力的 pipe 时，就可能去搭建一套 redis 或者 xxmq 出来。

我们希望，能有一套基于文件存储的各类数据结构，可以非常方便地在本地文件系统上做到 kv、数组、列表、队列、优先队列、栈、集合、字典、有序集合、位图、块存储等等。

扩展的话，还可能有一些需要用到存储的系统，比如 cron、锁、session(kv with cache) 等。
