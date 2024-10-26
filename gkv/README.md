## gkv

generic kv store, 意为 带泛型的 kv store，让 kv store 直接读取具体类型，而不是用 []byte 或 string 转换。

gkv 的底层存储基于 `kvstore.KVStorer` 接口，可以直接使用 `diskv.CreateDB` 创建一个符合要求的存储器。
同时，也可以使用 redis、bbolt、etcd、sqlite3 几种实现，详情可见 `kvstore` 目录。

### 快速上手

#### 创建一个用来存储自定义结构体的db

```go
type User struct {
    Name string
    Age  int
}

// 创建底层存储器
diskdb, err := diskv.CreateDB(ctx, &gkv.CreateConfig{Dir: "/tmp/gkv", KeysLen: 100, MaxLen:  64})
if err != nil {log.Fatal(err)}

// 创建 User{} 这个结构体的 db
db := gkv.NewT(User{}, diskdb)
// 你也可以使用 `diskv.New` 创建一个 db
// db := gkv.New[User](diskdb)

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

#### 一个 store 存储不同类型的数据

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

diskdb, err := diskv.CreateDB(ctx, &gkv.CreateConfig{Dir: "/tmp/gkv", KeysLen: 100, MaxLen:  64})
if err != nil {log.Fatal(err)}

// 创建一个存储器
ndb := gkv.NewNDiskv(diskdb) // 意为: normal diskv, 普通存储(不限定类型)

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
err := ndb.Set(ctx, "dog01", dog01)
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

### marshal 逻辑

存储读取时，就涉及到 marshal 和 unmarshal 的过程，marshal 方法的选取顺序依次为：

```go
// 1. 看是否在全局注册了 marshal 方法
func RegisterGMarshaler[T any](marshaler GMarshaler[T])
func RegisterMarshaler(t any, marshaler NMarshaler)

// 2. 看是否有 default 方法 (默认为 json marshaler)
SetDefaultMarshaler(NMarshaler)

// 其中，两类 marshaler 的定义如下:
type GMarshaler[T any] interface { // G 意味 Generic, 带泛型的，只能处理特定类型。
    Marshal(v *T) ([]byte, error)
    Unmarshal(data []byte, v *T) (err error)
}

type NMarshaler interface { // N 意为 Normal, 不确定类型，可以处理任何类型。
    Marshal(v any) ([]byte, error)
    Unmarshal(data []byte, v any) error
}

```

#### GMarshaler 的使用

GMarshaler 是一个泛型接口，可以用来定义确定类型的 marshal 和 unmarshal 逻辑。
比如，一个 User 结构体，我们希望存储时，只存储其 Name 和 Age 字段，而忽略其他字段 (有特定的 Marshal 逻辑)
```go

type User struct {
    Name string
    Age  int
    XX   string
}

type UserMarshaler struct {}
func (u UserMarshaler) Marshal(v *User) ([]byte, error) {
    // some logic here
    return []byte(""), nil
}
func (u UserMarshaler) Unmarshal(data []byte, v *User) (err error) {
    // some logic here
    return nil
}
func NewUserMarshaler() GMarshaler[User] {
    return &UserMarshaler{}
}

// 注册 User 类型的 marshaler
func init() {
    RegisterGMarshaler(NewUserMarshaler()) 
    // 上面这样用一个工厂方法返回一个带类型的 Marshaler，是由于 go 对于接口泛型的推断还不够好，需要主动指定类型
    // 也可以直接使用如下方式:
    RegisterGMarshaler[User](&UserMarshaler{})
}
```

如果一个 结构体实现了`自己类型`的 Marshal 和 Unmarshal 方法，注册方式如下:

```go
type MUser struct {
    Name string
    Age  int
}

func (u *MUser) Marshal(v *MUser) ([]byte, error) {
	// some logic here
	return []byte(""), nil
}

func (u *MUser) Unmarshal(data []byte, v *MUser) (err error) {
    // some logic here
	return nil
}

func init() {
    RegisterGMarshaler[MUser](&MUser{})
}
```

当前没有考虑结构体直接实现`对自身`的 Marshaler 接口的逻辑，这一定程度上增加了复杂度，用一套逻辑更好。
> 比如 `func (u *MUser) Marshal() ([]byte, error)` 直接返回本身的 Marshal 结果

#### NMarshaler 的使用

NMarshaler 是一个通用的接口，可以用来定义不确定类型的 marshal 和 unmarshal 逻辑。
实现时，接口接收的类型是 any，这要求实现者清楚自己要存储的类型如何处理，并且可能处理多种类型的数据。

在传入数据时，gkv 会自动把数据类型转换成对应的指针，比如注册的类型是一个 `User{}`，那么在 Marshal 和 Unmarshal 时的 v 的底层类型就是 `*User{}` (当然，表现上是 `any`)。

```go
type User struct {}
type UserMarshaler struct {}
func (u UserMarshaler) Marshal(v any) ([]byte, error) {
    // some logic here
    return []byte(""), nil
}
func (u UserMarshaler) Unmarshal(data []byte, v any) (err error) {
    // some logic here
    return nil
}

func init() {
    RegisterMarshaler(User{}, &UserMarshaler{}) // 注意，这里的参数是 User{}，而不是 *User{}
}
```

#### 默认的 marshaler

gkv 默认使用 json marshaler，但可以通过 SetDefaultMarshaler(NMarshaler) 来设置。

```go
type MyMarshaler struct {}
func (m MyMarshaler) Marshal(v any) ([]byte, error) {
    // some logic here
    return []byte(""), nil
}
func (m MyMarshaler) Unmarshal(data []byte, v any) (err error) {
    // some logic here
    return nil
}

func init() {
    SetDefaultMarshaler(MyMarshaler{})
}
```

### 使用其他的底层存储

gkv 并不要求一定使用 diskv 作为底层存储，而是支持使用任何实现了 `diskv.KVStore` 接口的存储系统。
目前，已经封装了 `redis`、`sqlite3`、`etcd3` 和 `bbolt` 的存储。

例子可参考 [examples/session](../examples/session/README.md) 部分。
