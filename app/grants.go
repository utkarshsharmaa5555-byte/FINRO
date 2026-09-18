package app

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type Grant struct {
	ID          int64    `json:"id"`
	Source      string   `json:"source"`
	SourceURL   string   `json:"source_url"`
	Title       string   `json:"title"`
	Summary     string   `json:"summary"`
	AmountText  string   `json:"amount_text"`
	Deadline    *string  `json:"deadline"`
	DeadlineRaw string   `json:"deadline_raw"`
	Eligibility string   `json:"eligibility"`
	Tags        []string `json:"tags"`
	FetchedAt   string   `json:"fetched_at"`
	Status      string   `json:"status"` // open | closed | rolling
	LinkOK      *bool    `json:"link_ok"`
}

var httpClient = &http.Client{Timeout: 20 * time.Second}

func fetch(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; FinroBot/1.0; +https://munshi-ochre.vercel.app)")
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("%s returned %d", url, resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 4<<20))
}

// ---- Source 1: BIRAC calls for proposals (structured HTML, deterministic parse) ----

const biracURL = "https://birac.nic.in/cfp.php"

var biracDeadlineRe = regexp.MustCompile(`(\d{1,2}-[A-Za-z]{3}-\d{4})`)

func parseBIRAC(html []byte, now time.Time) ([]Grant, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(html))
	if err != nil {
		return nil, err
	}
	var out []Grant
	doc.Find("table#current tbody tr, table#previous tbody tr").Each(func(i int, tr *goquery.Selection) {
		a := tr.Find("a").First()
		href, ok := a.Attr("href")
		title := strings.Join(strings.Fields(a.Text()), " ")
		if !ok || title == "" || len(out) >= 15 {
			return
		}
		g := Grant{Source: "BIRAC", SourceURL: "https://birac.nic.in/" + strings.TrimPrefix(href, "/"), Title: strings.TrimSuffix(title, "..."),
			Tags: []string{"startup", "biotech", "innovation", "research"}, Summary: "Call for proposals from the Biotechnology Industry Research Assistance Council (Dept. of Biotechnology, Govt. of India)."}
		tr.Find("small").Each(func(_ int, s *goquery.Selection) {
			t := strings.Join(strings.Fields(s.Text()), " ")
			if strings.Contains(strings.ToLower(t), "last date") {
				g.DeadlineRaw = t
				if m := biracDeadlineRe.FindString(t); m != "" {
					if d, err := time.Parse("2-Jan-2006", m); err == nil {
						ds := d.Format("2006-01-02")
						g.Deadline = &ds
					}
				}
			} else if t != "" {
				g.Eligibility = t
			}
		})
		out = append(out, g)
	})
	if len(out) == 0 {
		return nil, fmt.Errorf("BIRAC: no calls found — page layout may have changed")
	}
	return out, nil
}

// ---- Sources 2 & 3: official scheme portals (LLM extraction, quote-verified) ----

type schemePage struct {
	Source, URL string
	Tags        []string
}

var schemePages = []schemePage{
	{"Scheme portals", "https://pmsvanidhi.mohua.gov.in/", []string{"street-vendor", "micro-loan", "working-capital", "urban"}},
	{"Scheme portals", "https://www.mudra.org.in/", []string{"micro-loan", "small-business", "shopkeeper", "women"}},
	{"Scheme portals", "https://www.standupmitra.in/Home/SUISchemes", []string{"loan", "women", "sc-st", "greenfield"}},
	{"Scheme portals", "https://www.cgtmse.in/", []string{"collateral-free", "credit-guarantee", "msme"}},
	{"Atal Innovation Mission", "https://aim.gov.in/", []string{"startup", "incubation", "innovation", "grant"}},
}

var extractSchema = obj(map[string]any{
	"items": arr(obj(map[string]any{
		"title":          str("official scheme/programme/call name exactly as on the page"),
		"summary":        str("1-2 sentence plain-English summary of what the applicant gets"),
		"amount_text":    str("benefit amount as written, e.g. 'up to ₹10 lakh', else empty"),
		"amount_quote":   str("verbatim substring of PAGE TEXT containing the amount, else empty"),
		"deadline_iso":   str("YYYY-MM-DD only if the page states an application deadline, else empty"),
		"deadline_quote": str("verbatim substring of PAGE TEXT stating that deadline, else empty"),
		"eligibility":    str("who can apply, as stated on the page (short)"),
	})),
})

var spaceRe = regexp.MustCompile(`\s+`)

func normText(s string) string { return strings.ToLower(spaceRe.ReplaceAllString(s, " ")) }

// pageText extracts visible text; scripts/styles/nav removed.
func pageText(html []byte) (string, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(html))
	if err != nil {
		return "", err
	}
	doc.Find("script, style, noscript, svg, header nav, footer").Remove()
	return strings.TrimSpace(spaceRe.ReplaceAllString(doc.Find("body").Text(), " ")), nil
}

// verifyQuote keeps a claimed fact only if its supporting quote literally appears on the page.
func verifyQuote(page, quote string) bool {
	q := normText(strings.TrimSpace(quote))
	return len(q) >= 4 && strings.Contains(normText(page), q)
}

func extractSchemes(ctx context.Context, sp schemePage) ([]Grant, error) {
	html, err := fetch(ctx, sp.URL)
	if err != nil {
		return nil, err
	}
	text, err := pageText(html)
	if err != nil {
		return nil, err
	}
	if len(text) < 200 {
		return nil, fmt.Errorf("%s: page had no readable text", sp.URL)
	}
	text = truncate(text, 9000)
	var res struct {
		Items []struct {
			Title, Summary, Eligibility string
			AmountText                  string `json:"amount_text"`
			AmountQuote                 string `json:"amount_quote"`
			DeadlineISO                 string `json:"deadline_iso"`
			DeadlineQuote               string `json:"deadline_quote"`
		} `json:"items"`
	}
	sys := "Extract funding/loan/grant schemes that a small business, street vendor, or early-stage founder can apply to, from an official Indian government web page. " +
		"Return at most 4 items. Only use facts present in PAGE TEXT. Quotes must be copied character-for-character from PAGE TEXT. " +
		"Never guess deadlines: most schemes are open year-round, so leave deadline fields empty unless a specific last date is written."
	if err := chatJSON(ctx, "extract_schemes", sys, "SOURCE: "+sp.URL+"\n\nPAGE TEXT:\n"+text, extractSchema, &res); err != nil {
		return nil, err
	}
	var out []Grant
	for _, it := range res.Items {
		if strings.TrimSpace(it.Title) == "" {
			continue
		}
		g := Grant{Source: sp.Source, SourceURL: sp.URL + "#" + slugify(it.Title), Title: it.Title, Summary: it.Summary, Eligibility: it.Eligibility, Tags: sp.Tags}
		if verifyQuote(text, it.AmountQuote) {
			g.AmountText = it.AmountText
		}
		if d, err := time.Parse("2006-01-02", it.DeadlineISO); err == nil && verifyQuote(text, it.DeadlineQuote) {
			ds := d.Format("2006-01-02")
			g.Deadline, g.DeadlineRaw = &ds, it.DeadlineQuote
		}
		out = append(out, g)
	}
	return out, nil
}

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(s string) string {
	return strings.Trim(slugRe.ReplaceAllString(strings.ToLower(s), "-"), "-")
}

func upsertGrants(ctx context.Context, gs []Grant) error {
	db, err := DB()
	if err != nil {
		return err
	}
	for _, g := range gs {
		if g.Tags == nil {
			g.Tags = []string{}
		}
		if _, err := db.Exec(ctx, `INSERT INTO grants(source, source_url, title, summary, amount_text, deadline, deadline_raw, eligibility, tags, fetched_at)
			VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9, now())
			ON CONFLICT (source_url) DO UPDATE SET title=EXCLUDED.title, summary=EXCLUDED.summary, amount_text=EXCLUDED.amount_text,
			deadline=EXCLUDED.deadline, deadline_raw=EXCLUDED.deadline_raw, eligibility=EXCLUDED.eligibility, tags=EXCLUDED.tags, fetched_at=now()`,
			g.Source, g.SourceURL, g.Title, g.Summary, g.AmountText, g.Deadline, g.DeadlineRaw, g.Eligibility, g.Tags); err != nil {
			return err
		}
	}
	return nil
}

func logRun(ctx context.Context, source string, start time.Time, count int, err error) {
	db, dberr := DB()
	if dberr != nil {
		return
	}
	msg := ""
	if err != nil {
		msg = truncate(err.Error(), 500)
	}
	db.Exec(ctx, `INSERT INTO scraper_runs(source, ok, count, error, ms) VALUES($1,$2,$3,$4,$5)`, source, err == nil, count, msg, time.Since(start).Milliseconds())
}

// RefreshAll scrapes every source concurrently; a failing source keeps its last good rows.
func RefreshAll(ctx context.Context) map[string]any {
	var wg sync.WaitGroup
	var mu sync.Mutex
	report := map[string]any{}
	record := func(name string, n int, err error) {
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			report[name] = "error: " + err.Error()
		} else {
			report[name] = n
		}
	}
	wg.Go(func() {
		start := time.Now()
		html, err := fetch(ctx, biracURL)
		var gs []Grant
		if err == nil {
			gs, err = parseBIRAC(html, time.Now())
		}
		if err == nil {
			err = upsertGrants(ctx, gs)
		}
		logRun(ctx, biracURL, start, len(gs), err)
		record("BIRAC", len(gs), err)
	})
	for _, sp := range schemePages {
		wg.Go(func() {
			start := time.Now()
			gs, err := extractSchemes(ctx, sp)
			if err == nil {
				err = upsertGrants(ctx, gs)
			}
			logRun(ctx, sp.URL, start, len(gs), err)
			record(sp.URL, len(gs), err)
		})
	}
	wg.Go(func() {
		start := time.Now()
		n, err := checkDirectoryLinks(ctx)
		logRun(ctx, "scheme-directory-links", start, n, err)
		record("scheme-directory-links", n, err)
	})
	wg.Go(func() {
		start := time.Now()
		n, err := refreshScamNews(ctx)
		logRun(ctx, "scam-news", start, n, err)
		record("scam-news", n, err)
	})
	wg.Wait()
	return report
}

func grantStatus(deadline *string, now time.Time) string {
	if deadline == nil {
		return "rolling"
	}
	if *deadline >= now.In(ist).Format("2006-01-02") {
		return "open"
	}
	return "closed"
}

const grantCols = `id, source, source_url, title, summary, amount_text, to_char(deadline,'YYYY-MM-DD'), deadline_raw, eligibility, tags, to_char(fetched_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"'), link_ok`

func scanGrant(row interface{ Scan(...any) error }) (Grant, error) {
	var g Grant
	err := row.Scan(&g.ID, &g.Source, &g.SourceURL, &g.Title, &g.Summary, &g.AmountText, &g.Deadline, &g.DeadlineRaw, &g.Eligibility, &g.Tags, &g.FetchedAt, &g.LinkOK)
	g.Status = grantStatus(g.Deadline, time.Now())
	return g, err
}

func listGrants(ctx context.Context) ([]Grant, error) {
	db, err := DB()
	if err != nil {
		return nil, err
	}
	rows, err := db.Query(ctx, `SELECT `+grantCols+` FROM grants ORDER BY (deadline IS NULL OR deadline >= CURRENT_DATE) DESC, deadline DESC NULLS FIRST, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Grant{}
	for rows.Next() {
		g, err := scanGrant(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, nil
}

func getGrant(ctx context.Context, id int64) (Grant, error) {
	db, err := DB()
	if err != nil {
		return Grant{}, err
	}
	return scanGrant(db.QueryRow(ctx, `SELECT `+grantCols+` FROM grants WHERE id=$1`, id))
}

type SourceHealth struct {
	Source string  `json:"source"`
	OK     bool    `json:"ok"`
	Count  int     `json:"count"`
	Error  string  `json:"error"`
	RanAt  string  `json:"ran_at"`
	LastOK *string `json:"last_ok"`
}

func sourceHealth(ctx context.Context) ([]SourceHealth, error) {
	db, err := DB()
	if err != nil {
		return nil, err
	}
	rows, err := db.Query(ctx, `SELECT DISTINCT ON (source) source, ok, count, error, to_char(ran_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"'),
		(SELECT to_char(max(ran_at) AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"') FROM scraper_runs s2 WHERE s2.source=s.source AND s2.ok)
		FROM scraper_runs s ORDER BY source, ran_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []SourceHealth{}
	for rows.Next() {
		var h SourceHealth
		if err := rows.Scan(&h.Source, &h.OK, &h.Count, &h.Error, &h.RanAt, &h.LastOK); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, nil
}

// refreshIfStale triggers a background-safe synchronous refresh when data is older than maxAge.
func refreshIfStale(ctx context.Context, maxAge time.Duration) bool {
	db, err := DB()
	if err != nil {
		return false
	}
	var last *time.Time
	db.QueryRow(ctx, `SELECT max(ran_at) FROM scraper_runs`).Scan(&last)
	if last != nil && time.Since(*last) < maxAge {
		return false
	}
	RefreshAll(ctx)
	return true
}

func handleGrants(w http.ResponseWriter, r *http.Request, u *User) error {
	refreshed := refreshIfStale(r.Context(), 24*time.Hour)
	gs, err := listGrants(r.Context())
	if err != nil {
		return err
	}
	health, err := sourceHealth(r.Context())
	if err != nil {
		return err
	}
	if u.Lang != "en" {
		texts := make([]string, 0, len(gs)*2)
		for _, g := range gs {
			texts = append(texts, g.Summary, g.Eligibility)
		}
		tr := translateAll(r.Context(), texts, u.Lang)
		for i := range gs {
			gs[i].Summary, gs[i].Eligibility = tr[2*i], tr[2*i+1]
		}
	}
	return writeJSON(w, map[string]any{"grants": gs, "sources": health, "refreshed_now": refreshed})
}

func handleRefresh(w http.ResponseWriter, r *http.Request, u *User) error {
	return writeJSON(w, RefreshAll(r.Context()))
}

func handleCron(w http.ResponseWriter, r *http.Request) error {
	if s := os.Getenv("CRON_SECRET"); s == "" || r.Header.Get("Authorization") != "Bearer "+s {
		return httpErr(401, "unauthorized")
	}
	return writeJSON(w, RefreshAll(r.Context()))
}
