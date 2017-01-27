package dht

import msgpack "gopkg.in/vmihailenco/msgpack.v2"

const (
	MaxValueSize = 10 * 1024
)

type KeyValue struct {
	Key_   Address `json:"key"`
	Value_ []byte  `json:"value"`

	distance Address
}

func (kv KeyValue) Key() *Address {
	return &kv.Key_
}

func (kv KeyValue) Value() []byte {
	return kv.Value_
}

func (kv KeyValue) Decode(iface interface{}) error {
	return msgpack.Unmarshal(kv.Value(), iface)
}

func NewKeyValue(key Address, value []byte) *KeyValue {
	ret := &KeyValue{}

	ret.Key_ = key
	ret.Value_ = make([]byte, len(value))
	copy(ret.Value_, value)

	return ret
}

func (kv *KeyValue) Valid() bool {
	return len(kv.Value_) <= MaxValueSize && len(kv.Key_.Raw) == AddressBinarySize
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
