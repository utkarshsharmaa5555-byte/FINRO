package app

import "testing"

func TestUPILink(t *testing.T) {
	got, err := upiLink("lakshmi.idli@okaxis", "Amma Idli Bandi", 0)
	if err != nil || got != "upi://pay?cu=INR&pa=lakshmi.idli%40okaxis&pn=Amma%20Idli%20Bandi" {
		t.Fatalf("got %q %v", got, err)
	}
	got, _ = upiLink("a.b@ybl", "Shop", 12050)
	if got != "upi://pay?am=120.50&cu=INR&pa=a.b%40ybl&pn=Shop" {
		t.Fatalf("amount link %q", got)
	}
	for _, bad := range []string{"no-at-sign", "x@", "@bank", "a@b1"} {
		if _, err := upiLink(bad, "Shop", 0); err == nil {
			t.Errorf("accepted bad vpa %q", bad)
		}
	}
	if _, err := upiLink("a.b@ybl", "", 0); err == nil {
		t.Error("accepted empty payee")
	}
	if _, err := upiLink("a.b@ybl", "Shop", -1); err == nil {
		t.Error("accepted negative amount")
	}
}
