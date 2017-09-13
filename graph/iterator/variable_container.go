// Copyright 2014 The Cayley Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package iterator

import (
	"fmt"
	"sort"

	"github.com/codelingo/cayley/graph"
)

// A VariableContainer
type VariableContainer struct {
	subIt  graph.Iterator
	uid    uint64
	tags   graph.Tagger
	qs     graph.QuadStore
	itType varItType
	name   string
}

func NewVariableContainer(qs graph.QuadStore, variableIt graph.Iterator, varName string) *VariableContainer {
	if variableIt.Type().String() != "variable" {
		panic("Developer Error: variable containers must only contain variable iterators.")
	}

	return &VariableContainer{
		uid:    NextUID(),
		subIt:  variableIt,
		name:   varName,
		qs:     qs,
		itType: undetermined,
	}
}

func (it *VariableContainer) UID() uint64 {
	return it.uid
}

// TODO(BlakeMScurr) Allow resetting on the iteration context.
func (it *VariableContainer) Reset() {
	it.subIt.Reset()
}

func (it *VariableContainer) Close() error {
	return it.subIt.Close()
}

func (it *VariableContainer) Tagger() *graph.Tagger {
	return &it.tags
}

func (it *VariableContainer) TagResults(dst map[string]graph.Value) {
	for _, tag := range it.tags.Tags() {
		dst[tag] = it.Result()
	}

	for tag, value := range it.tags.Fixed() {
		dst[tag] = value
	}
}

func (it *VariableContainer) Clone() graph.Iterator {
	out := NewVariableContainer(it.qs, it.subIt.Clone(), it.name)
	out.tags.CopyFrom(it)
	return out
}

func (it *VariableContainer) Describe() graph.Description {
	fixed := make([]string, 0, len(it.tags.Fixed()))
	for k := range it.tags.Fixed() {
		fixed = append(fixed, k)
	}
	s, _ := it.Size()
	sort.Strings(fixed)
	return graph.Description{
		UID:  it.UID(),
		Type: it.Type(),
		Tags: fixed,
		Size: s,
		Name: fmt.Sprintf("Container for %s for \"%s\" variable.", string(it.itType), it.name),
	}
}

// Register this iterator as a VariableContainer iterator.
func (it *VariableContainer) Type() graph.Type { return graph.VariableContainer }

// Contains checks if the passed value is equal to the current value of the variableContainer.
// Contains is not defined for a bind variableContainer.
func (it *VariableContainer) Contains(ctx *graph.IterationContext, v graph.Value) bool {
	graph.ContainsLogIn(it, v)

	if it.itType == undetermined {
		if !ctx.IsBound(it.name) {
			it.ToBinder()
		} else {
			it.itType = user
		}
	}

	res := it.subIt.Contains(ctx, v)
	if res {
		fmt.Println("Got out of contains.")
	}
	return res
}

// Next advances the value of the variableContainer on the iteration context.
func (it *VariableContainer) Next(ctx *graph.IterationContext) bool {
	graph.NextLogIn(it)

	if it.itType == undetermined {
		if !ctx.IsBound(it.name) {
			it.ToBinder()
		} else {
			it.itType = user
		}
	}

	res := it.subIt.Next(ctx)
	if res {
		fmt.Println("Got out of next.")
	}
	return res
}

func (it *VariableContainer) Err() error {
	return it.subIt.Err()
}

func (it *VariableContainer) Result() graph.Value {
	return it.subIt.Result()
}

func (it *VariableContainer) NextPath(ctx *graph.IterationContext) bool {
	return it.subIt.NextPath(ctx)
}

// No sub-iterators.
func (it *VariableContainer) SubIterators() []graph.Iterator {
	return []graph.Iterator{it.subIt}
}

// There is no (apparent) optimization for a variableContainer iterator, because most of its information is stored
// in the iteration context, which doesn't exists until iteration.
func (it *VariableContainer) Optimize() (graph.Iterator, bool) {
	return it.subIt.Optimize()
}

// Size is unclear because it is largely store on the iteratorContext.
func (it *VariableContainer) Size() (int64, bool) {
	return it.subIt.Size()
}

func (it *VariableContainer) Stats() graph.IteratorStats {
	s, exact := it.Size()
	return graph.IteratorStats{
		ContainsCost: s,
		NextCost:     s,
		Size:         s,
		ExactSize:    exact,
	}
}

func (it *VariableContainer) ToBinder() {
	it.subIt = NewRecursive(it.qs, it.subIt, func(st graph.QuadStore, iter graph.Iterator) graph.Iterator {
		return iter
	}, 1)
	it.itType = binder
}

func (it *VariableContainer) ToUser() {
	it.subIt = NewAnd(it.qs, it.qs.NodesAllIterator(), it.subIt)
	it.itType = user
}
