package store

import (
	"testing"
	"github.com/nyaruka/junebug/store"
)

func TestInsertSorted(t *testing.T){
	pq := store.PriorityQueue{}

	for i:=0; i<200000; i++ {
		pq.Insert(uint64(i*2))
	}

	if pq.Len() != 200000 {
		t.Error("length should be 200,000")
	}

	for i:=0; i<200000; i++ {
		id := pq.Pop()
		if uint64(i*2) != id {
			t.Errorf("%d != %d", i, id)
		}
	}

	if pq.Len() != 0 {
		t.Error("length should be 0")
	}

	// alternative high and low priorities
	for i:=0; i<200000; i++ {
		if i%2 == 0 {
			pq.Insert(uint64(i))
		} else {
			pq.Insert(store.LOW_PRIORITY_MASK|uint64(i))
		}
	}

	if pq.Len() != 200000 {
		t.Error("Length should be 200,000")
	}

	// pop them back off
	for i:=0; i<100000; i++ {
		id := pq.Pop()
		if id != uint64(i*2) {
			t.Errorf("[%d] %d should be %d", i, id, i)
		}
	}

	for i:=1; i<100000; i+=2 {
		id := pq.Pop()
		test := store.LOW_PRIORITY_MASK|uint64(i)
		if id != test {
			t.Errorf("[%d] %d should be %d", i, id, test)
		}
	}
}