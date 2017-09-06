package iterator

import (
	"testing"

	"github.com/codelingo/cayley/graph"
	"github.com/codelingo/cayley/quad"
	"github.com/stretchr/testify/require"
)

func TestCount(t *testing.T) {
	fixed := NewFixed(Identity,
		graph.PreFetched(quad.String("a")),
		graph.PreFetched(quad.String("b")),
		graph.PreFetched(quad.String("c")),
		graph.PreFetched(quad.String("d")),
		graph.PreFetched(quad.String("e")),
	)
	it := NewCount(fixed, nil)
	require.True(t, it.Next(nil))
	require.Equal(t, fetchedValue{Val: quad.Int(5)}, it.Result())
	require.False(t, it.Next(nil))
	require.True(t, it.Contains(nil, fetchedValue{Val: quad.Int(5)}))
	require.False(t, it.Contains(nil, fetchedValue{Val: quad.Int(3)}))

	fixed.Reset()

	fixed2 := NewFixed(Identity,
		graph.PreFetched(quad.String("b")),
		graph.PreFetched(quad.String("d")),
	)
	it = NewCount(NewAnd(nil, fixed, fixed2), nil)
	require.True(t, it.Next(nil))
	require.Equal(t, fetchedValue{Val: quad.Int(2)}, it.Result())
	require.False(t, it.Next(nil))
	require.False(t, it.Contains(nil, fetchedValue{Val: quad.Int(5)}))
	require.True(t, it.Contains(nil, fetchedValue{Val: quad.Int(2)}))

	it.Reset()
	it.Tagger().Add("count")
	require.True(t, it.Next(nil))
	m := make(map[string]graph.Value)
	it.TagResults(m)
	require.Equal(t, map[string]graph.Value{"count": graph.PreFetched(quad.Int(2))}, m)
}
