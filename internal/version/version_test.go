package version

import "testing"

func TestString_usesInjectedVersion(t *testing.T) {
	old := Version
	t.Cleanup(func() { Version = old })
	Version = "  v1.2.3  "
	if got := String(); got != "v1.2.3" {
		t.Fatalf("String() = %q, want v1.2.3", got)
	}
}

func TestString_nonEmptyWithoutInjection(t *testing.T) {
	old := Version
	t.Cleanup(func() { Version = old })
	Version = ""
	got := String()
	if got == "" {
		t.Fatal("String() returned empty")
	}
}
