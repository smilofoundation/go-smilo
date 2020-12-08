package pretty

import (
	"testing"
)

type difftest struct {
	a   interface{}
	b   interface{}
	exp []string
}

type S struct {
	A int
	S *S
	I interface{}
	C []int
}

type U struct {
	A int
	b int
}

var diffs = []difftest{
	{a: nil, b: nil},
	{a: S{A: 1}, b: S{A: 1}},

	{0, "", []string{`int != string`}},
	{0, 1, []string{`0 != 1`}},
	{S{}, new(S), []string{`pretty.S != *pretty.S`}},
	{"a", "b", []string{`"a" != "b"`}},
	{S{}, S{A: 1}, []string{`A: 0 != 1`}},
	{new(S), &S{A: 1}, []string{`A: 0 != 1`}},
	{S{S: new(S)}, S{S: &S{A: 1}}, []string{`S.A: 0 != 1`}},
	{S{}, S{I: 0}, []string{`I: nil != 0`}},
	{S{I: 1}, S{I: "x"}, []string{`I: int != string`}},
	{S{}, S{C: []int{1}}, []string{`C: []int[0] != []int[1]`, `C[0]: (missing) != 1`}},
	{S{C: []int{}}, S{C: []int{1}}, []string{`C: []int[0] != []int[1]`, `C[0]: (missing) != 1`}},
	{S{C: []int{1, 2, 3}}, S{C: []int{1, 2, 4}}, []string{`C[2]: 3 != 4`}},
	{S{}, S{A: 1, S: new(S)}, []string{`A: 0 != 1`, `S: nil != &{0 <nil> <nil> []}`}},

	// Unexported fields are not considered in diff.
	// TODO: Ideally, if there's no exported fields in a struct, the whole struct must be compared. Or no?
	{U{A: 1, b: 2}, U{A: 1, b: 3}, []string{}},
}

func TestDiff(t *testing.T) {
	for _, tt := range diffs {
		got := Diff(tt.a, tt.b)
		eq := len(got) == len(tt.exp)
		if eq {
			for i := range got {
				eq = eq && got[i] == tt.exp[i]
			}
		}
		if !eq {
			t.Errorf("diffing % #v", tt.a)
			t.Errorf("with    % #v", tt.b)
			diffdiff(t, got, tt.exp)
			continue
		}
	}
}

func TestDiffMessage(t *testing.T) {
	v1 := []int{0, 1, 2, 3}
	v2 := []int{0, 1, 4}

	if DiffMessage(v1, v1) != "" {
		t.FailNow()
	}
	if DiffMessage(v1, v2) == "" {
		t.FailNow()
	}
}

func diffdiff(t *testing.T, got, exp []string) {
	minus(t, "missing:", got, exp)
	minus(t, "unexpected:", exp, got)
}

func minus(t *testing.T, s string, a, b []string) {
	var i, j int
	for i = 0; i < len(a); i++ {
		for j = 0; j < len(b); j++ {
			if a[i] == b[j] {
				break
			}
		}
		if j == len(b) {
			t.Error(s, a[i])
		}
	}
}
