package app

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type LedgerEntry struct {
	ID          int64   `json:"id"`
	Date        string  `json:"date"`
	Kind        string  `json:"kind"`
	AmountPaise int64   `json:"amount_paise"`
	Note        string  `json:"note"`
	Party       string  `json:"party"`
	Source      string  `json:"source"`
	Item        string  `json:"item"` // matched stock item name, optional
	Qty         float64 `json:"qty"`
}

type RecordsSummary struct {
	Days          int              `json:"days"`
	SalesPaise    int64            `json:"sales_paise"`
	ExpensePaise  int64            `json:"expense_paise"`
	ProfitPaise   int64            `json:"profit_paise"`
	AvgDailySale  int64            `json:"avg_daily_sale_paise"`
	UdhaarOut     int64            `json:"udhaar_outstanding_paise"`
	Daily         []map[string]any `json:"daily"`
	TopExpenses   []map[string]any `json:"top_expenses"`
	UdhaarParties []map[string]any `json:"udhaar_parties"`
}

func recordsSummary(ctx context.Context, uid int64, days int) (*RecordsSummary, error) {
	db, err := DB()
	if err != nil {
		return nil, err
	}
	s := &RecordsSummary{Days: days, Daily: []map[string]any{}, TopExpenses: []map[string]any{}, UdhaarParties: []map[string]any{}}
	since := time.Now().AddDate(0, 0, -days)
	var activeDays int64
	if err := db.QueryRow(ctx, `SELECT
		COALESCE(SUM(amount_paise) FILTER (WHERE kind='sale'),0),
		COALESCE(SUM(amount_paise) FILTER (WHERE kind='expense'),0),
		COUNT(DISTINCT entry_date) FILTER (WHERE kind='sale')
		FROM ledger_entries WHERE user_id=$1 AND entry_date >= $2`, uid, since).Scan(&s.SalesPaise, &s.ExpensePaise, &activeDays); err != nil {
		return nil, err
	}
	s.ProfitPaise = s.SalesPaise - s.ExpensePaise
	if activeDays > 0 {
		s.AvgDailySale = s.SalesPaise / activeDays
	}
	rows, err := db.Query(ctx, `SELECT d::date::text,
		COALESCE(SUM(amount_paise) FILTER (WHERE kind='sale'),0),
		COALESCE(SUM(amount_paise) FILTER (WHERE kind='expense'),0)
		FROM generate_series($2::date, CURRENT_DATE, '1 day') d
		LEFT JOIN ledger_entries e ON e.entry_date = d::date AND e.user_id=$1
		GROUP BY d ORDER BY d`, uid, since)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var d string
		var sale, exp int64
		if err := rows.Scan(&d, &sale, &exp); err != nil {
			rows.Close()
			return nil, err
		}
		s.Daily = append(s.Daily, map[string]any{"date": d, "sale": sale, "expense": exp})
	}
	rows.Close()
	rows, err = db.Query(ctx, `SELECT note, SUM(amount_paise) t FROM ledger_entries WHERE user_id=$1 AND kind='expense' AND entry_date >= $2
		GROUP BY note ORDER BY t DESC LIMIT 5`, uid, since)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var note string
		var t int64
		if err := rows.Scan(&note, &t); err != nil {
			rows.Close()
			return nil, err
		}
		s.TopExpenses = append(s.TopExpenses, map[string]any{"note": note, "amount_paise": t})
	}
	rows.Close()
	// udhaar outstanding across all time: given (they owe me) minus received back from the same party
	rows, err = db.Query(ctx, `SELECT party, SUM(CASE WHEN kind='udhaar_given' THEN amount_paise ELSE -amount_paise END) bal, MAX(entry_date)::text
		FROM ledger_entries WHERE user_id=$1 AND kind IN ('udhaar_given','udhaar_received') AND party <> ''
		GROUP BY party HAVING SUM(CASE WHEN kind='udhaar_given' THEN amount_paise ELSE -amount_paise END) <> 0 ORDER BY bal DESC`, uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var party, last string
		var bal int64
		if err := rows.Scan(&party, &bal, &last); err != nil {
			return nil, err
		}
		if bal > 0 {
			s.UdhaarOut += bal
		}
		s.UdhaarParties = append(s.UdhaarParties, map[string]any{"party": party, "balance_paise": bal, "last_date": last})
	}
	return s, nil
}

func listEntries(ctx context.Context, uid int64, limit int) ([]LedgerEntry, error) {
	db, err := DB()
	if err != nil {
		return nil, err
	}
	rows, err := db.Query(ctx, `SELECT e.id, e.entry_date::text, e.kind, e.amount_paise, e.note, e.party, e.source, COALESCE(i.name, ''), COALESCE(e.qty, 0)
		FROM ledger_entries e LEFT JOIN stock_items i ON i.id = e.item_id
		WHERE e.user_id=$1 ORDER BY e.entry_date DESC, e.id DESC LIMIT $2`, uid, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []LedgerEntry{}
	for rows.Next() {
		var e LedgerEntry
		if err := rows.Scan(&e.ID, &e.Date, &e.Kind, &e.AmountPaise, &e.Note, &e.Party, &e.Source, &e.Item, &e.Qty); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, nil
}

var entryKinds = map[string]bool{"sale": true, "expense": true, "udhaar_given": true, "udhaar_received": true}

func insertEntries(ctx context.Context, uid int64, entries []LedgerEntry) ([]LedgerEntry, error) {
	db, err := DB()
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if !entryKinds[e.Kind] || e.AmountPaise <= 0 || e.AmountPaise > 1_00_00_000_00 {
			return nil, httpErr(400, "each entry needs a type and an amount above ₹0")
		}
		if _, err := time.Parse("2006-01-02", e.Date); err != nil {
			return nil, httpErr(400, "bad date "+e.Date)
		}
	}
	tx, err := db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	out := make([]LedgerEntry, 0, len(entries))
	for _, e := range entries {
		if e.Source == "" {
			e.Source = "text"
		}
		e.Note, e.Party = strings.TrimSpace(e.Note), strings.TrimSpace(e.Party)
		// A sale or purchase of a known stock item also moves stock, so run rates and alerts stay live.
		var itemID *int64
		if (e.Kind == "sale" || e.Kind == "expense") && e.Qty > 0 && e.Qty < 1e6 {
			it, err := findItem(ctx, uid, e.Item)
			if err != nil {
				return nil, err
			}
			if it != nil {
				move, delta := "sold", -e.Qty
				if e.Kind == "expense" {
					move, delta = "bought", e.Qty
				}
				if _, err := tx.Exec(ctx, `INSERT INTO stock_moves(user_id, item_id, move_date, kind, qty, amount_paise) VALUES($1,$2,$3,$4,$5,$6)`,
					uid, it.ID, e.Date, move, e.Qty, e.AmountPaise); err != nil {
					return nil, err
				}
				if it.Kind != "dish" {
					if _, err := tx.Exec(ctx, `UPDATE stock_items SET qty = GREATEST(0, qty + $1), updated_at = now() WHERE id = $2`, delta, it.ID); err != nil {
						return nil, err
					}
				}
				itemID, e.Item = &it.ID, it.Name
			}
		}
		if err := tx.QueryRow(ctx, `INSERT INTO ledger_entries(user_id, entry_date, kind, amount_paise, note, party, source, item_id, qty) VALUES($1,$2,$3,$4,$5,$6,$7,$8,NULLIF($9::float8, 0)) RETURNING id`,
			uid, e.Date, e.Kind, e.AmountPaise, e.Note, e.Party, e.Source, itemID, e.Qty).Scan(&e.ID); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, tx.Commit(ctx)
}

var entrySchema = obj(map[string]any{
	"entries": arr(obj(map[string]any{
		"date":         str("YYYY-MM-DD; use today's date unless the text says otherwise (e.g. 'yesterday')"),
		"kind":         enum("sale", "expense", "udhaar_given", "udhaar_received"),
		"amount_paise": integer("amount in paise (rupees × 100)"),
		"note":         str("short English description, e.g. '40 idlis' or 'rice 10kg'"),
		"party":        str("person/shop name for udhaar, else empty"),
		"item":         str("exact name from STOCK ITEMS if this entry is a sale or purchase of that item, else empty"),
		"qty":          map[string]any{"type": "number", "description": "units of that item (plates, kg, packs…), 0 if not stated"},
	})),
	"unclear": str("anything you could not understand, else empty"),
})

// parseEntries turns speech/text or a bill photo into proposed entries. Nothing is saved here.
func parseEntries(ctx context.Context, uid int64, text, imageDataURL, source string) (map[string]any, error) {
	today := time.Now().In(ist).Format("2006-01-02 (Monday)")
	names := itemNames(ctx, uid)
	sys := "You convert a small Indian merchant's spoken or written notes, or a photo of a handwritten bill/khata page, into ledger entries. " +
		"Today is " + today + ". Notes may be in Hindi, Telugu, Tamil, English or mixed. " +
		"'sale' = money earned from selling; 'expense' = money spent on stock, rent, gas, wages; 'udhaar_given' = goods/money given on credit that someone owes the merchant; " +
		"'udhaar_received' = credit repaid to the merchant. Never invent amounts: if an amount is missing, leave that item out and mention it in 'unclear'. " +
		"STOCK ITEMS: " + strings.Join(names, "; ") + ". When the note is about one of these (e.g. '40 idlis' → 'Idli (plate of 4)' with qty 40 plates only if it is clearly plates), set item and qty; if 40 idlis are pieces and the item is a plate of 4, qty is 10. If an item is not in STOCK ITEMS, ALWAYS still record the sale or expense entry with item='' and qty=0."
	var user any = text
	if imageDataURL != "" {
		user = []map[string]any{
			{"type": "text", "text": "Extract ledger entries from this bill / khata photo. " + text},
			{"type": "image_url", "image_url": map[string]any{"url": imageDataURL}},
		}
	}
	var res struct {
		Entries []LedgerEntry `json:"entries"`
		Unclear string        `json:"unclear"`
	}
	if err := chatJSON(ctx, "parse_entries", sys, user, entrySchema, &res); err != nil {
		return nil, err
	}
	valid := []LedgerEntry{}
	for _, e := range res.Entries {
		if entryKinds[e.Kind] && e.AmountPaise > 0 {
			e.Source = source
			valid = append(valid, e)
		}
	}
	return map[string]any{"entries": valid, "unclear": res.Unclear}, nil
}

var ist = time.FixedZone("IST", 5*3600+1800)

func handleRecords(w http.ResponseWriter, r *http.Request, u *User) error {
	days, _ := strconv.Atoi(r.URL.Query().Get("days"))
	if days <= 0 || days > 365 {
		days = 30
	}
	s, err := recordsSummary(r.Context(), u.ID, days)
	if err != nil {
		return err
	}
	entries, err := listEntries(r.Context(), u.ID, 60)
	if err != nil {
		return err
	}
	return writeJSON(w, map[string]any{"summary": s, "entries": entries})
}

func handleAddEntries(w http.ResponseWriter, r *http.Request, u *User) error {
	var in struct {
		Entries []LedgerEntry `json:"entries"`
	}
	if err := readJSON(r, &in); err != nil {
		return err
	}
	if len(in.Entries) == 0 || len(in.Entries) > 50 {
		return httpErr(400, "add between 1 and 50 entries")
	}
	out, err := insertEntries(r.Context(), u.ID, in.Entries)
	if err != nil {
		return err
	}
	return writeJSON(w, out)
}

func handleDeleteEntry(w http.ResponseWriter, r *http.Request, u *User) error {
	db, err := DB()
	if err != nil {
		return err
	}
	_, err = db.Exec(r.Context(), `DELETE FROM ledger_entries WHERE id=$1 AND user_id=$2`, r.PathValue("id"), u.ID)
	if err != nil {
		return err
	}
	return writeJSON(w, map[string]bool{"ok": true})
}

func handleParseEntries(w http.ResponseWriter, r *http.Request, u *User) error {
	var in struct {
		Text   string `json:"text"`
		Image  string `json:"image"` // data URL
		Source string `json:"source"`
	}
	if err := readJSON(r, &in); err != nil {
		return err
	}
	if strings.TrimSpace(in.Text) == "" && in.Image == "" {
		return httpErr(400, "say or type what you sold or spent")
	}
	if in.Image != "" && !strings.HasPrefix(in.Image, "data:image/") {
		return httpErr(400, "image must be a photo")
	}
	if in.Source != "voice" && in.Source != "photo" {
		in.Source = "text"
	}
	out, err := parseEntries(r.Context(), u.ID, in.Text, in.Image, in.Source)
	if err != nil {
		return err
	}
	return writeJSON(w, out)
}
