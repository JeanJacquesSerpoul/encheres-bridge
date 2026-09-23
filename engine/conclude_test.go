package engine

import "testing"

// TestConcludeHandlersTableIsWellFormed guards the structure of
// concludeHandlers itself: a duplicate or empty name, or a missing when/run,
// would silently break the priority order conclude.go documents. This is
// the cheap regression check that motivated turning the old "if" chain into
// an explicit table in the first place -- the table's shape is now
// something a test can assert on directly.
func TestConcludeHandlersTableIsWellFormed(t *testing.T) {
	if len(concludeHandlers) == 0 {
		t.Fatal("concludeHandlers is empty")
	}
	seen := map[string]bool{}
	for i, h := range concludeHandlers {
		if h.name == "" {
			t.Fatalf("handler #%d has no name", i)
		}
		if seen[h.name] {
			t.Fatalf("duplicate handler name %q", h.name)
		}
		seen[h.name] = true
		if h.when == nil {
			t.Fatalf("handler %q has a nil when", h.name)
		}
		if h.run == nil {
			t.Fatalf("handler %q has a nil run", h.name)
		}
	}
}
