package app

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"net/http"
)

var langNames = map[string]string{"en": "English", "hi": "Hindi", "te": "Telugu", "ta": "Tamil"}

func hashText(s string) string {
	h := sha1.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}

// translateAll translates texts to lang with a DB cache; falls back to the original text on failure.
// This is the IndicTrans2 stand-in: an LLM translation layer with a persistent cache.
func translateAll(ctx context.Context, texts []string, lang string) []string {
	out := append([]string(nil), texts...)
	if lang == "en" || !langs[lang] || len(texts) == 0 {
		return out
	}
	db, err := DB()
	if err != nil {
		return out
	}
	hashes := make([]string, len(texts))
	for i, t := range texts {
		hashes[i] = hashText(t)
	}
	rows, err := db.Query(ctx, `SELECT hash, text FROM translations WHERE lang=$1 AND hash = ANY($2)`, lang, hashes)
	if err != nil {
		return out
	}
	cached := map[string]string{}
	for rows.Next() {
		var h, t string
		if rows.Scan(&h, &t) == nil {
			cached[h] = t
		}
	}
	rows.Close()
	var missing []int
	for i, t := range texts {
		if c, ok := cached[hashes[i]]; ok {
			out[i] = c
		} else if t != "" {
			missing = append(missing, i)
		}
	}
	if len(missing) == 0 {
		return out
	}
	src := make([]string, len(missing))
	for j, i := range missing {
		src[j] = texts[i]
	}
	var res struct {
		Items []string `json:"items"`
	}
	sys := "Translate each item into simple, everyday " + langNames[lang] + " that a small shopkeeper with basic schooling understands. " +
		"Keep numbers, ₹ amounts, dates, URLs, UPI IDs, scheme names and phone numbers exactly as they are. Keep markdown formatting. Return items in the same order."
	if err := chatJSON(ctx, "translate", sys, mustJSON(map[string]any{"items": src}), obj(map[string]any{"items": arr(str(""))}), &res); err != nil || len(res.Items) != len(src) {
		return out
	}
	for j, i := range missing {
		out[i] = res.Items[j]
		db.Exec(ctx, `INSERT INTO translations(hash, lang, text) VALUES($1,$2,$3) ON CONFLICT DO NOTHING`, hashes[i], lang, res.Items[j])
	}
	return out
}

func handleTranslate(w http.ResponseWriter, r *http.Request, u *User) error {
	var in struct {
		Texts []string `json:"texts"`
		Lang  string   `json:"lang"`
	}
	if err := readJSON(r, &in); err != nil {
		return err
	}
	if len(in.Texts) > 80 {
		return httpErr(400, "too many texts")
	}
	return writeJSON(w, map[string]any{"texts": translateAll(r.Context(), in.Texts, in.Lang)})
}
