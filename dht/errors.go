package dht

import "fmt"

type InvalidValue struct {
	Value string
}

func (iv *InvalidValue) Error() string {
	return fmt.Sprintf("Invalid value: %s", iv.Value)
}

type NoCapacity struct {
	Max int
}

func (nc *NoCapacity) Error() string {
	return fmt.Sprintf("Out of capacity, max: %d", nc.Max)
}
