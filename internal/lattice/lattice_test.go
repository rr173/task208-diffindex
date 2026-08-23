package lattice

import (
	"math"
	"testing"

	"task208-diffindex/internal/model"
)

func TestReciprocalRoundTripAndMillerAssignment(t *testing.T) {
	want := model.CellParams{A: 10, B: 12, C: 15, Alpha: 90, Beta: 90, Gamma: 90}
	r := BuildReciprocal(want)
	q := r.Vector(2, -1, 3)
	h, k, l, err := r.AssignMiller(q)
	if err != nil {
		t.Fatal(err)
	}
	if h != 2 || k != -1 || l != 3 {
		t.Fatalf("assigned Miller = (%d,%d,%d), want (2,-1,3)", h, k, l)
	}

	got, err := CellFromReciprocal(r.As, r.Bs, r.Cs)
	if err != nil {
		t.Fatal(err)
	}
	for name, actual := range map[string][2]float64{
		"a":     [2]float64{got.A, want.A},
		"b":     [2]float64{got.B, want.B},
		"c":     [2]float64{got.C, want.C},
		"alpha": [2]float64{got.Alpha, want.Alpha},
		"beta":  [2]float64{got.Beta, want.Beta},
		"gamma": [2]float64{got.Gamma, want.Gamma},
	} {
		if math.Abs(actual[0]-actual[1]) > 1e-9 {
			t.Fatalf("%s = %v, want %v; recovered=%+v", name, actual[0], actual[1], got)
		}
	}
}

func TestCellFromReciprocalRejectsDegenerateBasis(t *testing.T) {
	_, err := CellFromReciprocal([3]float64{1, 0, 0}, [3]float64{2, 0, 0}, [3]float64{3, 0, 0})
	if err == nil {
		t.Fatal("expected degenerate reciprocal basis to be rejected")
	}
}
