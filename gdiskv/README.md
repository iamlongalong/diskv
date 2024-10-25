## gdiskv

generic disk kv store

让 kv store 直接读取具体类型，而不是用 []byte 或 string 转换。

### 快速上手

#### 创建一个用来存储自定义结构体的db

```go
type User struct {
    Name string
    Age  int
}

// 创建底层存储器
diskdb, err := diskv.CreateDB(ctx, &gdiskv.CreateConfig{Dir: "/tmp/gdiskv", KeysLen: 100, MaxLen:  64})
if err != nil {log.Fatal(err)}

// 创建 User{} 这个结构体的 db
db := gdiskv.NewT(User{}, diskdb)
// 你也可以使用 `diskv.New` 创建一个 db
// db := gdiskv.New[User](diskdb)

// 存储
err = db.Set(ctx, "user1", User{Name: "user1", Age: 18})
if err != nil {log.Fatal(err)}

// 读取
user, ok, err := db.Get(ctx, "user1")
if err != nil {log.Fatal(err)}

if !ok {log.Fatal("not found")}
fmt.Printf("%+v\n", user)

// 删除 by set nil
err = db.Set(ctx, "user1", nil)
if err != nil {log.Fatal(err)}
```

若你希望一个 db 可以用来存不同类型的数据，则可以使用 NDiskv, 如下：

```go
type User struct {
    Name string
    Age  int
}

type Dog struct {
    Name string
    Color string
    Category string
}

diskdb, err := diskv.CreateDB(ctx, &gdiskv.CreateConfig{Dir: "/tmp/gdiskv", KeysLen: 100, MaxLen:  64})
if err != nil {log.Fatal(err)}

// 创建一个存储器
ndb := gdiskv.NewNDiskv(diskdb)

// 存储
user01 := User{
    Name: "testname",
    Age:  18,
}
err := ndb.Set(ctx, "user01", user01)
if err != nil {log.Fatal(err)}

dog01 := Dog{
    Name: "lala",
    Color: "colorful",
}
err := ndb.Set(ctx, "dog01", user01)
if err != nil {log.Fatal(err)}

// 读取
userx := User{}
ok, err := ndb.Get(ctx, "user01", &userx)
if err != nil {log.Fatal(err)}
if !ok {log.Fatal("not found")}

dogx := Dog{}
ok, err := ndb.Get(ctx, "dog01", &dogx)
if err != nil {log.Fatal(err)}
if !ok {log.Fatal("not found")}

// 删除
err = ndb.Set(ctx, "user01", nil)
if err != nil {log.Fatal(err)}

```

存储读取时，就涉及到 marshal 和 unmarshal 的过程，marshal 方法的选取顺序依次为：
```go
// 1. 看自身是否实现了 marshal 方法
type TMarshaler interface {
	Marshal() ([]byte, error)
	Unmarshal([]byte) error
}

// 2. 看是否在全局注册了 marshal 方法
func RegisterGMarshaler[T any](marshaler GMarshaler[T])
func RegisterMarshaler(t any, marshaler NMarshaler)

// 3. 看是否有 default 方法 (默认为 json marshaler)
SetDefaultMarshaler(NMarshaler)

// 其中，两类 marshaler 的定义如下:
type GMarshaler[T any] interface {
    Marshal(v *T) ([]byte, error)
    Unmarshal(data []byte, v *T) (err error)
}

type NMarshaler interface {
    Marshal(v any) ([]byte, error)
    Unmarshal(data []byte, v any) error
}

```
