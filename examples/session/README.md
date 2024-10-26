## 说明

这是把 `gkv` 作为 session 存储使用的例子，可以实现 session 的持久化。

使用 `gkv` 的价值，是在于可以存储任意结构体，存储和使用都是直接基于结构体实例的，而无须自己做一次 marshal 和 unmarshal。
可以看到，`sessionStore.Set` 和 `sessionStore.Get` 的方法签名如下:
```go
func (gd *gkv.Gkv[User]) Set(ctx context.Context, key string, v *User) error
func (gd *gkv.Gkv[User]) Get(ctx context.Context, key string) (*User, bool, error)
```

使用 `diskv` 作为底层存储，最大的好处是 diskv 直接基于本地文件，无须再安装各种依赖系统，比如 redis 等。另外，diskv 的存储是基于明文的，其格式非常简单，你甚至可以直接通过文件系统查看数据。

当然，从扩展性的角度看，`gkv` 的底层存储可以替换为 `redis`、`sqlite3`、`etcd3`、`bbolt` 等等。可见 [其他存储类型](#使用其他存储系统)

marshal 和 unmarshal 的默认实现是 `encoding/json`，但同样你可以自行实现 marshal 和 unmarshal 方法，以支持更多格式。(详见 gkv 中的说明)

### 动手指令

#### 启动 server 端
```bash
# pwd in ./examples
go run ./session/main.go
```

#### 请求 user 接口
```bash
curl localhost:8080/user
```
因为我们没有登录，所以得到了: `Unauthorized, no session_id in query`

#### 请求 login 接口
```bash
curl -X POST -H "Content-Type: application/json" -d '{"name":"baobei","user_id":"123456","slogan":"every day is a new day"}' http://localhost:8080/login
```
得到了如下:
```json
{"session_id":"MALiA4dNiX","user":{"session_id":"MALiA4dNiX","user_id":"123456","name":"baobei","avatar_url":"","slogan":"every day is a new day","LoginAt":"2024-10-26T11:19:09.833136+08:00","LastActive":"2024-10-26T11:19:09.833136+08:00","Expire":3600000000000,"ext":null}}
```

#### 再次请求 user 接口

```bash
curl -X GET 'http://127.0.0.1:8080/user?session_id=MALiA4dNiX'
```
> 注意，session_id 从上次请求中得到

这次就能正常获取信息了：
```json
{"user":{"session_id":"MALiA4dNiX","user_id":"123456","name":"baobei","avatar_url":"","slogan":"every day is a new day","LoginAt":"2024-10-26T11:19:09.833136+08:00","LastActive":"2024-10-26T11:22:19.509718+08:00","Expire":3600000000000,"ext":null}}
```

#### 登出
```bash
curl -X POST 'http://127.0.0.1:8080/logout?session_id=MALiA4dNiX'
```

然后再一次尝试访问：
```bash
curl -X GET 'http://127.0.0.1:8080/user?session_id=MALiA4dNiX'
```
就像我们从没登录过一样，会得到 401 Unauthorized 的错误: `Unauthorized, login first`

### 查看数据存储情况

启动 server 之后，就会在当前目录生成一个 `test/.data` 的子目录，其中有两个文件: `diskv.db` 和 `diskv.idx`，可以查看其中内容:

这就是数据文件，以 log 的形式存储了数据，非常简单易读：

```text
_set[MALiA4dNiX]{"session_id":"MALiA4dNiX","user_id":"123456","name":"baobei","avatar_url":"","slogan":"every day is a new day","LoginAt":"2024-10-26T11:19:09.833136+08:00","LastActive":"2024-10-26T11:19:09.833136+08:00","Expire":3600000000000,"ext":null}
_set[MALiA4dNiX]{"session_id":"MALiA4dNiX","user_id":"123456","name":"baobei","avatar_url":"","slogan":"every day is a new day","LoginAt":"2024-10-26T11:19:09.833136+08:00","LastActive":"2024-10-26T11:22:19.509718+08:00","Expire":3600000000000,"ext":null}
_del[MALiA4dNiX]
```

idx 文件则只有 `[maxlength:000128,keyslen:000500,x:0000000000000000000000000000]` 这段文本，其他都是空数据，是空数据的原因是我们的 session 信息已经删除了。

前面这段文本是 idx 的 meta 信息，表示 单个 key 的元信息占用空间为 `128bytes`，总共有 `500` 个预分配空间。(x 这部分为占位符，当前未使用)

### 使用其他存储系统

#### redis
redis 是互联网系统中最常用的 kv 存储系统了，基本上一涉及到分布式系统的 kv 缓存，首选就是 redis 了。
```go
import (
    "github.com/go-redis/redis/v8"
    "github.com/iamlongalong/diskv/kvstore/rediskv"
)

func x() {
    store := rediskv.NewStore(&redis.Options{Addr: "127.0.0.1:6379"}, "prefix")
    db := gkv.NewT(User{}, store)
}
```

#### bbolt
bbolt 是 etcd 的底层存储，是一个基于 LSM tree 的本地 kv 数据库，使用起来非常方便，性能、可靠性等也都非常不错。
```go
import (
    "github.com/iamlongalong/diskv/kvstore/bboltkv"
)

func x() {
    store, err := bboltkv.NewStore("./data/dbpath")
    if err != nil {log.Fatal(err)}

    db := gkv.NewT(User{}, store)
}
```

#### sqlite3
真实业务中，也有很多项目使用 sqlite 作为 kv 存储的，比如 vscode 的插件配置存储等等，轻量级数据持久化首选(十万量级以内)。

```go
import (
    "github.com/iamlongalong/diskv/kvstore/sqlitekv"
)

func x() {
    db, err := sqlitekv.NewStore("./data/dbpath")
    if err != nil {log.Fatal(err)}

    db := gkv.NewT(User{}, db)
}
```
