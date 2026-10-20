package entities

import "fmt"

type EntityID uint64

func makeEntityID(index, generation uint32) EntityID {
	return EntityID(uint64(index)<<32 | uint64(generation))
}

func (id EntityID) Index() uint32 {
	return uint32(uint64(id) >> 32)
}

func (id EntityID) Generation() uint32 {
	return uint32(id)
}

func (id EntityID) Valid() bool {
	return id != 0
}

func (id EntityID) String() string {
	if !id.Valid() {
		return "EntityID(invalid)"
	}
	return fmt.Sprintf("EntityID(%d:%d)", id.Index(), id.Generation())
}
