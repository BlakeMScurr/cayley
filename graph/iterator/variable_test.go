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
	"testing"

	"github.com/codelingo/cayley/graph"
)

func TestSimpleVariableIterator(t *testing.T) {
	qs := &oldstore{
		data: []string{"a", "b", "c", "d"},
		iter: NewFixed(Identity),
	}

	binder := NewVariable(qs, "var1")
	user1 := NewVariable(qs, "var1")
	user2 := NewVariable(qs, "var1")

	ctx := graph.NewIterationContext()
	var count int

	for binder.Next(ctx) {
		count++
		checkUserAgainstBinder(t, ctx, user1, binder)
		checkUserAgainstBinder(t, ctx, user2, binder)
	}
	fmt.Println(count)
	if count != 4 {
		t.Error("Variable binder should iterate over all nodes")
	}
}

func checkUserAgainstBinder(t *testing.T, ctx *graph.IterationContext, user, binder *Variable) {
	if !user.Contains(ctx, binder.Result()) {
		t.Error("User should represent the current result of the binder.")
	}
}
