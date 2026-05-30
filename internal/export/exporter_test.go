package export

import "testing"

func TestGetCast(t *testing.T) {
	exp, err := Get("cast")
	if err != nil {
		t.Fatalf("Get(cast): %v", err)
	}
	if exp.Format() != "cast" {
		t.Fatalf("Format = %q, want cast", exp.Format())
	}
}

func TestGetUnknown(t *testing.T) {
	if _, err := Get("bogus"); err == nil {
		t.Fatal("Get(bogus) err = nil, want error")
	}
}
