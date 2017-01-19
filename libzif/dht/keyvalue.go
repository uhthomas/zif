package dht

const (
	MaxValueSize = 10 * 1024
)

type KeyValue struct {
	key   Address
	value []byte // Max size of 64kbs

	distance Address
}

func (kv KeyValue) Key() *Address {
	return &kv.key
}

func (kv KeyValue) Value() []byte {
	return kv.value
}

func NewKeyValue(key Address, value []byte) *KeyValue {
	ret := &KeyValue{}

	ret.key = key
	ret.value = make([]byte, len(value))
	copy(ret.value, value)

	return ret
}

func (kv *KeyValue) Valid() bool {
	return len(kv.value) <= MaxValueSize && len(kv.key.Raw) == AddressBinarySize
}

type Pairs []*KeyValue

func (e Pairs) Len() int {
	return len(e)
}

func (e Pairs) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}

func (e Pairs) Less(i, j int) bool {
	return e[i].distance.Less(&e[j].distance)
}
