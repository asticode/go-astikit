package astikit

type BitFlags uint64

func (fs BitFlags) Add(f uint64) uint64 { return uint64(fs) | f }

func (fs BitFlags) Del(f uint64) uint64 { return uint64(fs) &^ f }

func (fs BitFlags) Has(f uint64) bool { return uint64(fs)&f > 0 }
