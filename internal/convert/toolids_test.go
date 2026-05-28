package convert

import "testing"

func TestClampCallID(t *testing.T) {
	short := "toolu_123"
	if got := ClampCallID(short); got != short {
		t.Fatalf("short id changed to %q", got)
	}

	long := "toolu_" + "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	got := ClampCallID(long)
	if len(got) != 64 {
		t.Fatalf("len = %d, want 64: %q", len(got), got)
	}
	want := "toolu_abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNO-b7def75d4e2f3b14"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
