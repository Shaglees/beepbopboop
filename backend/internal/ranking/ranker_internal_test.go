package ranking

import "testing"

// TestProject_ShortVector_DoesNotPanic verifies project() does not panic
// when the input vector is shorter than a weight matrix row.
func TestProject_ShortVector_DoesNotPanic(t *testing.T) {
	w := [][]float32{{1, 2, 3, 4}} // row has 4 cols
	v := []float32{1, 2}           // only 2 elements — would OOB without guard

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("project() panicked with short input vector: %v", r)
		}
	}()
	_ = project(w, v)
}

// TestDot_MismatchedLengths verifies dot() returns 0 when vectors differ in
// length rather than panicking or reading past the end of the shorter slice.
func TestDot_MismatchedLengths(t *testing.T) {
	a := []float32{1, 2, 3}
	b := []float32{1, 2} // len(a) > len(b) — would OOB without guard

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("dot() panicked with mismatched lengths: %v", r)
		}
	}()
	result := dot(a, b)
	if result != 0 {
		t.Errorf("dot() mismatched lengths returned %f, want 0", result)
	}
}
