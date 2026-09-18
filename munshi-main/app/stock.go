package app

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// ---------- seed data ----------

// seedItem describes 45 days of synthetic but consistent history: daily units sold/used follow
// Base × weekday factor × a linear trend (Trend = multiplier reached by today) × small noise.
type seedItem struct {
	Name, Kind, Category, Unit   string
	Price, Cost                  float64 // rupees per unit
	Base                         float64 // units sold/used per day at the start of the window
	Trend                        float64 // 1.0 flat, 1.4 = +40% by today
	Weekend                      float64 // multiplier on Sat/Sun
	StartQty, Reorder, RestockTo float64
	StopRestock                  int // days before today after which no restock happened (creates low stock)
}

var seedStock = map[string]struct {
	RestockEvery int
	ClosedSunday bool
	Items        []seedItem
}{
	"9000000001": {RestockEvery: 3, ClosedSunday: true, Items: []seedItem{
		{"Idli (plate of 4)", "dish", "breakfast", "plate", 40, 14, 38, 1.05, 1.3, 0, 0, 0, 0},
		{"Masala dosa", "dish", "breakfast", "plate", 60, 22, 14, 1.55, 1.4, 0, 0, 0, 0},
		{"Plain dosa", "dish", "breakfast", "plate", 40, 13, 16, 0.8, 1.1, 0, 0, 0, 0},
		{"Medu vada (2 pcs)", "dish", "breakfast", "plate", 30, 11, 20, 0.62, 1.2, 0, 0, 0, 0},
		{"Pesarattu", "dish", "breakfast", "plate", 50, 18, 6, 1.3, 1.5, 0, 0, 0, 0},
		{"Idli rice", "ingredient", "grains", "kg", 0, 42, 4.2, 1.05, 1.25, 30, 8, 30, 0},
		{"Urad dal", "ingredient", "pulses", "kg", 0, 125, 1.6, 1.05, 1.25, 10, 3, 10, 6},
		{"Groundnut oil", "ingredient", "oil", "litre", 0, 165, 0.9, 0.9, 1.2, 8, 2, 8, 0},
		{"Coconut", "ingredient", "fresh", "pcs", 0, 25, 6, 1.1, 1.3, 25, 10, 25, 0},
		{"Cooking gas (LPG)", "ingredient", "fuel", "cylinder", 0, 1110, 0.09, 1.0, 1.1, 2, 1, 2, 12},
		{"Moong dal", "ingredient", "pulses", "kg", 0, 118, 0.3, 1.3, 1.4, 12, 1, 12, 0},
	}},
	"9000000002": {RestockEvery: 4, Items: []seedItem{
		{"Aashirvaad atta 5kg", "product", "staples", "pack", 295, 262, 12, 1.0, 1.4, 80, 20, 80, 0},
		{"Loose sugar", "product", "staples", "kg", 48, 41, 15, 0.72, 1.2, 100, 25, 100, 0},
		{"Fortune mustard oil 1L", "product", "oil", "bottle", 185, 160, 10, 1.1, 1.3, 60, 15, 60, 5},
		{"Toor dal", "product", "pulses", "kg", 168, 142, 8, 1.05, 1.2, 50, 12, 50, 9},
		{"Parle-G biscuits", "product", "snacks", "pack", 10, 8.3, 70, 1.25, 1.5, 400, 120, 400, 7},
		{"Maggi noodles", "product", "snacks", "pack", 15, 12.2, 40, 1.6, 1.6, 250, 60, 250, 0},
		{"Local namkeen 200g", "product", "snacks", "pack", 45, 30, 18, 1.45, 1.5, 90, 25, 90, 0},
		{"Amul milk 500ml", "product", "dairy", "packet", 34, 31.5, 60, 1.1, 1.3, 120, 40, 120, 0},
		{"Surf Excel 1kg", "product", "household", "pack", 150, 124, 1.2, 0.68, 1.2, 95, 6, 60, 0},
		{"Lifebuoy soap", "product", "household", "bar", 40, 33, 7, 0.95, 1.2, 50, 15, 50, 0},
		{"Cold drinks 750ml", "product", "beverages", "bottle", 50, 38, 16, 1.7, 1.7, 100, 30, 100, 0},
	}},
	"9000000000": {RestockEvery: 2, ClosedSunday: true, Items: []seedItem{
		{"Masala chai", "dish", "beverages", "cup", 15, 5, 120, 1.1, 0.8, 0, 0, 0, 0},
		{"Samosa", "dish", "snacks", "pcs", 15, 7, 45, 0.85, 0.9, 0, 0, 0, 0},
		{"Bun maska", "dish", "snacks", "plate", 25, 9, 18, 1.4, 0.9, 0, 0, 0, 0},
		{"Milk", "ingredient", "dairy", "litre", 0, 56, 9, 1.05, 0.85, 20, 8, 25, 3},
		{"Tea powder", "ingredient", "beverages", "kg", 0, 480, 0.35, 1.05, 0.85, 3, 1, 3, 0},
		{"Sugar", "ingredient", "staples", "kg", 0, 42, 1.6, 1.05, 0.85, 12, 4, 15, 0},
		{"Paper cups", "ingredient", "packaging", "pcs", 0, 0.8, 125, 1.05, 0.85, 1500, 400, 2000, 0},
	}},
}

// noise is a deterministic ±12% wobble so the seed looks lived-in but is reproducible.
func noise(day, item int) float64 { return 1 + 0.12*math.Sin(float64(day)*1.7+float64(item)*2.3) }

// seedStockHistory creates items, 45 days of sold/used + restock moves, day-sales and restock expenses in the ledger.
func seedStockHistory(ctx context.Context, tx pgx.Tx, uid int64, phone string) error {
	cfg, ok := seedStock[phone]
	if !ok {
		return nil
	}
	const days = 45
	ids := make([]int64, len(cfg.Items))
	qty := make([]float64, len(cfg.Items))
	for i, it := range cfg.Items {
		if err := tx.QueryRow(ctx, `INSERT INTO stock_items(user_id, name, kind, category, unit, qty, reorder_level, cost_paise, price_paise)
			VALUES($1,$2,$3,$4,$5,0,$6,$7,$8) RETURNING id`, uid, it.Name, it.Kind, it.Category, it.Unit, it.Reorder,
			int64(math.Round(it.Cost*100)), int64(math.Round(it.Price*100))).Scan(&ids[i]); err != nil {
			return err
		}
		qty[i] = it.StartQty
	}
	today := time.Now().In(ist)
	for d := days; d >= 1; d-- {
		day := today.AddDate(0, 0, -d)
		if cfg.ClosedSunday && day.Weekday() == time.Sunday {
			continue
		}
		progress := float64(days-d) / float64(days)
		var salePaise int64
		for i, it := range cfg.Items {
			f := 1 + (it.Trend-1)*progress*progress // accelerating trend so recent weeks show it clearly
			if day.Weekday() == time.Saturday || day.Weekday() == time.Sunday {
				f *= it.Weekend
			}
			units := it.Base * f * noise(d, i)
			if countable(it.Unit) {
				units = math.Round(units)
			} else {
				units = math.Round(units*100) / 100
			}
			if it.Kind != "dish" {
				units = math.Min(units, qty[i]) // cannot sell or use what is not on the shelf
			}
			if units > 0 {
				kind, amount := "sold", int64(math.Round(units*it.Price*100))
				if it.Kind == "ingredient" {
					kind, amount = "used", 0
				}
				if _, err := tx.Exec(ctx, `INSERT INTO stock_moves(user_id, item_id, move_date, kind, qty, amount_paise) VALUES($1,$2,$3,$4,$5,$6)`,
					uid, ids[i], day, kind, units, amount); err != nil {
					return err
				}
				salePaise += amount
				if it.Kind != "dish" {
					qty[i] -= units
				}
			}
			// On buying days, restock anything that would not last until the next buying day (sized to demand),
			// except items deliberately left to run low for the demo.
			demand := it.Base * (1 + (it.Trend-1)*progress*progress) * float64(cfg.RestockEvery+1)
			if it.Kind != "dish" && d%cfg.RestockEvery == 0 && qty[i] <= math.Max(it.Reorder, demand) && (it.StopRestock == 0 || d > it.StopRestock) {
				target := math.Max(it.RestockTo, demand*2.2)
				buy := math.Round((target-qty[i])*100) / 100
				if countable(it.Unit) {
					buy = math.Ceil(target - qty[i])
				}
				if buy > 0 {
					cost := int64(math.Round(buy * it.Cost * 100))
					if _, err := tx.Exec(ctx, `INSERT INTO stock_moves(user_id, item_id, move_date, kind, qty, amount_paise) VALUES($1,$2,$3,'bought',$4,$5)`,
						uid, ids[i], day, buy, cost); err != nil {
						return err
					}
					qty[i] += buy
					if _, err := tx.Exec(ctx, `INSERT INTO ledger_entries(user_id, entry_date, kind, amount_paise, note, party, source, item_id, qty) VALUES($1,$2,'expense',$3,$4,$5,'text',$6,$7)`,
						uid, day, cost, "Stock purchase — "+it.Name, supplierFor(phone, it.Category), ids[i], buy); err != nil {
						return err
					}
				}
			}
		}
		if salePaise > 0 {
			if _, err := tx.Exec(ctx, `INSERT INTO ledger_entries(user_id, entry_date, kind, amount_paise, note, source) VALUES($1,$2,'sale',$3,'Day sales','text')`,
				uid, day, salePaise); err != nil {
				return err
			}
		}
	}
	for i := range cfg.Items {
		if _, err := tx.Exec(ctx, `UPDATE stock_items SET qty=$1 WHERE id=$2`, math.Round(qty[i]*100)/100, ids[i]); err != nil {
			return err
		}
	}
	return nil
}

// ---------- stats (deterministic, no AI) ----------

type ItemStats struct {
	ID         int64    `json:"id"`
	Name       string   `json:"name"`
	Kind       string   `json:"kind"`
	Category   string   `json:"category"`
	Unit       string   `json:"unit"`
	Qty        float64  `json:"qty"`
	Reorder    float64  `json:"reorder_level"`
	CostPaise  int64    `json:"cost_paise"`
	PricePaise int64    `json:"price_paise"`
	Rate7      float64  `json:"rate_7d"`   // units per day, last 7 days
	Rate30     float64  `json:"rate_30d"`  // units per day, last 30 days
	TrendPct   float64  `json:"trend_pct"` // last 7d vs previous 7d
	DaysLeft   *float64 `json:"days_left"`
	Revenue30  int64    `json:"revenue_30d_paise"`
	MarginPct  float64  `json:"margin_pct"`
	Status     string   `json:"status"`      // out | critical | low | ok | overstock | dead | fresh
	ReorderQty float64  `json:"reorder_qty"` // to cover 7 days at trend-adjusted rate
	TypicalBuy float64  `json:"typical_buy"` // average purchase size in last 30 days
	LastBought *string  `json:"last_bought"`
}

func round1(f float64) float64 { return math.Round(f*10) / 10 }

// classify turns run-rate and stock into an alert status. Dishes are cooked fresh, so they only report demand.
func classify(s *ItemStats) {
	if s.Kind == "dish" {
		s.Status = "fresh"
		return
	}
	rate := s.Rate7
	if rate == 0 {
		rate = s.Rate30
	}
	if rate > 0 {
		dl := round1(s.Qty / rate)
		s.DaysLeft = &dl
	}
	switch {
	case s.Qty <= 0:
		s.Status = "out"
	case rate == 0:
		s.Status = "dead"
	case *s.DaysLeft < 2:
		s.Status = "critical"
	case *s.DaysLeft < 5 || s.Qty <= s.Reorder:
		s.Status = "low"
	case *s.DaysLeft > 30:
		s.Status = "overstock"
	default:
		s.Status = "ok"
	}
	trendRate := rate * math.Max(0.5, 1+s.TrendPct/100)
	need := trendRate*7 - s.Qty
	if need < 0 {
		need = 0
	}
	if s.Unit == "kg" || s.Unit == "litre" || s.Unit == "cylinder" {
		s.ReorderQty = math.Ceil(need*2) / 2
	} else {
		s.ReorderQty = math.Ceil(need)
	}
}

func stockStats(ctx context.Context, uid int64) ([]ItemStats, error) {
	db, err := DB()
	if err != nil {
		return nil, err
	}
	// Windows end at the latest day with activity (today once something is sold), so a quiet morning does not look like a slump.
	rows, err := db.Query(ctx, `
		WITH a AS (SELECT LEAST(CURRENT_DATE, COALESCE(MAX(move_date), CURRENT_DATE)) AS d FROM stock_moves WHERE user_id = $1 AND kind IN ('sold','used'))
		SELECT i.id, i.name, i.kind, i.category, i.unit, i.qty, i.reorder_level, i.cost_paise, i.price_paise,
		  COALESCE(SUM(m.qty) FILTER (WHERE m.kind IN ('sold','used') AND m.move_date > a.d - 7 AND m.move_date <= a.d), 0) / 7.0,
		  COALESCE(SUM(m.qty) FILTER (WHERE m.kind IN ('sold','used') AND m.move_date > a.d - 30 AND m.move_date <= a.d), 0) / 30.0,
		  COALESCE(SUM(m.qty) FILTER (WHERE m.kind IN ('sold','used') AND m.move_date <= a.d - 7 AND m.move_date > a.d - 28), 0) / 21.0,
		  COALESCE(SUM(m.amount_paise) FILTER (WHERE m.kind = 'sold' AND m.move_date > a.d - 30), 0),
		  COALESCE(AVG(m.qty) FILTER (WHERE m.kind = 'bought' AND m.move_date > CURRENT_DATE - 30), 0),
		  to_char(MAX(m.move_date) FILTER (WHERE m.kind = 'bought'), 'YYYY-MM-DD')
		FROM a, stock_items i LEFT JOIN stock_moves m ON m.item_id = i.id
		WHERE i.user_id = $1
		GROUP BY i.id, a.d ORDER BY i.kind, i.name`, uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ItemStats{}
	for rows.Next() {
		var s ItemStats
		var prev7 float64
		if err := rows.Scan(&s.ID, &s.Name, &s.Kind, &s.Category, &s.Unit, &s.Qty, &s.Reorder, &s.CostPaise, &s.PricePaise,
			&s.Rate7, &s.Rate30, &prev7, &s.Revenue30, &s.TypicalBuy, &s.LastBought); err != nil {
			return nil, err
		}
		if prev7 > 0 {
			s.TrendPct = math.Round((s.Rate7 - prev7) / prev7 * 100)
		}
		if s.PricePaise > 0 {
			s.MarginPct = math.Round(float64(s.PricePaise-s.CostPaise) / float64(s.PricePaise) * 100)
		}
		s.Rate7, s.Rate30, s.TypicalBuy = round1(s.Rate7), round1(s.Rate30), round1(s.TypicalBuy)
		classify(&s)
		out = append(out, s)
	}
	return out, rows.Err()
}

// ---------- AI purchase advice ----------

type AdviceLine struct {
	Item    string  `json:"item"`
	Why     string  `json:"why"`
	Qty     float64 `json:"qty"`
	Unit    string  `json:"unit"`
	Urgency string  `json:"urgency,omitempty"`
	ItemID  int64   `json:"item_id"`
}

type StockAdvice struct {
	Headline   string       `json:"headline"`
	BuyMore    []AdviceLine `json:"buy_more"`
	BuyLess    []AdviceLine `json:"buy_less"`
	KeepSteady []string     `json:"keep_steady"`
	Insights   []string     `json:"insights"`
	MoneyTip   string       `json:"money_tip"`
	CreatedAt  string       `json:"created_at"`
}

var adviceSchema = obj(map[string]any{
	"headline": str("one short line summarising this week, reply language"),
	"buy_more": arr(obj(map[string]any{
		"item":    str("exact item name from DATA"),
		"why":     str("1 sentence with the numbers that justify it, reply language"),
		"urgency": enum("today", "this_week"),
	})),
	"buy_less": arr(obj(map[string]any{
		"item": str("exact item name from DATA"),
		"why":  str("1 sentence with the numbers that justify it, reply language"),
	})),
	"keep_steady": arr(str("exact item names that need no change")),
	"insights":    arr(str("2-4 observations about which dishes/products are rising or falling and what that means, reply language")),
	"money_tip":   str("one practical tip on cash or margins, reply language"),
})

func stockAdvice(ctx context.Context, uid int64, lang string) (*StockAdvice, error) {
	items, err := stockStats(ctx, uid)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, httpErr(409, "add a few stock items first")
	}
	profile, _, _ := loadProfile(ctx, uid)
	byName := map[string]ItemStats{}
	data := make([]map[string]any, 0, len(items))
	for _, s := range items {
		byName[strings.ToLower(s.Name)] = s
		row := map[string]any{"item": s.Name, "type": s.Kind, "unit": s.Unit, "sold_or_used_per_day_7d": s.Rate7, "per_day_30d": s.Rate30,
			"trend_last7d_vs_prior3weeks_pct": s.TrendPct, "margin_pct": s.MarginPct, "revenue_30d_rupees": s.Revenue30 / 100}
		if s.Kind != "dish" {
			row["in_stock"], row["days_of_stock_left"], row["status"], row["typical_purchase"] = s.Qty, s.DaysLeft, s.Status, s.TypicalBuy
			if restockNow(s.Status) {
				row["restock_now"] = true
				row["note"] = "recent sales are held back by an empty or nearly empty shelf, not by falling demand"
			}
		}
		data = append(data, row)
	}
	sys := "You are Finro, advising an Indian small merchant what to buy more of and less of for next week, based ONLY on DATA computed from their records. " +
		"Dishes are cooked fresh: if a dish is rising, recommend buying more of its ingredients (use ingredient item names). " +
		"buy_more: items that will run out within a week, are rising, or have strong margins. buy_less: items with falling sales, overstock or dead stock. Items with restock_now MUST be in buy_more and NEVER in buy_less. " +
		"Only use exact item names from DATA. Do not invent quantities — the app computes them. Mention the numbers (per day, days left, trend %) in 'why'. " +
		"Write in " + langNames[lang] + " (native script), simple words."
	var res StockAdvice
	user := mustJSON(map[string]any{"business": profile["business_type"], "city": profile["city"], "today": time.Now().In(ist).Format("Monday 2 Jan"), "data": data})
	if err := chatJSON(ctx, "stock_advice", sys, user, adviceSchema, &res); err != nil {
		return nil, err
	}
	groundAdvice(&res, items, lang)
	res.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	if db, err := DB(); err == nil {
		db.Exec(ctx, `INSERT INTO ai_insights(user_id, kind, lang, data) VALUES($1,'stock_advice',$2,$3)
			ON CONFLICT (user_id, kind, lang) DO UPDATE SET data=EXCLUDED.data, created_at=now()`, uid, lang, res)
	}
	return &res, nil
}

func cachedAdvice(ctx context.Context, uid int64, lang string) *StockAdvice {
	db, err := DB()
	if err != nil {
		return nil
	}
	var a StockAdvice
	if db.QueryRow(ctx, `SELECT data FROM ai_insights WHERE user_id=$1 AND kind='stock_advice' AND lang=$2`, uid, lang).Scan(&a) != nil {
		return nil
	}
	return &a
}

// ---------- overview ----------

type Overview struct {
	TodaySalesPaise     int64            `json:"today_sales_paise"`
	YesterdaySalesPaise int64            `json:"yesterday_sales_paise"`
	Avg7SalesPaise      int64            `json:"avg7_sales_paise"`
	Avg30SalesPaise     int64            `json:"avg30_sales_paise"`
	Records             *RecordsSummary  `json:"records"`
	Weekdays            []map[string]any `json:"weekdays"`
	Items               []ItemStats      `json:"items"`
	Alerts              []ItemStats      `json:"alerts"`
	TopSellers          []ItemStats      `json:"top_sellers"`
	SlowMovers          []ItemStats      `json:"slow_movers"`
	Advice              *StockAdvice     `json:"advice"`
	StockValuePaise     int64            `json:"stock_value_paise"`
}

func buildOverview(ctx context.Context, uid int64, lang string) (*Overview, error) {
	db, err := DB()
	if err != nil {
		return nil, err
	}
	o := &Overview{Weekdays: []map[string]any{}}
	if err := db.QueryRow(ctx, `SELECT
		COALESCE(SUM(amount_paise) FILTER (WHERE entry_date = CURRENT_DATE), 0),
		COALESCE(SUM(amount_paise) FILTER (WHERE entry_date = CURRENT_DATE - 1), 0),
		(COALESCE(SUM(amount_paise) FILTER (WHERE entry_date > CURRENT_DATE - 7), 0) / 7)::bigint,
		(COALESCE(SUM(amount_paise) FILTER (WHERE entry_date > CURRENT_DATE - 30), 0) / 30)::bigint
		FROM ledger_entries WHERE user_id=$1 AND kind='sale'`, uid).Scan(&o.TodaySalesPaise, &o.YesterdaySalesPaise, &o.Avg7SalesPaise, &o.Avg30SalesPaise); err != nil {
		return nil, err
	}
	if o.Records, err = recordsSummary(ctx, uid, 30); err != nil {
		return nil, err
	}
	rows, err := db.Query(ctx, `SELECT EXTRACT(ISODOW FROM entry_date)::int, AVG(t)::bigint FROM
		(SELECT entry_date, SUM(amount_paise) t FROM ledger_entries WHERE user_id=$1 AND kind='sale' AND entry_date > CURRENT_DATE - 56 GROUP BY entry_date) d
		GROUP BY 1 ORDER BY 1`, uid)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var dow int
		var avg int64
		if err := rows.Scan(&dow, &avg); err != nil {
			rows.Close()
			return nil, err
		}
		o.Weekdays = append(o.Weekdays, map[string]any{"dow": dow, "avg_paise": avg})
	}
	rows.Close()
	if o.Items, err = stockStats(ctx, uid); err != nil {
		return nil, err
	}
	o.Alerts, o.TopSellers, o.SlowMovers = []ItemStats{}, []ItemStats{}, []ItemStats{}
	rank := map[string]int{"out": 0, "critical": 1, "low": 2}
	for _, s := range o.Items {
		if _, alert := rank[s.Status]; alert {
			o.Alerts = append(o.Alerts, s)
		}
		if s.Kind != "ingredient" {
			o.TopSellers = append(o.TopSellers, s)
		}
		if s.Kind != "dish" {
			o.StockValuePaise += int64(s.Qty * float64(s.CostPaise))
		}
		_, alerted := rank[s.Status]
		if s.Status == "overstock" || s.Status == "dead" || (!alerted && s.Kind != "ingredient" && s.TrendPct <= -12) {
			o.SlowMovers = append(o.SlowMovers, s)
		}
	}
	slices.SortFunc(o.Alerts, func(a, b ItemStats) int {
		if rank[a.Status] != rank[b.Status] {
			return rank[a.Status] - rank[b.Status]
		}
		return int((derefOr(a.DaysLeft, 0) - derefOr(b.DaysLeft, 0)) * 10)
	})
	slices.SortFunc(o.TopSellers, func(a, b ItemStats) int { return int(b.Revenue30 - a.Revenue30) })
	o.Advice = cachedAdvice(ctx, uid, lang)
	return o, nil
}

func derefOr(p *float64, d float64) float64 {
	if p == nil {
		return d
	}
	return *p
}

// stockBrief is a short text for the agent/blueprint context.
func stockBrief(ctx context.Context, uid int64) map[string]any {
	items, err := stockStats(ctx, uid)
	if err != nil || len(items) == 0 {
		return nil
	}
	var alerts, rising, falling, sellers []string
	for _, s := range items {
		switch s.Status {
		case "out", "critical", "low":
			alerts = append(alerts, fmt.Sprintf("%s: %.1f %s left (~%.1f days)", s.Name, s.Qty, s.Unit, derefOr(s.DaysLeft, 0)))
		}
		if s.Kind != "ingredient" {
			sellers = append(sellers, fmt.Sprintf("%s %.1f %s/day", s.Name, s.Rate7, s.Unit))
			if s.TrendPct >= 15 {
				rising = append(rising, fmt.Sprintf("%s +%.0f%%", s.Name, s.TrendPct))
			} else if s.TrendPct <= -15 {
				falling = append(falling, fmt.Sprintf("%s %.0f%%", s.Name, s.TrendPct))
			}
		}
	}
	return map[string]any{"low_stock": alerts, "rising_items": rising, "falling_items": falling, "run_rates": sellers}
}

// ---------- item matching for khata entries ----------

func findItem(ctx context.Context, uid int64, name string) (*ItemStats, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, nil
	}
	db, err := DB()
	if err != nil {
		return nil, err
	}
	var s ItemStats
	err = db.QueryRow(ctx, `SELECT id, name, kind, unit, price_paise, cost_paise FROM stock_items WHERE user_id=$1
		ORDER BY (lower(name) = lower($2)) DESC, (name ILIKE '%' || $2 || '%' OR $2 ILIKE '%' || name || '%') DESC LIMIT 1`, uid, name).
		Scan(&s.ID, &s.Name, &s.Kind, &s.Unit, &s.PricePaise, &s.CostPaise)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !strings.EqualFold(s.Name, name) && !strings.Contains(strings.ToLower(s.Name), strings.ToLower(name)) && !strings.Contains(strings.ToLower(name), strings.ToLower(s.Name)) {
		return nil, nil
	}
	return &s, nil
}

func itemNames(ctx context.Context, uid int64) []string {
	db, err := DB()
	if err != nil {
		return nil
	}
	rows, err := db.Query(ctx, `SELECT name FROM stock_items WHERE user_id=$1 ORDER BY name`, uid)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var n string
		if rows.Scan(&n) == nil {
			out = append(out, n)
		}
	}
	return out
}

// ---------- handlers ----------

func handleStock(w http.ResponseWriter, r *http.Request, u *User) error {
	items, err := stockStats(r.Context(), u.ID)
	if err != nil {
		return err
	}
	return writeJSON(w, map[string]any{"items": items, "advice": cachedAdvice(r.Context(), u.ID, u.Lang)})
}

var itemKinds = map[string]bool{"product": true, "dish": true, "ingredient": true}

func handleCreateItem(w http.ResponseWriter, r *http.Request, u *User) error {
	var in struct {
		Name, Kind, Category, Unit string
		Qty                        float64 `json:"qty"`
		Reorder                    float64 `json:"reorder_level"`
		CostPaise                  int64   `json:"cost_paise"`
		PricePaise                 int64   `json:"price_paise"`
	}
	if err := readJSON(r, &in); err != nil {
		return err
	}
	in.Name, in.Unit = strings.TrimSpace(in.Name), strings.TrimSpace(in.Unit)
	switch {
	case in.Name == "" || len(in.Name) > 80:
		return httpErr(400, "enter an item name")
	case !itemKinds[in.Kind]:
		return httpErr(400, "choose product, dish or ingredient")
	case in.Qty < 0 || in.Reorder < 0 || in.CostPaise < 0 || in.PricePaise < 0:
		return httpErr(400, "numbers cannot be negative")
	}
	if in.Unit == "" {
		in.Unit = "pcs"
	}
	db, err := DB()
	if err != nil {
		return err
	}
	var id int64
	err = db.QueryRow(r.Context(), `INSERT INTO stock_items(user_id, name, kind, category, unit, qty, reorder_level, cost_paise, price_paise)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT (user_id, name) DO NOTHING RETURNING id`,
		u.ID, in.Name, in.Kind, strings.TrimSpace(in.Category), in.Unit, in.Qty, in.Reorder, in.CostPaise, in.PricePaise).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return httpErr(409, "an item with this name already exists")
	}
	if err != nil {
		return err
	}
	return writeJSON(w, map[string]any{"id": id})
}

// handleMoveItem records a restock/wastage/sale/adjustment and keeps qty consistent in one transaction.
func handleMoveItem(w http.ResponseWriter, r *http.Request, u *User) error {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	var in struct {
		Kind    string   `json:"kind"` // bought | sold | used | wasted | adjust (qty = new absolute count)
		Qty     float64  `json:"qty"`
		Reorder *float64 `json:"reorder_level"`
	}
	if err := readJSON(r, &in); err != nil {
		return err
	}
	if in.Qty < 0 || in.Qty > 1e7 {
		return httpErr(400, "quantity must be 0 or more")
	}
	db, err := DB()
	if err != nil {
		return err
	}
	tx, err := db.Begin(r.Context())
	if err != nil {
		return err
	}
	defer tx.Rollback(r.Context())
	var cur float64
	var cost, price int64
	if err := tx.QueryRow(r.Context(), `SELECT qty, cost_paise, price_paise FROM stock_items WHERE id=$1 AND user_id=$2 FOR UPDATE`, id, u.ID).Scan(&cur, &cost, &price); err != nil {
		return httpErr(404, "item not found")
	}
	if in.Reorder != nil && *in.Reorder >= 0 {
		if _, err := tx.Exec(r.Context(), `UPDATE stock_items SET reorder_level=$1 WHERE id=$2`, *in.Reorder, id); err != nil {
			return err
		}
	}
	if in.Kind != "" {
		next, move, amount := cur, in.Qty, int64(0)
		switch in.Kind {
		case "bought":
			next, amount = cur+in.Qty, int64(in.Qty*float64(cost))
		case "sold":
			next, amount = math.Max(0, cur-in.Qty), int64(in.Qty*float64(price))
		case "used", "wasted":
			next = math.Max(0, cur-in.Qty)
		case "adjust":
			next, move = in.Qty, in.Qty-cur
		default:
			return httpErr(400, "unknown movement")
		}
		if _, err := tx.Exec(r.Context(), `INSERT INTO stock_moves(user_id, item_id, kind, qty, amount_paise) VALUES($1,$2,$3,$4,$5)`, u.ID, id, in.Kind, move, amount); err != nil {
			return err
		}
		if _, err := tx.Exec(r.Context(), `UPDATE stock_items SET qty=$1, updated_at=now() WHERE id=$2`, math.Round(next*100)/100, id); err != nil {
			return err
		}
	}
	if err := tx.Commit(r.Context()); err != nil {
		return err
	}
	return writeJSON(w, map[string]bool{"ok": true})
}

func handleDeleteItem(w http.ResponseWriter, r *http.Request, u *User) error {
	db, err := DB()
	if err != nil {
		return err
	}
	if _, err := db.Exec(r.Context(), `DELETE FROM stock_items WHERE id=$1 AND user_id=$2`, r.PathValue("id"), u.ID); err != nil {
		return err
	}
	return writeJSON(w, map[string]bool{"ok": true})
}

func handleStockAdvice(w http.ResponseWriter, r *http.Request, u *User) error {
	a, err := stockAdvice(r.Context(), u.ID, u.Lang)
	if err != nil {
		return err
	}
	return writeJSON(w, a)
}

func handleOverview(w http.ResponseWriter, r *http.Request, u *User) error {
	o, err := buildOverview(r.Context(), u.ID, u.Lang)
	if err != nil {
		return err
	}
	return writeJSON(w, o)
}

// supplierFor names a plausible local supplier so purchase entries read like a real khata.
func supplierFor(phone, category string) string {
	switch phone {
	case "9000000001":
		if category == "fuel" {
			return "HP Gas agency, Governorpet"
		}
		if category == "fresh" {
			return "Rythu Bazaar, Patamata"
		}
		return "Sri Lakshmi Ganapathi Traders"
	case "9000000002":
		switch category {
		case "dairy":
			return "Amul distributor (Ravindrapuri)"
		case "beverages":
			return "Shree Shyam Agencies"
		}
		return "Vishweshwarganj mandi wholesaler"
	default:
		if category == "dairy" {
			return "Sangam Dairy milk agent"
		}
		return "Thullur kirana wholesale"
	}
}

func countable(unit string) bool {
	switch unit {
	case "plate", "pack", "pcs", "cup", "bar", "bottle", "packet":
		return true
	}
	return false
}

func restockNow(status string) bool {
	return status == "out" || status == "critical" || status == "low"
}

// restockWhy is a computed, localized reason used when the guard adds an item the model missed.
func restockWhy(lang string, s ItemStats) string {
	rate := fmt.Sprintf("%.1f %s", s.Rate7, s.Unit)
	days := fmt.Sprintf("%.1f", derefOr(s.DaysLeft, 0))
	out := s.Status == "out"
	switch lang {
	case "hi":
		if out {
			return "स्टॉक ख़त्म है; रोज़ करीब " + rate + " बिकता है।"
		}
		return "सिर्फ़ " + days + " दिन का स्टॉक बचा है; रोज़ करीब " + rate + " बिकता है।"
	case "te":
		if out {
			return "సరుకు అయిపోయింది; రోజుకు సుమారు " + rate + " అవసరం."
		}
		return "కేవలం " + days + " రోజుల సరుకు మిగిలింది; రోజుకు సుమారు " + rate + " అవసరం."
	case "ta":
		if out {
			return "இருப்பு தீர்ந்தது; தினமும் சுமார் " + rate + " தேவை."
		}
		return days + " நாள் இருப்பு மட்டுமே உள்ளது; தினமும் சுமார் " + rate + " தேவை."
	}
	if out {
		return "Out of stock; you sell about " + rate + " a day."
	}
	return "Only " + days + " days of stock left at about " + rate + " a day."
}

// groundAdvice keeps only real items, replaces model quantities with computed ones, and
// guarantees items about to run out are in buy_more (never buy_less).
func groundAdvice(res *StockAdvice, items []ItemStats, lang string) {
	byName := map[string]ItemStats{}
	for _, s := range items {
		byName[strings.ToLower(s.Name)] = s
	}
	// Ground every line in a real item and replace quantities with computed ones.
	ground := func(lines []AdviceLine, more bool) []AdviceLine {
		out := []AdviceLine{}
		seen := map[int64]bool{}
		for _, l := range lines {
			s, ok := byName[strings.ToLower(strings.TrimSpace(l.Item))]
			if !ok || seen[s.ID] {
				continue
			}
			seen[s.ID] = true
			l.Item, l.ItemID, l.Unit = s.Name, s.ID, s.Unit
			switch {
			case s.Kind == "dish":
				l.Qty = 0 // demand signal only
			case more:
				l.Qty = math.Max(s.ReorderQty, 1)
			default:
				// next purchase sized to one week of current demand, never more than the usual buy
				l.Qty = math.Max(0, math.Ceil(s.Rate7*7-s.Qty))
				if s.TypicalBuy > 0 {
					l.Qty = math.Min(l.Qty, s.TypicalBuy)
				}
			}
			out = append(out, l)
		}
		return out
	}
	res.BuyMore, res.BuyLess = ground(res.BuyMore, true), ground(res.BuyLess, false)
	// Guard: an empty shelf is never a reason to buy less; every alert item ends up in buy_more.
	inMore := map[int64]bool{}
	for _, l := range res.BuyMore {
		inMore[l.ItemID] = true
	}
	less := res.BuyLess[:0]
	for _, l := range res.BuyLess {
		if !restockNow(byName[strings.ToLower(l.Item)].Status) && !inMore[l.ItemID] {
			less = append(less, l)
		}
	}
	res.BuyLess = less
	for _, it := range items {
		if restockNow(it.Status) && !inMore[it.ID] {
			res.BuyMore = append(res.BuyMore, AdviceLine{Item: it.Name, ItemID: it.ID, Unit: it.Unit, Qty: math.Max(it.ReorderQty, 1), Urgency: "today", Why: restockWhy(lang, it)})
		}
	}
	keep := []string{}
	for _, k := range res.KeepSteady {
		if s, ok := byName[strings.ToLower(strings.TrimSpace(k))]; ok {
			keep = append(keep, s.Name)
		}
	}
	res.KeepSteady = keep
}
