package diskv

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"hash/fnv"
)

type Diskv struct {
	mu sync.RWMutex // migrate 和 数据操作 不能同时进行

	dir string

	idxFile string
	idx     *idx

	dbFile  string
	dbstore *dbsotre
}

var DefaultCreateConfig = CreateConfig{
	Dir: ".",

	// KeySize:      15,
	// ValueLenSize: 6,
	// OffsetSize:   9,
	MaxLen:  32,
	KeysLen: 10000,
}

func init() {
	root, err := os.UserHomeDir()
	if err != nil {
		root = "."
	}

	root, err = filepath.Abs(root)
	if err != nil {
		panic(fmt.Errorf("get dir . abs path failed: %s", err))
	}

	DefaultCreateConfig.Dir = root
}

type CreateConfig struct {
	Dir string

	// KeySize      int // key 的长度 (key 的长度为多少) (采用固定长度 key, 前面用 0 补齐)
	// ValueLenSize int // value 长度的长度 (用多长的数字表示 value 长度)
	// OffsetSize   int // value 偏移量的长度 (用多长的数字表示 value 的偏移量)
	MaxLen  int // block 的最大长度 (key + valueLen + offset 共用)
	KeysLen int // 预分配多少 key 的空间
}

func OpenDB(ctx context.Context, dir string) (*Diskv, error) {
	d := &Diskv{
		dir: dir,
	}

	return d, d.openDB(ctx, dir)
}

func (d *Diskv) idxFileName(dir string) string {
	return filepath.Join(dir, "diskv.idx")
}

func (d *Diskv) dbFileName(dir string) string {
	return filepath.Join(dir, "diskv.db")
}

func (d *Diskv) openDB(ctx context.Context, dir string) error {
	idxFile := d.idxFileName(dir)
	idx, ok, err := d.getIdx(idxFile)
	if err != nil {
		return fmt.Errorf("open idx file error: %s", err)
	}
	if !ok {
		return fmt.Errorf("idx file not found")
	}
	d.idx = idx
	dbstore, err := d.getOrCreateDBStore(d.dbFileName(dir))
	if err != nil {
		return fmt.Errorf("open db file error: %s", err)
	}
	d.dbstore = dbstore
	return nil
}

func CreateDB(ctx context.Context, config *CreateConfig) (*Diskv, error) {
	if config == nil {
		config = &DefaultCreateConfig
	}

	d := &Diskv{}

	err := os.MkdirAll(config.Dir, 0777)
	if err != nil {
		return nil, fmt.Errorf("create dir error: %s", err)
	}

	idxFile := d.idxFileName(config.Dir)
	idx, err := d.createIdx(ctx, idxFile, *config)
	if err != nil {
		return nil, fmt.Errorf("create idx file error: %s", err)
	}

	dbFile := d.dbFileName(config.Dir)
	dbstore, err := d.getOrCreateDBStore(dbFile)
	if err != nil {
		return nil, fmt.Errorf("create db file error: %s", err)
	}

	d.dir = config.Dir
	d.dbFile = dbFile
	d.dbstore = dbstore

	d.idx = idx
	d.idxFile = idxFile

	return d, nil
}

func (d *Diskv) createIdx(ctx context.Context, idxFile string, config CreateConfig) (*idx, error) {
	f, err := os.OpenFile(idxFile, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		return nil, fmt.Errorf("create idx file error: %s", err)
	}

	idx := &idx{
		f:        f,
		filePath: idxFile,
	}

	err = idx.setIdxMeta(ctx, &idxMeta{
		// keySize:      config.KeySize,
		// valueLenSize: config.ValueLenSize,
		// offsetSize:   config.OffsetSize,
		maxLength: config.MaxLen,
		keysLen:   config.KeysLen,
	})
	if err != nil {
		return nil, fmt.Errorf("set idx meta error: %s", err)
	}

	return idx, nil
}

func (d *Diskv) getIdx(idxFile string) (*idx, bool, error) {
	f, err := os.OpenFile(idxFile, os.O_RDWR, 0666)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}

		return nil, false, err
	}

	idx := &idx{
		f:        f,
		filePath: idxFile,
	}

	return idx, true, nil
}

func (d *Diskv) getOrCreateDBStore(dbFile string) (*dbsotre, error) {
	f, err := os.OpenFile(dbFile, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}

	return &dbsotre{
		f:        f,
		filePath: dbFile,
	}, nil
}

type idx struct {
	meta *idxMeta

	filePath string // 索引文件地址
	mu       sync.Mutex
	f        *os.File
}

type idxMeta struct {
	// keySize      int // key 的长度 (key 的长度为多少) (采用固定长度 key, 前面用 0 补齐)
	// valueLenSize int // value 长度的长度 (用多长的数字表示 value 长度)
	// offsetSize   int // value 偏移量的长度 (用多长的数字表示 value 的偏移量)
	maxLength int

	keysLen int // 预分配的 key 的数量
}

type valueMeta struct {
	key string

	offset int
	length int
}

type valueItem struct {
	key   string
	value []byte
}

func (idx *idx) runWithFile(ctx context.Context, rf func(ctx context.Context, f *os.File) error) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	if idx.f == nil {
		f, err := os.OpenFile(idx.filePath, os.O_CREATE|os.O_RDWR, 0666)
		if err != nil {
			return fmt.Errorf("open idx file error: %s", err)
		}
		idx.f = f
	}

	return rf(ctx, idx.f)
}

func (idx *idx) setIdxMeta(ctx context.Context, meta *idxMeta) (err error) {
	metaBytes := formatIdxMeta(meta)

	if len(metaBytes) != dbMetaLen {
		return fmt.Errorf("write idx file error of unexpected length: %d", len(metaBytes))
	}

	err = idx.runWithFile(ctx, func(ctx context.Context, f *os.File) error {
		_, err := f.WriteAt(metaBytes, 0)
		if err != nil {
			return fmt.Errorf("write idx file error: %s", err)
		}

		return nil
	})
	if err != nil {
		return err
	}

	idx.meta = meta

	return nil
}

func (idx *idx) getIdxMeta(ctx context.Context) (*idxMeta, error) {
	if idx.meta != nil {
		return idx.meta, nil
	}
	data := make([]byte, dbMetaLen)

	err := idx.runWithFile(ctx, func(ctx context.Context, f *os.File) error {
		n, err := f.ReadAt(data, 0)
		if err != nil {
			return fmt.Errorf("read idx file error: %s", err)
		}

		if n != dbMetaLen {
			return fmt.Errorf("read idx file error of unexpected length: %d", n)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	idxMeta, err := parseIdxMeta(data)
	if err != nil {
		return nil, fmt.Errorf("parse idx meta error: %s", err)
	}

	idx.meta = idxMeta

	return idx.meta, nil
}

func (m *idxMeta) getKeyBlockLength() int {
	// 2 个字节是分隔符, eg: 0000000longtest,000000,000000000
	// return m.keySize + 1 + m.valueLenSize + 1 + m.offsetSize
	return m.maxLength
}

func (m *idxMeta) getBlockStartOffset(slot int) int {
	return dbMetaLen + m.getKeyBlockLength()*slot
}

const dbMetaLen = 64

// // kv db meta seems like: [keysize:000015,lensize:000006,offsetsize:000010,keyslen:001000]
// // value meta eg: 0000000longtest,000000,000000000
// // 为了做对齐，最好要能被 8 整除，例如 32 byte,64 byte 等等，上述配置基本是最小配置了，15个字符的key长度，6个字符的value长度 (单个 value 最大能到 0.95MB)，9个字符的value偏移量(单个文件最大到 0.93GB)
func formatIdxMeta(meta *idxMeta) []byte {
	// idxStr := fmt.Sprintf("[keysize:%06d,lensize:%06d,offsetsize:%06d,keyslen:%06d]", meta.keySize, meta.valueLenSize, meta.offsetSize, meta.keysLen)
	idxStr := fmt.Sprintf("[maxlength:%06d,keyslen:%06d,x:%028d]", meta.maxLength, meta.keysLen, 0)
	return []byte(idxStr)
}

func parseIdxMeta(data []byte) (*idxMeta, error) {
	idxMeta := &idxMeta{}
	idxMetaStr := string(data)
	if idxMetaStr[0] != '[' || idxMetaStr[len(idxMetaStr)-1] != ']' {
		return nil, errors.New("idx meta format error")
	}
	var err error
	idxMetaStr = idxMetaStr[1 : len(idxMetaStr)-1]
	for _, item := range strings.Split(idxMetaStr, ",") {
		kv := strings.Split(item, ":")
		if len(kv) != 2 {
			return nil, errors.New("idx meta format error")
		}

		switch kv[0] {
		case "maxlength":
			_, err = fmt.Sscanf(kv[1], "%d", &idxMeta.maxLength)
			if err != nil {
				return nil, fmt.Errorf("idx meta format error: %s", err)
			}
		// case "lensize":
		// 	_, err = fmt.Sscanf(kv[1], "%d", &idxMeta.valueLenSize)
		// 	if err != nil {
		// 		return nil, fmt.Errorf("idx meta format error: %s", err)
		// 	}
		// case "offsetsize":
		// 	_, err = fmt.Sscanf(kv[1], "%d", &idxMeta.offsetSize)
		// 	if err != nil {
		// 		return nil, fmt.Errorf("idx meta format error: %s", err)
		// 	}
		case "keyslen":
			_, err = fmt.Sscanf(kv[1], "%d", &idxMeta.keysLen)
			if err != nil {
				return nil, fmt.Errorf("idx meta format error: %s", err)
			}
		default:

		}
	}

	return idxMeta, nil
}

// value meta eg: 0000000longtest,000000,000000000
func formatValueMeta(meta *valueMeta) []byte {
	return []byte(fmt.Sprintf("%s,%d,%d", meta.key, meta.length, meta.offset))
}

func isEmpty(data []byte) bool { // 认为第一个字符为 \x0 时就是空的
	if len(data) == 0 {
		return true
	}

	if data[0] == 0 {
		return true
	}

	return false
}
func parseValueMeta(data []byte) (meta *valueMeta, ok bool, err error) {
	if isEmpty(data) { // 空数据
		return nil, false, nil
	}

	meta = &valueMeta{}
	dataStr := string(data)

	dataStrs := strings.Split(dataStr, ",")

	if len(dataStrs) != 3 {
		return nil, false, fmt.Errorf("parse value meta error: %s", dataStr)
	}

	key := dataStrs[0]

	_, err = fmt.Sscanf(dataStrs[1], "%d", &meta.length)
	if err != nil {
		return nil, false, fmt.Errorf("parse value meta error: %s", err)
	}
	_, err = fmt.Sscanf(dataStrs[2], "%d", &meta.offset)
	if err != nil {
		return nil, false, fmt.Errorf("parse value meta error: %s", err)
	}

	meta.key = key

	return meta, true, nil
}

func (idx *idx) delValueMeta(ctx context.Context, key string) (has bool, err error) {
	slot, err := idx.hashKey(ctx, key)
	if err != nil {
		return false, fmt.Errorf("hash key error: %s", err)
	}

	idxMeta, err := idx.getIdxMeta(ctx)
	if err != nil {
		return false, err
	}

	for {
		v, ok, err := idx.getValueOfSlot(ctx, slot)
		if err != nil {
			return false, err
		}

		if !ok {
			return false, nil
		}

		if v.key != key {
			slot++
			continue
		}

		size := idxMeta.getKeyBlockLength()
		data := make([]byte, size)

		return true, idx.runWithFile(ctx, func(ctx context.Context, f *os.File) error {
			offset := int64(idx.meta.getBlockStartOffset(slot))

			_, err := f.WriteAt(data, offset)
			if err != nil {
				return fmt.Errorf("write idx file in del error: %s", err)
			}

			return nil
		})
	}
}

func (idx *idx) setValueMeta(ctx context.Context, valueMeta *valueMeta) error {
	// idxMeta, err := idx.getIdxMeta(ctx)
	// if err != nil {
	// 	return err
	// }

	slot, err := idx.hashKey(ctx, valueMeta.key)
	if err != nil {
		return err
	}

	for {
		slotmeta, ok, err := idx.getValueOfSlot(ctx, slot)
		if err != nil {
			return err
		}
		if ok && slotmeta.key != valueMeta.key { // 位置被占了，往下一个
			slot++
			continue
		}

		offset := int64(idx.meta.getBlockStartOffset(slot))

		valueMetaData := formatValueMeta(valueMeta)
		if len(valueMetaData) > idx.meta.getKeyBlockLength() {
			return fmt.Errorf("value too long, max is %d", idx.meta.getKeyBlockLength())
		}

		return idx.runWithFile(ctx, func(ctx context.Context, f *os.File) error {
			_, err := f.WriteAt(valueMetaData, offset)
			if err != nil {
				return fmt.Errorf("write idx file error: %s", err)
			}
			return nil
		})
	}

}

func (idx *idx) getValueMeta(ctx context.Context, key string) (*valueMeta, bool, error) {
	slot, err := idx.hashKey(ctx, key)
	if err != nil {
		return nil, false, err
	}

	return idx.getValueOfKeyFromSlot(ctx, key, slot)
}

func (idx *idx) getValueOfKeyFromSlot(ctx context.Context, key string, slot int) (*valueMeta, bool, error) {
	// idxMeta, err := idx.getIdxMeta(ctx)
	// if err != nil {
	// 	return nil, err
	// }

	for {
		valueMeta, ok, err := idx.getValueOfSlot(ctx, slot)
		if err != nil {
			return nil, false, err
		}

		if !ok { // 空数据
			return nil, false, nil
		}

		if valueMeta.key == key { // key 相同才是命中，否则顺延 slot
			return valueMeta, true, nil
		}

		slot++
		// if slot >= idxMeta.keysLen { // 暂时不设置最大 slot 限制，允许退化成遍历
		// 	return nil, ErrKeyNotFound
		// }
	}
}

func (idx *idx) getValueOfSlot(ctx context.Context, slot int) (valueMeta *valueMeta, ok bool, err error) {
	idxMeta, err := idx.getIdxMeta(ctx)
	if err != nil {
		return nil, false, err
	}

	startOffset := idxMeta.getBlockStartOffset(slot)
	blockLen := idxMeta.getKeyBlockLength()

	data := make([]byte, blockLen)
	err = idx.runWithFile(ctx, func(ctx context.Context, f *os.File) error {
		n, err := f.ReadAt(data, int64(startOffset))
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return fmt.Errorf("read idx file error: %s", err)
		}

		if n != blockLen {
			return fmt.Errorf("read idx file error of unexpected length: %d", n)
		}
		return nil
	})
	if err != nil {
		return nil, false, err
	}

	valueMeta, ok, err = parseValueMeta(data)
	if err != nil {
		return nil, false, fmt.Errorf("parse value meta error: %s", err)
	}

	return valueMeta, ok, nil
}

func (idx *idx) hashKey(ctx context.Context, key string) (int, error) {
	meta, err := idx.getIdxMeta(ctx)
	if err != nil {
		return 0, err
	}

	keysLen := meta.keysLen
	// 计算 key 的哈希值，使用 fnv1a 算法
	h := fnv.New32a()
	h.Write([]byte(key))
	hashValue := int(h.Sum32())

	hashValue = hashValue % keysLen

	return hashValue, nil
}

func (d *Diskv) forEachKey(ctx context.Context, f func(ctx context.Context, valMeta *valueMeta) (ok bool)) error {
	idxMeta, err := d.idx.getIdxMeta(ctx)
	if err != nil {
		return err
	}

	slot := 0
	maxSlots := idxMeta.keysLen

	for {
		valMeta, ok, err := d.idx.getValueOfSlot(ctx, slot)
		if err != nil {
			return err
		}

		if !ok {
			if slot >= maxSlots {
				return nil // 结束
			}

			slot++
			continue
		}

		if !f(ctx, valMeta) { // 用户主动退出
			return nil
		}
		slot++
	}
}

func (d *Diskv) ForEach(ctx context.Context, f func(ctx context.Context, key string, value []byte) (ok bool)) error {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return d.forEach(ctx, f)
}

func (d *Diskv) forEach(ctx context.Context, f func(ctx context.Context, key string, value []byte) (ok bool)) error {
	var err error
	d.forEachKey(ctx, func(ctx context.Context, valMeta *valueMeta) bool {
		var val *valueItem
		val, err = d.dbstore.read(ctx, valMeta)
		if err != nil {
			return false
		}

		if !f(ctx, valMeta.key, val.value) { // 用户主动退出
			return false
		}

		return true
	})
	return err
}

func (d *Diskv) Get(ctx context.Context, key string) (data []byte, ok bool, err error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	meta, ok, err := d.idx.getValueMeta(ctx, key)
	if err != nil {
		return nil, false, err
	}

	if !ok {
		return nil, false, nil
	}

	val, err := d.dbstore.read(ctx, meta)
	if err != nil {
		return nil, false, err
	}

	return val.value, true, nil
}

func (d *Diskv) GetString(ctx context.Context, key string) (data string, ok bool, err error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	val, ok, err := d.Get(ctx, key)
	if err != nil {
		return "", false, err
	}

	return string(val), ok, nil
}

func (d *Diskv) Set(ctx context.Context, key string, val []byte) error {
	d.mu.RLock()
	defer d.mu.RUnlock()

	valMeta, err := d.dbstore.write(ctx, &valueItem{key: key, value: val})
	if err != nil {
		return err
	}

	return d.idx.setValueMeta(ctx, valMeta)
}

func (d *Diskv) SetString(ctx context.Context, key string, val string) error {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return d.Set(ctx, key, []byte(val))
}

func (d *Diskv) Has(ctx context.Context, key string) (has bool, err error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	_, has, err = d.idx.getValueMeta(ctx, key)
	return has, err
}

// ok = true => 值存在并已删除
// ok = false => 值不存在
func (d *Diskv) Del(ctx context.Context, key string) (ok bool, err error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// db file 记录删除
	err = d.dbstore.del(ctx, key)
	if err != nil {
		return false, err
	}

	return d.idx.delValueMeta(ctx, key) // 只删索引，不删值
}

type dbsotre struct {
	filePath string
	mu       sync.Mutex
	f        *os.File
}

func (d *dbsotre) runWithFile(ctx context.Context, rf func(ctx context.Context, f *os.File) error) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.f == nil {
		f, err := os.OpenFile(d.filePath, os.O_RDONLY|os.O_APPEND, 0666)
		if err != nil {
			return err
		}
		d.f = f
	}

	return rf(ctx, d.f)
}

func (d *dbsotre) read(ctx context.Context, m *valueMeta) (*valueItem, error) {
	data := make([]byte, m.length)

	err := d.runWithFile(ctx, func(ctx context.Context, f *os.File) error {
		n, err := f.ReadAt(data, int64(m.offset))
		if err != nil {
			return err
		}

		if n != m.length {
			return errors.New("read data error, value length not match")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	_, val, err := decodeValue(m.key, data)
	return val, err
}

const (
	opSet = "_set"
	opDel = "_del"

	splitOp = '\n'
)

func (d *dbsotre) del(ctx context.Context, key string) error {
	return d.runWithFile(ctx, func(ctx context.Context, f *os.File) error {
		_, err := f.Write(encodeValueItem(opDel, &valueItem{key: key}))
		return err
	})
}

func (d *dbsotre) write(ctx context.Context, valueItem *valueItem) (*valueMeta, error) {
	meta := &valueMeta{
		key: valueItem.key,
	}

	err := d.runWithFile(ctx, func(ctx context.Context, f *os.File) error {
		fileInfo, err := f.Stat()
		if err != nil {
			return err
		}

		meta.offset = int(fileInfo.Size())

		val := encodeValueItem(opSet, valueItem)
		meta.length = len(val)

		_, err = f.Write(val)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return meta, nil
}

func decodeValue(key string, data []byte) (op string, val *valueItem, err error) {

	val = &valueItem{}
	if len(data) < 5 {
		return "", nil, errors.New("read data error, data length not match")
	}
	vals := bytes.SplitN(data, []byte{'['}, 2)
	if len(vals) != 2 {
		return "", nil, errors.New("read data error, split length not match")
	}

	op = string(vals[0])

	kvdata := vals[1][0 : len(vals[1])-1] // 去除结尾的 splitOp

	dataVals := bytes.SplitN(kvdata, []byte("]"), 2)
	if len(dataVals) < 1 {
		return "", nil, errors.New("read data error, split length not match")
	}

	val.key = string(dataVals[0])
	if val.key != key {
		return "", nil, errors.New("read data error, key not match")
	}

	if len(dataVals) == 2 {
		val.value = dataVals[1]
	}

	return op, val, nil
}

func encodeValueItem(op string, val *valueItem) []byte {
	res := append([]byte(op+"["+val.key+"]"), val.value...)
	return append(res, splitOp)
}
