package render_test

import (
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/convert/render"
)

//nolint:wsl // table-driven page-index assertions
func TestPlanCopyMappings(t *testing.T) {
	t.Parallel()

	plan, err := render.NewPlan([]int{1}, []int{2}, 2, false)
	if err != nil {
		t.Fatalf("NewPlan: %v", err)
	}
	if got, want := plan.LogicalN(), 3; got != want {
		t.Fatalf("LogicalN = %d, want %d", got, want)
	}
	if owner, ok := plan.OwnerOf(3); !ok || owner.Object != 1 || owner.Local != 0 {
		t.Fatalf("OwnerOf(3) = %#v, %v", owner, ok)
	}
	if got, want := plan.Remap(2, 3), 5; got != want {
		t.Fatalf("Remap = %d, want %d", got, want)
	}
}

//nolint:wsl // table-driven page-index assertions
func TestNonCollateOrder(t *testing.T) {
	t.Parallel()

	got, err := render.NonCollateOrder([]render.Range{{Start: 0, Count: 1}, {Start: 1, Count: 2}}, 2)
	if err != nil {
		t.Fatalf("NonCollateOrder: %v", err)
	}
	want := []int{0, 3, 1, 2, 4, 5}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("order[%d] = %d, want %d (full=%v)", index, got[index], want[index], got)
		}
	}
}

//nolint:wsl // table-driven seam checks keep all copy limits together.
//nolint:wsl // table-driven seam checks keep all copy limits together.
func TestCopySeamsShareValidation(t *testing.T) {
	t.Parallel()

	for _, copies := range []int{-1, 0, render.MaxCopies + 1} {
		if err := render.ValidateCopies(copies); err == nil {
			t.Errorf("ValidateCopies(%d) = nil", copies)
		}
		if _, err := render.NewPlan(nil, []int{1}, copies, true); err == nil {
			t.Errorf("NewPlan copies=%d = nil error", copies)
		}
		if _, err := render.NonCollateOrder([]render.Range{{Start: 0, Count: 1}}, copies); err == nil {
			t.Errorf("NonCollateOrder copies=%d = nil error", copies)
		}
	}
}
