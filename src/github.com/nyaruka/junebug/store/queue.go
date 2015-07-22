package store

import (
	"sort"
)

// Optimized data structure for queues of Msg ids.
// This has some optimizations for the common case of appending ids greater than the current
// contents as well as the binary nature of our priorities.

type PriorityQueue struct {
	high []uint64
	low  []uint64
}

func (*PriorityQueue) insertInSlice(ids []uint64, id uint64) []uint64 {
    // if we are at capacity, double it
	if len(ids) == cap(ids) {
		new_ids := make([]uint64, len(ids), (cap(ids)+1) * 2)
		copy(new_ids, ids)
		ids = new_ids
	}

	// first check whether we might just belong at the end, this is the common case
	i := 0
	if len(ids) > 0 && ids[len(ids)-1] < id {
		i = len(ids)
	} else {
		// otherwise, do a more expensive binary search
		i = sort.Search(len(ids), func(i int) bool { return (ids)[i] >= id })
	}

	// special case inserting at the end, which is a simple append
	if i == len(ids) {
		ids = append(ids, id)
	} else {
		// we are inserting in the middle somewhere
		ids = append(ids, 0)
		copy(ids[i+1:], ids[i:])
		ids[i] = id
	}
	return ids
}

func (q *PriorityQueue) Insert(id uint64) {
	if id >= LOW_PRIORITY_MASK {
		q.low = q.insertInSlice(q.low, id)
	} else {
		q.high = q.insertInSlice(q.high, id)
	}
}

func (q *PriorityQueue) Len() int {
	return len(q.low) + len(q.high)
}

func (q *PriorityQueue) Pop() (id uint64) {
	if len(q.high) > 0 {
		id = q.high[0]
		q.high = q.high[1:]
	} else {
		id = q.low[0]
		q.low = q.low[1:]
	}

	return id
}
