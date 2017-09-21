package iterator

import (
	"fmt"
	"math"
	"reflect"

	"github.com/codelingo/cayley/graph"
	"github.com/codelingo/cayley/quad"
	"github.com/waigani/xxx"
)

// Recursive iterator takes a base iterator and a morphism to be applied recursively, for each result.
type Recursive struct {
	uid      uint64
	tags     graph.Tagger
	subIt    graph.Iterator
	result   seenAt
	runstats graph.IteratorStats
	err      error

	qs            graph.QuadStore
	morphism      graph.ApplyMorphism
	seen          map[interface{}]seenAt
	currentIt     int
	nextIts       []itGivenVariables
	depth         int
	pathMap       map[interface{}][]map[string]graph.Value
	pathIndex     int
	containsValue graph.Value
	depthTags     graph.Tagger
	// Stores values at the current depth with the values of variables when
	// the value was found.
	depthCache []*valuesGivenVariables
	baseIt     graph.FixedIterator
	vars       map[string]graph.Value
	minDepth   int
	maxDepth   int
}

type itGivenVariables struct {
	varVals map[string]graph.Value
	it      graph.Iterator
}

type valuesGivenVariables struct {
	vars map[string]graph.Value
	vals []graph.Value
}

type seenAt struct {
	depth int
	val   graph.Value
}

var _ graph.Iterator = &Recursive{}

var MaxRecursiveSteps = 50

func NewRecursive(qs graph.QuadStore, it graph.Iterator, morphism graph.ApplyMorphism, max int) *Recursive {
	return &Recursive{
		uid:   NextUID(),
		subIt: it,

		qs:       qs,
		morphism: morphism,
		seen:     make(map[interface{}]seenAt),
		nextIts: []itGivenVariables{
			itGivenVariables{
				it:      &Null{},
				varVals: map[string]graph.Value{},
			},
		},
		depthCache:    []*valuesGivenVariables{},
		baseIt:        qs.FixedIterator(),
		pathMap:       make(map[interface{}][]map[string]graph.Value),
		containsValue: nil,
		maxDepth:      max,
	}
}

func (it *Recursive) UID() uint64 {
	return it.uid
}

func (it *Recursive) Reset() {
	it.result.val = nil
	it.result.depth = 0
	it.err = nil
	it.subIt.Reset()
	it.seen = make(map[interface{}]seenAt)
	it.pathMap = make(map[interface{}][]map[string]graph.Value)
	it.containsValue = nil
	it.pathIndex = 0
	it.nextIts = []itGivenVariables{
		itGivenVariables{
			it:      &Null{},
			varVals: map[string]graph.Value{},
		},
	}

	it.baseIt = it.qs.FixedIterator()
	it.depth = 0
	it.depthCache = []*valuesGivenVariables{}
	it.currentIt = 0
}

func (it *Recursive) Tagger() *graph.Tagger {
	return &it.tags
}

func (it *Recursive) AddDepthTag(s string) {
	it.depthTags.Add(s)
}

func (it *Recursive) TagResults(dst map[string]graph.Value) {
	for _, tag := range it.tags.Tags() {
		dst[tag] = it.Result()
	}

	for _, tag := range it.depthTags.Tags() {
		dst[tag] = it.qs.ValueOf(quad.Int(it.result.depth))
	}

	for tag, value := range it.tags.Fixed() {
		dst[tag] = value
	}

	for tag, value := range it.depthTags.Fixed() {
		dst[tag] = value
	}
	if it.containsValue != nil {
		paths := it.pathMap[graph.ToKey(it.containsValue)]
		if len(paths) != 0 {
			for k, v := range paths[it.pathIndex] {
				dst[k] = v
			}
		}
	}

	if len(it.nextIts) != 0 {
		for _, internalIt := range it.nextIts {
			internalIt.it.TagResults(dst)
		}
	}
	delete(dst, "__base_recursive")
}

func (it *Recursive) Clone() graph.Iterator {
	n := NewRecursive(it.qs, it.subIt.Clone(), it.morphism, it.maxDepth)
	n.tags.CopyFrom(it)
	n.depthTags.CopyFromTagger(&it.depthTags)
	return n
}

func (it *Recursive) SubIterators() []graph.Iterator {
	return []graph.Iterator{it.subIt}
}

func (it *Recursive) Next(ctx *graph.IterationContext) bool {
	it.pathIndex = 0
	if it.depth == 0 {
		it.depthCache = append(it.depthCache, &valuesGivenVariables{
			vars: ctx.Values(),
		})
		for it.subIt.Next(ctx) {
			res := it.subIt.Result()
			variableValues := ctx.Values()
			if len(variableValues) == 1 {
				fmt.Println("Nil stuff")
			}
			if it.VarsUpdated(ctx) {
				last := len(it.depthCache) - 1
				if len(it.depthCache[last].vals) == 0 {
					it.depthCache = it.depthCache[:last]
				}
				it.depthCache = append(it.depthCache, &valuesGivenVariables{
					vars: variableValues,
				})
			}

			last := len(it.depthCache) - 1
			it.depthCache[last].vals = append(it.depthCache[last].vals, it.subIt.Result())
			tags := make(map[string]graph.Value)
			it.subIt.TagResults(tags)
			key := graph.ToKey(res)
			it.pathMap[key] = append(it.pathMap[key], tags)
			for it.subIt.NextPath(ctx) {
				tags := make(map[string]graph.Value)
				it.subIt.TagResults(tags)
				it.pathMap[key] = append(it.pathMap[key], tags)
			}
		}
	}

	for {
		ok := it.nextIts[it.currentIt].it.Next(ctx)
		if !ok {
			if it.currentIt < len(it.nextIts)-1 {
				it.currentIt++
				continue
			}

			it.depth++
			if it.depth > it.maxDepth {
				return graph.NextLogOut(it, false)
			}

			if len(it.depthCache) == 0 {
				return graph.NextLogOut(it, false)
			}

			newDepthCache := []*valuesGivenVariables{}
			it.nextIts = []itGivenVariables{}
			for _, cacheSlot := range it.depthCache {
				if len(cacheSlot.vals) == 0 {
					continue
				}

				it.baseIt = it.qs.FixedIterator()
				for _, x := range cacheSlot.vals {
					it.baseIt.Add(x)
				}

				it.baseIt.Tagger().Add("__base_recursive")
				it.nextIts = append(it.nextIts, itGivenVariables{
					it:      it.morphism(it.qs, it.baseIt),
					varVals: cacheSlot.vars,
				})

				newDepthCache = append(newDepthCache, &valuesGivenVariables{
					vars: cacheSlot.vars,
				})
			}

			if len(it.nextIts) == 0 {
				it.nextIts = append(it.nextIts, itGivenVariables{
					it: &Null{},
				})
			}

			it.currentIt = 0
			it.depthCache = newDepthCache
			continue
		}

		val := it.nextIts[it.currentIt].it.Result()
		results := make(map[string]graph.Value)
		it.nextIts[it.currentIt].it.TagResults(results)
		key := graph.ToKey(val)
		if _, ok := it.seen[key]; ok {
			continue
		}
		it.seen[key] = seenAt{
			val:   results["__base_recursive"],
			depth: it.depth,
		}
		it.result.depth = it.depth
		it.result.val = val
		it.containsValue = it.getBaseValue(val)
		it.depthCache[it.currentIt].vals = append(it.depthCache[it.currentIt].vals, val)

		break
	}
	vals := it.nextIts[it.currentIt].varVals
	xxx.Dump(vals)
	var str string
	for _, v := range vals {
		str += it.qs.NameOf(v).String()
	}
	if str == "" {
		fmt.Println("Here we have set the value to zero.")
	}
	fmt.Printf("Which has value: %s\n", str)
	ctx.SetValues(vals)
	it.vars = vals
	return graph.NextLogOut(it, true)
}

func (it *Recursive) Err() error {
	return it.err
}

func (it *Recursive) Result() graph.Value {
	return it.result.val
}

func (it *Recursive) getBaseValue(val graph.Value) graph.Value {
	var at seenAt
	var ok bool
	if at, ok = it.seen[graph.ToKey(val)]; !ok {
		panic("trying to getBaseValue of something unseen")
	}
	for at.depth != 1 {
		if at.depth == 0 {
			panic("seen chain is broken")
		}
		at = it.seen[graph.ToKey(at.val)]
	}
	return at.val
}

func (it *Recursive) VarsUpdated(ctx *graph.IterationContext) bool {
	if ctx != nil {
		newVars := ctx.Values()
		// Using reflect is not ideal, and we should also not be throwing all this
		// information away, it could be useful if we have the same var value at a
		// later point.
		if !reflect.DeepEqual(newVars, it.vars) {
			fmt.Println("We had:")
			for n, v := range it.vars {
				fmt.Println(n + ": " + it.qs.NameOf(v).String())
			}
			fmt.Println("We have: ")
			for n, v := range newVars {
				fmt.Println(n + ": " + it.qs.NameOf(v).String())
			}
			it.vars = newVars
			return true
		}
	}
	return false
}

func (it *Recursive) Contains(ctx *graph.IterationContext, val graph.Value) bool {
	if it.VarsUpdated(ctx) {
		it.Reset()
	}

	graph.ContainsLogIn(it, val)
	it.pathIndex = 0
	key := graph.ToKey(val)
	if at, ok := it.seen[key]; ok {
		it.containsValue = it.getBaseValue(val)
		it.result.depth = at.depth
		it.result.val = val
		return graph.ContainsLogOut(it, val, true)
	}
	for it.Next(ctx) {
		if it.Result() == val {

			return graph.ContainsLogOut(it, val, true)
		}
	}
	return graph.ContainsLogOut(it, val, false)
}

func (it *Recursive) NextPath(ctx *graph.IterationContext) bool {
	if it.pathIndex+1 >= len(it.pathMap[graph.ToKey(it.containsValue)]) {
		return false
	}
	it.pathIndex++
	return true
}

func (it *Recursive) Close() error {
	err := it.subIt.Close()
	if err != nil {
		return err
	}
	for _, nextIt := range it.nextIts {
		err = nextIt.it.Close()
		if err != nil {
			return err
		}
	}
	it.seen = nil
	return it.err
}

func (it *Recursive) Type() graph.Type { return graph.Recursive }

func (it *Recursive) Optimize() (graph.Iterator, bool) {
	newIt, optimized := it.subIt.Optimize()
	if optimized {
		it.subIt = newIt
	}
	return it, false
}

func (it *Recursive) Size() (int64, bool) {
	return it.Stats().Size, false
}

func (it *Recursive) Stats() graph.IteratorStats {
	base := it.qs.FixedIterator()
	base.Add(Int64Node(20))
	fanoutit := it.morphism(it.qs, base)
	fanoutStats := fanoutit.Stats()
	subitStats := it.subIt.Stats()

	size := int64(math.Pow(float64(subitStats.Size*fanoutStats.Size), 5))
	return graph.IteratorStats{
		NextCost:     subitStats.NextCost + fanoutStats.NextCost,
		ContainsCost: (subitStats.NextCost+fanoutStats.NextCost)*(size/10) + subitStats.ContainsCost,
		Size:         size,
		Next:         it.runstats.Next,
		Contains:     it.runstats.Contains,
		ContainsNext: it.runstats.ContainsNext,
	}
}

func (it *Recursive) Describe() graph.Description {
	base := it.qs.FixedIterator()
	base.Add(Int64Node(20))
	fanoutdesc := it.morphism(it.qs, base).Describe()
	subIts := []graph.Description{
		it.subIt.Describe(),
		fanoutdesc,
	}

	return graph.Description{
		UID:       it.UID(),
		Type:      it.Type(),
		Tags:      it.tags.Tags(),
		Iterators: subIts,
	}
}
