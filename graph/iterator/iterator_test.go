package iterator_test

import (
	"github.com/codelingo/cayley/graph"
	. "github.com/codelingo/cayley/graph/iterator"
)

// A testing iterator that returns the given values for Next() and Err().
type testIterator struct {
	*Fixed

	NextVal bool
	ErrVal  error
}

func newTestIterator(next bool, err error) graph.Iterator {
	return &testIterator{
		Fixed:   NewFixed(Identity),
		NextVal: next,
		ErrVal:  err,
	}
}

func (it *testIterator) Next(ctx *graph.IterationContext) bool {
	return it.NextVal
}

func (it *testIterator) Err() error {
	return it.ErrVal
}
