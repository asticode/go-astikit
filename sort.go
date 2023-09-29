package astikit

import "sort"

// SortInt64 sorts a slice of int64s in increasing order.
func SortInt64(a []int64) { sort.Sort(SortInt64Slice(a)) }

// SortInt64Slice attaches the methods of Interface to []int64, sorting in increasing order.
type SortInt64Slice []int64

func (p SortInt64Slice) Len() int           { return len(p) }
func (p SortInt64Slice) Less(i, j int) bool { return p[i] < p[j] }
func (p SortInt64Slice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

// SortUint64 sorts a slice of uint64s in increasing order.
func SortUint64(a []uint64) { sort.Sort(SortUint64Slice(a)) }

// SortUint64Slice attaches the methods of Interface to []uint64, sorting in increasing order.
type SortUint64Slice []uint64

func (p SortUint64Slice) Len() int           { return len(p) }
func (p SortUint64Slice) Less(i, j int) bool { return p[i] < p[j] }
func (p SortUint64Slice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
