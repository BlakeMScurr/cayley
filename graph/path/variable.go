package path

import "github.com/codelingo/cayley/graph/iterator"

type Var struct {
	name string
	op   iterator.Operator
}

// NewVar creates a variable that can be used in place of a graph.Value
func NewVar(name string, op iterator.Operator) Var {
	return Var{
		name: name,
		op:   op,
	}
}

func (v Var) String() string {
	return v.name
}

func (v Var) Native() interface{} {
	return v.name
}

func (v Var) Operator() iterator.Operator {
	return v.op
}
