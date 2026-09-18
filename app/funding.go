package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/jackc/pgx/v5"
)

// ---------- Blueprint ----------

var moneyItem = obj(map[string]any{"item": str(""), "amount_inr": integer("")})

var blueprintSchema = obj(map[string]any{
	"business_name": str(""),
	"one_liner":     str("one sentence: who you serve and what you sell"),
	"problem":       str("customer problem / local need"),
	"offering":      str("products/services and price points"),
	"customers":     str("who buys, where, how many per day"),
	"stage":         enum("idea", "running", "growing"),
	"team":          str(""),
	"revenue_model": str(""),
	"funding_need": obj(map[string]any{
		"amount_inr":   integer("total needed; 0 if unknown"),
		"purpose":      str(""),
		"use_of_funds": arr(moneyItem),
	}),
	"milestones":     arr(obj(map[string]any{"month": integer("1-12"), "goal": str("")})),
	"vision":         str("one sentence picture of the business 12 months from now if the plan works"),
	"growth_drivers": arr(str("2-4 concrete drivers from the data, e.g. a rising item and its trend")),
	"strengths":      arr(str("")),
	"risks":          arr(obj(map[string]any{"risk": str(""), "mitigation": str("")})),
	"missing_info":   arr(str("facts a grant officer would ask for that are not known yet")),
})

type Blueprint struct {
	ID        int64          `json:"id"`
	Data      map[string]any `json:"data"`
	CreatedAt string         `json:"created_at"`
}

// businessContext gathers everything Finro knows, with ledger-computed numbers the model must not change.
func businessContext(ctx context.Context, uid int64) (string, map[string]any, error) {
	profile, mems, err := loadProfile(ctx, uid)
	if err != nil {
		return "", nil, err
	}
	s, err := recordsSummary(ctx, uid, 30)
	if err != nil {
		return "", nil, err
	}
	fin := map[string]any{
		"basis":                  fmt.Sprintf("Finro Khata records, last %d days", s.Days),
		"sales_last_30d_inr":     s.SalesPaise / 100,
		"expenses_last_30d_inr":  s.ExpensePaise / 100,
		"profit_last_30d_inr":    s.ProfitPaise / 100,
		"avg_daily_sale_inr":     s.AvgDailySale / 100,
		"udhaar_outstanding_inr": s.UdhaarOut / 100,
		"top_expenses":           s.TopExpenses,
	}
	facts := make([]string, len(mems))
	for i, m := range mems {
		facts[i] = m.Fact
	}
	return mustJSON(map[string]any{"profile": profile, "remembered_facts": facts, "financials_verified": fin, "stock_and_run_rates": stockBrief(ctx, uid)}), fin, nil
}

func buildBlueprint(ctx context.Context, uid int64, extra string) (*Blueprint, error) {
	bctx, fin, err := businessContext(ctx, uid)
	if err != nil {
		return nil, err
	}
	sys := "You are Finro, writing a one-page business blueprint for an Indian micro-entrepreneur that a bank officer or scheme evaluator would accept. " +
		"Use ONLY facts in the context. financials_verified come from the merchant's own daily records — do not restate them differently. " +
		"Where something is unknown, do not invent it: list it under missing_info. Keep language simple and concrete (₹ amounts, places, counts). " +
		"Use of funds must add up to funding_need.amount_inr when an amount is known."
	var data map[string]any
	if err := chatJSON(ctx, "blueprint", sys, "CONTEXT:\n"+bctx+"\n\nLATEST CONVERSATION NOTES:\n"+extra, blueprintSchema, &data); err != nil {
		return nil, err
	}
	data["financials"] = fin // authoritative numbers, never model-generated
	db, err := DB()
	if err != nil {
		return nil, err
	}
	b := &Blueprint{Data: data}
	err = db.QueryRow(ctx, `INSERT INTO blueprints(user_id, data) VALUES($1,$2) RETURNING id, to_char(created_at,'YYYY-MM-DD HH24:MI')`, uid, data).Scan(&b.ID, &b.CreatedAt)
	return b, err
}

func latestBlueprint(ctx context.Context, uid int64) (*Blueprint, error) {
	db, err := DB()
	if err != nil {
		return nil, err
	}
	b := &Blueprint{}
	err = db.QueryRow(ctx, `SELECT id, data, to_char(created_at,'YYYY-MM-DD HH24:MI') FROM blueprints WHERE user_id=$1 ORDER BY id DESC LIMIT 1`, uid).Scan(&b.ID, &b.Data, &b.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return b, err
}

// ---------- Matching ----------

type Criterion struct {
	Rule        string `json:"rule"`
	Status      string `json:"status"` // pass | fail | unknown
	Evidence    string `json:"evidence"`
	SourceQuote string `json:"source_quote"`
}

type Fit struct {
	GrantID   int64       `json:"grant_id"`
	Score     int         `json:"score"`
	Verdict   string      `json:"verdict"` // strong | possible | not_a_fit
	Why       string      `json:"why"`
	Criteria  []Criterion `json:"criteria"`
	NextSteps []string    `json:"next_steps"`
	Grant     *Grant      `json:"grant,omitempty"`
}

var fitSchema = obj(map[string]any{
	"fits": arr(obj(map[string]any{
		"grant_id": integer(""),
		"score":    integer("0-100 eligibility and usefulness"),
		"verdict":  enum("strong", "possible", "not_a_fit"),
		"why":      str("one plain sentence"),
		"criteria": arr(obj(map[string]any{
			"rule":         str("an eligibility rule of THIS scheme"),
			"status":       enum("pass", "fail", "unknown"),
			"evidence":     str("fact from the blueprint/profile, or what is missing"),
			"source_quote": str("verbatim words from the scheme text that state this rule, else empty"),
		})),
		"next_steps": arr(str("")),
	})),
})

// rankForProfile keeps open grants allowed in the user's state and orders them by tag overlap with
// the user's domain, needs and category, capped at 12.
// ponytail: tag-overlap heuristic, swap for embeddings if the catalogue grows past a few hundred schemes.
func rankForProfile(grants []Grant, profile map[string]any, state string) []Grant {
	want := map[string]bool{}
	for _, k := range []string{"business_domain", "business_stage", "social_category", "gender"} {
		if v, ok := profile[k].(string); ok {
			want[strings.ToLower(v)] = true
		}
	}
	if needs, ok := profile["needs"].([]any); ok {
		for _, n := range needs {
			want[strings.ToLower(fmt.Sprint(n))] = true
		}
	}
	if want["female"] {
		want["women"] = true
	}
	if want["sc"] || want["st"] {
		want["sc-st"] = true
	}
	type scored struct {
		g Grant
		s int
	}
	var list []scored
	for _, g := range grants {
		if g.Status == "closed" || !stateAllowed(g.Tags, state) {
			continue
		}
		s := 0
		for _, t := range g.Tags {
			if want[strings.ToLower(t)] {
				s += 2
			}
			if t == "any-gender" || strings.HasPrefix(t, "state:") {
				s++
			}
		}
		list = append(list, scored{g, s})
	}
	slices.SortStableFunc(list, func(a, b scored) int { return b.s - a.s })
	out := []Grant{}
	for _, x := range list[:min(len(list), 12)] {
		out = append(out, x.g)
	}
	return out
}

// scoreFits evaluates the most relevant open grants against a blueprint, one rule at a time.
func scoreFits(ctx context.Context, uid int64, b *Blueprint) ([]Fit, error) {
	grants, err := listGrants(ctx)
	if err != nil {
		return nil, err
	}
	profile, _, _ := loadProfile(ctx, uid)
	state, _ := profile["state"].(string)
	byID := map[int64]Grant{}
	var cands []map[string]any
	for _, g := range rankForProfile(grants, profile, state) {
		byID[g.ID] = g
		cands = append(cands, map[string]any{"grant_id": g.ID, "title": g.Title, "source": g.Source, "summary": g.Summary, "eligibility": g.Eligibility, "amount": g.AmountText, "tags": g.Tags})
	}
	if len(cands) == 0 {
		return nil, httpErr(409, "no open funding sources right now — refresh Live Funding first")
	}
	sys := "You are a strict scheme eligibility checker for Indian government funding. Return exactly ONE entry per grant given. For each, list 2-5 concrete eligibility rules from its text and check each against the applicant. " +
		"Never assume: if the applicant's data does not show it, status is 'unknown'. A single clear 'fail' on a mandatory rule means verdict 'not_a_fit' and score below 40. " +
		"Do not broad-match: a street-vendor scheme is not a fit for a shop owner, an artisan scheme is not a fit for a trader, a biotech research call is not a fit for a food business, " +
		"and schemes for 'new units only' fail for an existing business unless the plan is a genuinely new unit. 'strong' needs score ≥ 70 with no fails."
	// Small parallel batches: faster than one big call and the model reliably answers for every grant.
	var res struct {
		Fits []Fit `json:"fits"`
	}
	var mu sync.Mutex
	var wg sync.WaitGroup
	var firstErr error
	for i := 0; i < len(cands); i += 3 {
		batch := cands[i:min(i+3, len(cands))]
		wg.Go(func() {
			var part struct {
				Fits []Fit `json:"fits"`
			}
			user := mustJSON(map[string]any{"applicant_blueprint": b.Data, "applicant_profile": profile, "grants": batch})
			err := chatJSON(ctx, "grant_fit", sys, user, fitSchema, &part)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				firstErr = err
				return
			}
			res.Fits = append(res.Fits, part.Fits...)
		})
	}
	wg.Wait()
	if len(res.Fits) == 0 && firstErr != nil {
		return nil, firstErr
	}
	db, err := DB()
	if err != nil {
		return nil, err
	}
	out := []Fit{}
	for _, f := range res.Fits {
		g, ok := byID[f.GrantID]
		if !ok {
			continue // model referenced a grant that does not exist
		}
		srcText := g.Title + " " + g.Summary + " " + g.Eligibility + " " + g.AmountText
		for i := range f.Criteria {
			if !verifyQuote(srcText, f.Criteria[i].SourceQuote) {
				f.Criteria[i].SourceQuote = ""
			}
		}
		f.Score = min(max(f.Score, 0), 100)
		if slices.ContainsFunc(f.Criteria, func(c Criterion) bool { return c.Status == "fail" }) && f.Verdict == "strong" {
			f.Verdict = "possible"
		}
		if _, err := db.Exec(ctx, `INSERT INTO grant_fits(blueprint_id, grant_id, score, verdict, result) VALUES($1,$2,$3,$4,$5)
			ON CONFLICT (blueprint_id, grant_id) DO UPDATE SET score=EXCLUDED.score, verdict=EXCLUDED.verdict, result=EXCLUDED.result, created_at=now()`,
			b.ID, f.GrantID, f.Score, f.Verdict, f); err != nil {
			return nil, err
		}
		gc := g
		f.Grant = &gc
		out = append(out, f)
	}
	slices.SortFunc(out, func(a, b Fit) int { return b.Score - a.Score })
	return out, nil
}

func loadFits(ctx context.Context, blueprintID int64) ([]Fit, error) {
	db, err := DB()
	if err != nil {
		return nil, err
	}
	rows, err := db.Query(ctx, `SELECT result FROM grant_fits WHERE blueprint_id=$1 ORDER BY score DESC`, blueprintID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Fit{}
	for rows.Next() {
		var f Fit
		if err := rows.Scan(&f); err != nil {
			return nil, err
		}
		if g, err := getGrant(ctx, f.GrantID); err == nil {
			f.Grant = &g
		}
		out = append(out, f)
	}
	return out, nil
}

// ---------- Drafts ----------

var draftSchema = obj(map[string]any{
	"title": str(""),
	"sections": arr(obj(map[string]any{
		"heading": str(""),
		"body":    str("filled from blueprint; unknown facts as [NEEDED: what]"),
	})),
	"documents":  arr(obj(map[string]any{"name": str(""), "have": map[string]any{"type": "boolean"}})),
	"cover_note": str("3-4 sentence note to the officer"),
})

func createDraft(ctx context.Context, uid, grantID int64) (map[string]any, error) {
	g, err := getGrant(ctx, grantID)
	if err != nil {
		return nil, httpErr(404, "funding scheme not found")
	}
	if g.Status == "closed" {
		return nil, httpErr(409, "this call closed on "+*g.Deadline+" — pick an open one")
	}
	b, err := latestBlueprint(ctx, uid)
	if err != nil {
		return nil, err
	}
	if b == nil {
		if b, err = buildBlueprint(ctx, uid, ""); err != nil {
			return nil, err
		}
	}
	profile, _, _ := loadProfile(ctx, uid)
	sys := "Draft a grant/loan application for the scheme below using ONLY the applicant's blueprint and profile. " +
		"Sections should follow what this kind of scheme asks: applicant details, business description, current financials (use blueprint.financials exactly), " +
		"funding requirement and use of funds, expected impact (jobs, income), repayment/ sustainability plan, declaration. " +
		"Every fact not in the data must be written as [NEEDED: description] — never invent Aadhaar numbers, account numbers, dates, or amounts. " +
		"documents: list the documents this scheme typically requires; have=true only if profile.documents lists it. Do not state any deadline."
	var content map[string]any
	user := mustJSON(map[string]any{"scheme": g, "blueprint": b.Data, "profile": profile})
	if err := chatJSON(ctx, "draft", sys, user, draftSchema, &content); err != nil {
		return nil, err
	}
	content["grant"] = map[string]any{"id": g.ID, "title": g.Title, "source": g.Source, "source_url": g.SourceURL, "deadline": g.Deadline, "deadline_raw": g.DeadlineRaw, "fetched_at": g.FetchedAt}
	db, err := DB()
	if err != nil {
		return nil, err
	}
	var id int64
	if err := db.QueryRow(ctx, `INSERT INTO drafts(user_id, grant_id, blueprint_id, content) VALUES($1,$2,$3,$4) RETURNING id`, uid, grantID, b.ID, content).Scan(&id); err != nil {
		return nil, err
	}
	content["id"] = id
	content["needed_count"] = strings.Count(mustJSON(content["sections"]), "[NEEDED")
	return content, nil
}

// ---------- Handlers ----------

func handleGetBlueprint(w http.ResponseWriter, r *http.Request, u *User) error {
	b, err := latestBlueprint(r.Context(), u.ID)
	if err != nil {
		return err
	}
	return writeJSON(w, b)
}

func handleBuildBlueprint(w http.ResponseWriter, r *http.Request, u *User) error {
	var in struct{ Notes string }
	readJSON(r, &in)
	b, err := buildBlueprint(r.Context(), u.ID, truncate(in.Notes, 4000))
	if err != nil {
		return err
	}
	return writeJSON(w, b)
}

func handleMatches(w http.ResponseWriter, r *http.Request, u *User) error {
	b, err := latestBlueprint(r.Context(), u.ID)
	if err != nil || b == nil {
		return writeJSON(w, map[string]any{"blueprint": nil, "fits": []Fit{}})
	}
	fits, err := loadFits(r.Context(), b.ID)
	if err != nil {
		return err
	}
	return writeJSON(w, map[string]any{"blueprint": b, "fits": fits})
}

func handleRunMatches(w http.ResponseWriter, r *http.Request, u *User) error {
	b, err := latestBlueprint(r.Context(), u.ID)
	if err != nil {
		return err
	}
	if b == nil {
		if b, err = buildBlueprint(r.Context(), u.ID, ""); err != nil {
			return err
		}
	}
	fits, err := scoreFits(r.Context(), u.ID, b)
	if err != nil {
		return err
	}
	return writeJSON(w, map[string]any{"blueprint": b, "fits": fits})
}

func handleListDrafts(w http.ResponseWriter, r *http.Request, u *User) error {
	db, err := DB()
	if err != nil {
		return err
	}
	rows, err := db.Query(r.Context(), `SELECT id, status, content, to_char(updated_at,'YYYY-MM-DD HH24:MI') FROM drafts WHERE user_id=$1 ORDER BY id DESC`, u.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id int64
		var status, updated string
		var content map[string]any
		if err := rows.Scan(&id, &status, &content, &updated); err != nil {
			return err
		}
		content["id"], content["status"], content["updated_at"] = id, status, updated
		content["needed_count"] = strings.Count(mustJSON(content["sections"]), "[NEEDED")
		out = append(out, content)
	}
	return writeJSON(w, out)
}

func handleCreateDraft(w http.ResponseWriter, r *http.Request, u *User) error {
	var in struct {
		GrantID int64 `json:"grant_id"`
	}
	if err := readJSON(r, &in); err != nil {
		return err
	}
	d, err := createDraft(r.Context(), u.ID, in.GrantID)
	if err != nil {
		return err
	}
	return writeJSON(w, d)
}

func handleUpdateDraft(w http.ResponseWriter, r *http.Request, u *User) error {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	var in struct {
		Sections []map[string]string `json:"sections"`
		Status   string              `json:"status"`
	}
	if err := readJSON(r, &in); err != nil {
		return err
	}
	if in.Status != "" && in.Status != "draft" && in.Status != "ready" && in.Status != "submitted" {
		return httpErr(400, "bad status")
	}
	db, err := DB()
	if err != nil {
		return err
	}
	tag, err := db.Exec(r.Context(), `UPDATE drafts SET
		content = CASE WHEN $3::jsonb IS NULL THEN content ELSE jsonb_set(content, '{sections}', $3::jsonb) END,
		status = COALESCE(NULLIF($4,''), status), updated_at=now() WHERE id=$1 AND user_id=$2`, id, u.ID, in.Sections, in.Status)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return httpErr(404, "draft not found")
	}
	return writeJSON(w, map[string]bool{"ok": true})
}
