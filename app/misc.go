package app

import (
	"io"
	"net/http"
	"strings"
)

// ---------- Help desk ----------

type FAQ struct {
	ID    int64  `json:"id"`
	Topic string `json:"topic"`
	Q     string `json:"q"`
	A     string `json:"a"`
}

func loadFAQs(r *http.Request, lang string) ([]FAQ, error) {
	db, err := DB()
	if err != nil {
		return nil, err
	}
	rows, err := db.Query(r.Context(), `SELECT id, topic, q, a FROM faqs ORDER BY sort`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []FAQ{}
	for rows.Next() {
		var f FAQ
		if err := rows.Scan(&f.ID, &f.Topic, &f.Q, &f.A); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	if lang != "en" {
		texts := make([]string, 0, 2*len(out))
		for _, f := range out {
			texts = append(texts, f.Q, f.A)
		}
		tr := translateAll(r.Context(), texts, lang)
		for i := range out {
			out[i].Q, out[i].A = tr[2*i], tr[2*i+1]
		}
	}
	return out, nil
}

func handleFAQs(w http.ResponseWriter, r *http.Request, u *User) error {
	fs, err := loadFAQs(r, u.Lang)
	if err != nil {
		return err
	}
	return writeJSON(w, fs)
}

func handleHelpAsk(w http.ResponseWriter, r *http.Request, u *User) error {
	var in struct{ Question string }
	if err := readJSON(r, &in); err != nil {
		return err
	}
	if strings.TrimSpace(in.Question) == "" {
		return httpErr(400, "type your question")
	}
	fs, err := loadFAQs(r, "en")
	if err != nil {
		return err
	}
	var kb strings.Builder
	for _, f := range fs {
		kb.WriteString("Q: " + f.Q + "\nA: " + f.A + "\n")
	}
	kb.WriteString("App sections: Assistant (chat), Live Funding, Matching, Applications, Khata (records), Business Profile & Memory, Fraud desk, Help desk, Settings. " +
		"Theme and language are in the top bar. PIN change and account deletion are in Settings.")
	var res struct {
		Answer        string `json:"answer"`
		Confident     bool   `json:"confident"`
		SuggestTicket bool   `json:"suggest_ticket"`
	}
	sys := "You are Finro's help desk. Answer ONLY from the knowledge base in " + langNames[u.Lang] + ", in 1-3 short sentences. " +
		"If the knowledge base does not cover it, say you are not sure and set suggest_ticket=true."
	schema := obj(map[string]any{"answer": str(""), "confident": map[string]any{"type": "boolean"}, "suggest_ticket": map[string]any{"type": "boolean"}})
	if err := chatJSON(r.Context(), "help", sys, "KNOWLEDGE BASE:\n"+kb.String()+"\n\nQUESTION: "+truncate(in.Question, 1000), schema, &res); err != nil {
		return err
	}
	return writeJSON(w, res)
}

func handleCreateTicket(w http.ResponseWriter, r *http.Request, u *User) error {
	var in struct{ Subject, Body string }
	if err := readJSON(r, &in); err != nil {
		return err
	}
	in.Subject, in.Body = strings.TrimSpace(in.Subject), strings.TrimSpace(in.Body)
	if in.Subject == "" || in.Body == "" || len(in.Subject) > 150 || len(in.Body) > 3000 {
		return httpErr(400, "add a subject and describe the problem")
	}
	db, err := DB()
	if err != nil {
		return err
	}
	var id int64
	if err := db.QueryRow(r.Context(), `INSERT INTO tickets(user_id, subject, body) VALUES($1,$2,$3) RETURNING id`, u.ID, in.Subject, in.Body).Scan(&id); err != nil {
		return err
	}
	return writeJSON(w, map[string]any{"id": id})
}

func handleTickets(w http.ResponseWriter, r *http.Request, u *User) error {
	db, err := DB()
	if err != nil {
		return err
	}
	q := `SELECT t.id, t.subject, t.body, t.status, to_char(t.created_at,'YYYY-MM-DD HH24:MI'), u.name FROM tickets t JOIN users u ON u.id=t.user_id WHERE t.user_id=$1 ORDER BY t.id DESC`
	args := []any{u.ID}
	if u.Role == "admin" && r.URL.Query().Get("all") == "1" {
		q = `SELECT t.id, t.subject, t.body, t.status, to_char(t.created_at,'YYYY-MM-DD HH24:MI'), u.name FROM tickets t JOIN users u ON u.id=t.user_id ORDER BY t.id DESC LIMIT 50`
		args = nil
	}
	rows, err := db.Query(r.Context(), q, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id int64
		var subj, body, status, created, name string
		if err := rows.Scan(&id, &subj, &body, &status, &created, &name); err != nil {
			return err
		}
		out = append(out, map[string]any{"id": id, "subject": subj, "body": body, "status": status, "created_at": created, "name": name})
	}
	return writeJSON(w, out)
}

// ---------- Voice ----------

func handleTranscribe(w http.ResponseWriter, r *http.Request, u *User) error {
	r.Body = http.MaxBytesReader(w, r.Body, 4<<20)
	f, hdr, err := r.FormFile("audio")
	if err != nil {
		return httpErr(400, "recording missing or longer than about 2 minutes")
	}
	defer f.Close()
	text, err := transcribe(r.Context(), f, hdr.Filename, u.Lang)
	if err != nil {
		return httpErr(502, "could not hear that clearly — please try again")
	}
	return writeJSON(w, map[string]string{"text": text})
}

func handleTTS(w http.ResponseWriter, r *http.Request, u *User) error {
	var in struct{ Text string }
	if err := readJSON(r, &in); err != nil {
		return err
	}
	text := strings.NewReplacer("**", "", "*", "", "#", "").Replace(strings.TrimSpace(in.Text))
	if text == "" {
		return httpErr(400, "nothing to read")
	}
	audio, err := speak(r.Context(), truncate(text, 3500))
	if err != nil {
		return httpErr(502, "voice unavailable")
	}
	defer audio.Close()
	w.Header().Set("Content-Type", "audio/mpeg")
	_, err = io.Copy(w, audio)
	return err
}

// ---------- Admin & public ----------

func handleAdminOverview(w http.ResponseWriter, r *http.Request, u *User) error {
	db, err := DB()
	if err != nil {
		return err
	}
	ctx := r.Context()
	counts := map[string]int64{}
	for k, q := range map[string]string{
		"merchants": `SELECT count(*) FROM users WHERE role='merchant'`, "sessions": `SELECT count(*) FROM sessions`,
		"messages": `SELECT count(*) FROM messages`, "scam_checks": `SELECT count(*) FROM scam_checks`,
		"scams_flagged": `SELECT count(*) FROM scam_checks WHERE verdict='scam'`, "qr_codes": `SELECT count(*) FROM qr_codes`,
		"ledger_entries": `SELECT count(*) FROM ledger_entries`, "blueprints": `SELECT count(*) FROM blueprints`,
		"drafts": `SELECT count(*) FROM drafts`, "grants": `SELECT count(*) FROM grants`, "open_tickets": `SELECT count(*) FROM tickets WHERE status='open'`,
	} {
		var n int64
		db.QueryRow(ctx, q).Scan(&n)
		counts[k] = n
	}
	health, err := sourceHealth(ctx)
	if err != nil {
		return err
	}
	ai := []map[string]any{}
	rows, err := db.Query(ctx, `SELECT kind, count(*), COALESCE(avg(ms),0)::int, COALESCE(percentile_cont(0.9) WITHIN GROUP (ORDER BY ms),0)::int, COALESCE(sum(tokens),0), count(*) FILTER (WHERE NOT ok)
		FROM ai_calls WHERE created_at > now() - interval '7 days' GROUP BY kind ORDER BY count(*) DESC`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var kind string
		var n, avg, p90, tokens, failed int64
		if err := rows.Scan(&kind, &n, &avg, &p90, &tokens, &failed); err != nil {
			rows.Close()
			return err
		}
		ai = append(ai, map[string]any{"kind": kind, "calls": n, "avg_ms": avg, "p90_ms": p90, "tokens": tokens, "failed": failed})
	}
	rows.Close()
	runs := []map[string]any{}
	rows, err = db.Query(ctx, `SELECT source, ok, count, error, ms, to_char(ran_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"') FROM scraper_runs ORDER BY id DESC LIMIT 30`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var source, errMsg, ranAt string
		var ok bool
		var count, ms int
		if err := rows.Scan(&source, &ok, &count, &errMsg, &ms, &ranAt); err != nil {
			return err
		}
		runs = append(runs, map[string]any{"source": source, "ok": ok, "count": count, "error": errMsg, "ms": ms, "ran_at": ranAt})
	}
	return writeJSON(w, map[string]any{"counts": counts, "sources": health, "ai": ai, "runs": runs})
}

func handlePublicStats(w http.ResponseWriter, r *http.Request) error {
	db, err := DB()
	if err != nil {
		return err
	}
	var grants, patterns, sources int64
	var last *string
	db.QueryRow(r.Context(), `SELECT (SELECT count(*) FROM grants), (SELECT count(*) FROM scam_patterns),
		(SELECT count(DISTINCT source) FROM grants), (SELECT to_char(max(ran_at) AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"') FROM scraper_runs WHERE ok)`).Scan(&grants, &patterns, &sources, &last)
	w.Header().Set("Cache-Control", "public, max-age=60")
	return writeJSON(w, map[string]any{"grants": grants, "scam_patterns": patterns, "sources": sources, "last_refresh": last})
}
