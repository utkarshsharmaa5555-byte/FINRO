package app

import (
	"os"
	"testing"
	"time"
)

func TestParseBIRAC(t *testing.T) {
	html, err := os.ReadFile("testdata/birac_cfp.html")
	if err != nil {
		t.Fatal(err)
	}
	gs, err := parseBIRAC(html, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(gs) == 0 || len(gs) > 15 {
		t.Fatalf("got %d grants", len(gs))
	}
	g := gs[0]
	if g.Deadline == nil || *g.Deadline != "2026-07-15" || g.SourceURL != "https://birac.nic.in/cfp_view.php?id=118&scheme_type=6" {
		t.Fatalf("first grant parsed wrong: %+v", g)
	}
}

func TestVerifyQuoteAndStatus(t *testing.T) {
	page := "Loans up to  ₹10 lakh under Kishore.\nLast date: 30 November 2026"
	if !verifyQuote(page, "up to ₹10 lakh") || !verifyQuote(page, "last date: 30 November 2026") {
		t.Error("real quotes rejected")
	}
	if verifyQuote(page, "Last date: 31 December 2026") || verifyQuote(page, "") {
		t.Error("hallucinated quote accepted")
	}
	now := time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)
	past, future := "2026-07-15", "2026-11-30"
	if grantStatus(&past, now) != "closed" || grantStatus(&future, now) != "open" || grantStatus(nil, now) != "rolling" {
		t.Error("grant status wrong")
	}
}
