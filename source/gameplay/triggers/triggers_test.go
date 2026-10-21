package triggers

import (
	"testing"

	maths "gokeen/engine/mathematics"
)

func TestReached(t *testing.T) {
	a := maths.V3(0, 10, 0)
	if !Reached(maths.V3(1, 12, 1), a, 3, 6) {
		t.Fatal("expected trigger")
	}
	if Reached(maths.V3(10, 10, 0), a, 3, 6) {
		t.Fatal("unexpected horizontal trigger")
	}
	if Reached(maths.V3(0, 20, 0), a, 3, 6) {
		t.Fatal("unexpected vertical trigger")
	}
}
