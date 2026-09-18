package app

import "testing"

func TestClassify(t *testing.T) {
	cases := []struct {
		s          ItemStats
		want       string
		reorderQty float64
	}{
		{ItemStats{Kind: "dish", Rate7: 30}, "fresh", 0},
		{ItemStats{Kind: "product", Unit: "pack", Qty: 0, Rate7: 5}, "out", 35},
		{ItemStats{Kind: "product", Unit: "pack", Qty: 6, Rate7: 5}, "critical", 29},
		{ItemStats{Kind: "ingredient", Unit: "kg", Qty: 3, Rate7: 1, Reorder: 1}, "low", 4},
		{ItemStats{Kind: "product", Unit: "pack", Qty: 40, Rate7: 1, Reorder: 5}, "overstock", 0},
		{ItemStats{Kind: "product", Unit: "pack", Qty: 40, Rate7: 0, Rate30: 0}, "dead", 0},
		{ItemStats{Kind: "product", Unit: "pack", Qty: 70, Rate7: 10, Reorder: 20, TrendPct: 50}, "ok", 35},
	}
	for _, c := range cases {
		s := c.s
		classify(&s)
		if s.Status != c.want || s.ReorderQty != c.reorderQty {
			t.Errorf("%+v → status %s reorder %v, want %s %v", c.s, s.Status, s.ReorderQty, c.want, c.reorderQty)
		}
	}
}

func TestGroundAdvice(t *testing.T) {
	d := 1.0
	items := []ItemStats{
		{ID: 1, Name: "Toor dal", Kind: "product", Unit: "kg", Status: "out", Rate7: 8, ReorderQty: 56},
		{ID: 2, Name: "Surf Excel 1kg", Kind: "product", Unit: "pack", Status: "overstock", Qty: 60, Rate7: 1, TypicalBuy: 10},
		{ID: 3, Name: "Maggi noodles", Kind: "product", Unit: "pack", Status: "critical", DaysLeft: &d, Rate7: 40, ReorderQty: 240},
		{ID: 4, Name: "Masala dosa", Kind: "dish", Unit: "plate", Status: "fresh", Rate7: 18},
	}
	res := &StockAdvice{
		BuyMore: []AdviceLine{{Item: "maggi noodles", Qty: 999}, {Item: "Masala dosa", Qty: 50}, {Item: "Imaginary item"}},
		BuyLess: []AdviceLine{{Item: "Toor dal"}, {Item: "Surf Excel 1kg", Qty: 500}},
	}
	groundAdvice(res, items, "en")
	got := map[string]float64{}
	for _, l := range res.BuyMore {
		got[l.Item] = l.Qty
	}
	if len(res.BuyMore) != 3 || got["Maggi noodles"] != 240 || got["Masala dosa"] != 0 || got["Toor dal"] != 56 {
		t.Fatalf("buy_more wrong: %+v", res.BuyMore)
	}
	if len(res.BuyLess) != 1 || res.BuyLess[0].Item != "Surf Excel 1kg" || res.BuyLess[0].Qty != 0 {
		t.Fatalf("buy_less wrong: %+v", res.BuyLess)
	}
}
