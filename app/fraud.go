package app

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type ScamPattern struct {
	ID          int64    `json:"id"`
	Slug        string   `json:"slug"`
	Kind        string   `json:"kind"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	RedFlags    []string `json:"red_flags"`
	Actions     []string `json:"actions"`
	SourceURL   string   `json:"source_url"`
	PatternSlug string   `json:"pattern_slug"`
	PublishedAt *string  `json:"published_at"`
}

func loadPatterns(ctx context.Context, kind string, limit int) ([]ScamPattern, error) {
	db, err := DB()
	if err != nil {
		return nil, err
	}
	rows, err := db.Query(ctx, `SELECT id, slug, kind, title, description, red_flags, actions, source_url, pattern_slug, to_char(published_at, 'YYYY-MM-DD')
		FROM scam_patterns WHERE kind=$1 ORDER BY COALESCE(published_at, created_at) DESC LIMIT $2`, kind, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ScamPattern{}
	for rows.Next() {
		var p ScamPattern
		if err := rows.Scan(&p.ID, &p.Slug, &p.Kind, &p.Title, &p.Description, &p.RedFlags, &p.Actions, &p.SourceURL, &p.PatternSlug, &p.PublishedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

type ScamVerdict struct {
	Verdict     string   `json:"verdict"` // scam | suspicious | looks_safe
	Confidence  int      `json:"confidence"`
	Headline    string   `json:"headline"`
	Matched     []string `json:"matched_patterns"`
	RedFlags    []string `json:"red_flags_found"`
	Actions     []string `json:"what_to_do"`
	Explanation string   `json:"explanation"`
}

var verdictSchema = obj(map[string]any{
	"verdict":          enum("scam", "suspicious", "looks_safe"),
	"confidence":       integer("0-100"),
	"headline":         str("one short line in the reply language"),
	"matched_patterns": arr(str("slug from the catalogue")),
	"red_flags_found":  arr(str("specific warning signs quoted/observed in THIS input, reply language")),
	"what_to_do":       arr(str("2-4 concrete actions, reply language")),
	"explanation":      str("2-3 simple sentences in the reply language, no jargon"),
})

// checkScam classifies a message or screenshot against the pattern catalogue + recent news.
func checkScam(ctx context.Context, uid int64, text, imageDataURL, lang string) (*ScamVerdict, error) {
	cat, err := loadPatterns(ctx, "catalogue", 50)
	if err != nil {
		return nil, err
	}
	news, _ := loadPatterns(ctx, "news", 8)
	var b strings.Builder
	for _, p := range cat {
		fmt.Fprintf(&b, "- %s: %s. %s Red flags: %s\n", p.Slug, p.Title, p.Description, strings.Join(p.RedFlags, "; "))
	}
	if len(news) > 0 {
		b.WriteString("\nRecently reported in Indian news:\n")
		for _, n := range news {
			fmt.Fprintf(&b, "- %s (pattern: %s)\n", n.Title, n.PatternSlug)
		}
	}
	sys := "You are Finro's fraud desk for small Indian merchants. Judge whether the input (SMS, WhatsApp message, call description, payment screenshot or QR) is a UPI/payment scam. " +
		"Use this catalogue of current patterns:\n" + b.String() +
		"\nRules: A genuine credit never needs a PIN, QR scan, app install or OTP. Be decisive but honest: use 'suspicious' when information is missing. " +
		"Always include calling 1930 / cybercrime.gov.in if money may already be lost. Reply language: " + langNames[lang] + "."
	var user any = "Input to check:\n" + text
	if imageDataURL != "" {
		user = []map[string]any{
			{"type": "text", "text": "Check this screenshot for scam signs (look at sender, amounts, UPI IDs, links, edited-looking fonts, 'receive' requests). Extra context: " + text},
			{"type": "image_url", "image_url": map[string]any{"url": imageDataURL}},
		}
	}
	v := &ScamVerdict{}
	if err := chatJSON(ctx, "scam_check", sys, user, verdictSchema, v); err != nil {
		return nil, err
	}
	if db, err := DB(); err == nil {
		input := text
		if imageDataURL != "" {
			input = "[screenshot] " + text
		}
		db.Exec(ctx, `INSERT INTO scam_checks(user_id, input, verdict, result) VALUES($1,$2,$3,$4)`, uid, truncate(input, 2000), v.Verdict, v)
	}
	return v, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

type rss struct {
	Items []struct {
		Title   string `xml:"title"`
		Link    string `xml:"link"`
		PubDate string `xml:"pubDate"`
		Source  string `xml:"source"`
	} `xml:"channel>item"`
}

// refreshScamNews pulls Google News RSS and lets the model map each headline to a catalogue pattern.
func refreshScamNews(ctx context.Context) (int, error) {
	q := url.QueryEscape(`"UPI" (scam OR fraud) when:14d`)
	body, err := fetch(ctx, "https://news.google.com/rss/search?q="+q+"&hl=en-IN&gl=IN&ceid=IN:en")
	if err != nil {
		return 0, err
	}
	var feed rss
	if err := xml.Unmarshal(body, &feed); err != nil {
		return 0, err
	}
	if len(feed.Items) > 15 {
		feed.Items = feed.Items[:15]
	}
	if len(feed.Items) == 0 {
		return 0, fmt.Errorf("no news items")
	}
	titles := make([]string, len(feed.Items))
	for i, it := range feed.Items {
		titles[i] = it.Title
	}
	cat, err := loadPatterns(ctx, "catalogue", 50)
	if err != nil {
		return 0, err
	}
	slugs := make([]string, 0, len(cat)+2)
	for _, p := range cat {
		slugs = append(slugs, p.Slug)
	}
	slugs = append(slugs, "new-pattern", "not-a-scam-story")
	var res struct {
		Items []struct {
			Index   int    `json:"index"`
			Pattern string `json:"pattern"`
			Lesson  string `json:"lesson"`
		} `json:"items"`
	}
	sys := "Classify each Indian news headline about UPI fraud into one scam pattern slug. Use 'not-a-scam-story' for policy/stats news without a modus operandi. " +
		"'lesson' = one practical sentence a shopkeeper can act on."
	schema := obj(map[string]any{"items": arr(obj(map[string]any{"index": integer(""), "pattern": enum(slugs...), "lesson": str("")}))})
	if err := chatJSON(ctx, "scam_news", sys, mustJSON(titles), schema, &res); err != nil {
		return 0, err
	}
	db, err := DB()
	if err != nil {
		return 0, err
	}
	n := 0
	for _, it := range res.Items {
		if it.Index < 0 || it.Index >= len(feed.Items) || it.Pattern == "not-a-scam-story" {
			continue
		}
		item := feed.Items[it.Index]
		pub, err := time.Parse(time.RFC1123, item.PubDate)
		if err != nil {
			pub = time.Now()
		}
		tag, err := db.Exec(ctx, `INSERT INTO scam_patterns(slug, kind, title, description, source_url, pattern_slug, published_at, actions)
			VALUES($1,'news',$2,$3,$4,$5,$6,$7) ON CONFLICT (slug) DO NOTHING`,
			"news-"+hashText(item.Link)[:16], item.Title, it.Lesson, item.Link, it.Pattern, pub, []string{it.Lesson})
		if err == nil && tag.RowsAffected() > 0 {
			n++
		}
	}
	return n, nil
}

func handlePatterns(w http.ResponseWriter, r *http.Request, u *User) error {
	cat, err := loadPatterns(r.Context(), "catalogue", 50)
	if err != nil {
		return err
	}
	news, err := loadPatterns(r.Context(), "news", 12)
	if err != nil {
		return err
	}
	if lang := u.Lang; lang != "en" {
		localizePatterns(r.Context(), cat, lang)
		localizePatterns(r.Context(), news, lang)
	}
	return writeJSON(w, map[string]any{"catalogue": cat, "news": news})
}

func localizePatterns(ctx context.Context, ps []ScamPattern, lang string) {
	var texts []string
	for _, p := range ps {
		texts = append(texts, p.Title, p.Description)
		texts = append(texts, p.RedFlags...)
		texts = append(texts, p.Actions...)
	}
	tr := translateAll(ctx, texts, lang)
	i := 0
	for k := range ps {
		ps[k].Title, ps[k].Description = tr[i], tr[i+1]
		i += 2
		for j := range ps[k].RedFlags {
			ps[k].RedFlags[j] = tr[i]
			i++
		}
		for j := range ps[k].Actions {
			ps[k].Actions[j] = tr[i]
			i++
		}
	}
}

func handleScamCheck(w http.ResponseWriter, r *http.Request, u *User) error {
	var in struct {
		Text  string `json:"text"`
		Image string `json:"image"`
	}
	if err := readJSON(r, &in); err != nil {
		return err
	}
	if strings.TrimSpace(in.Text) == "" && in.Image == "" {
		return httpErr(400, "paste a message or add a screenshot")
	}
	if in.Image != "" && !strings.HasPrefix(in.Image, "data:image/") {
		return httpErr(400, "screenshot must be an image")
	}
	v, err := checkScam(r.Context(), u.ID, truncate(in.Text, 4000), in.Image, u.Lang)
	if err != nil {
		return err
	}
	return writeJSON(w, v)
}
