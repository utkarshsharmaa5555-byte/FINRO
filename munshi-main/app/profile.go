package app

import (
	"context"
	"net/http"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

type Memory struct {
	ID        int64  `json:"id"`
	Fact      string `json:"fact"`
	CreatedAt string `json:"created_at"`
}

func loadProfile(ctx context.Context, uid int64) (map[string]any, []Memory, error) {
	db, err := DB()
	if err != nil {
		return nil, nil, err
	}
	profile := map[string]any{}
	if err := db.QueryRow(ctx, `SELECT data FROM profiles WHERE user_id=$1`, uid).Scan(&profile); err != nil {
		profile = map[string]any{}
	}
	rows, err := db.Query(ctx, `SELECT id, fact, to_char(created_at, 'YYYY-MM-DD') FROM memories WHERE user_id=$1 ORDER BY id DESC LIMIT 50`, uid)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	mems := []Memory{}
	for rows.Next() {
		var m Memory
		if err := rows.Scan(&m.ID, &m.Fact, &m.CreatedAt); err != nil {
			return nil, nil, err
		}
		mems = append(mems, m)
	}
	return profile, mems, nil
}

// mergeProfile shallow-merges fields into the profile; empty-string values delete the key.
func mergeProfile(ctx context.Context, uid int64, fields map[string]any) (map[string]any, error) {
	db, err := DB()
	if err != nil {
		return nil, err
	}
	var out map[string]any
	err = db.QueryRow(ctx, `INSERT INTO profiles(user_id, data) VALUES($1, $2)
		ON CONFLICT (user_id) DO UPDATE SET data = profiles.data || EXCLUDED.data, updated_at = now()
		RETURNING data`, uid, fields).Scan(&out)
	if err != nil {
		return nil, err
	}
	var drop []string
	for k, v := range out {
		if s, ok := v.(string); ok && s == "" {
			drop = append(drop, k)
		}
	}
	if len(drop) > 0 {
		err = db.QueryRow(ctx, `UPDATE profiles SET data = data - $2::text[] WHERE user_id=$1 RETURNING data`, uid, drop).Scan(&out)
	}
	return out, err
}

func addMemory(ctx context.Context, uid int64, fact string) error {
	fact = strings.TrimSpace(fact)
	if fact == "" || len(fact) > 400 {
		return httpErr(400, "memory must be 1–400 characters")
	}
	db, err := DB()
	if err != nil {
		return err
	}
	_, err = db.Exec(ctx, `INSERT INTO memories(user_id, fact) SELECT $1, $2 WHERE NOT EXISTS (SELECT 1 FROM memories WHERE user_id=$1 AND fact=$2)`, uid, fact)
	return err
}

func handleGetProfile(w http.ResponseWriter, r *http.Request, u *User) error {
	p, m, err := loadProfile(r.Context(), u.ID)
	if err != nil {
		return err
	}
	return writeJSON(w, map[string]any{"user": u, "profile": p, "memories": m})
}

func handlePutProfile(w http.ResponseWriter, r *http.Request, u *User) error {
	var in map[string]any
	if err := readJSON(r, &in); err != nil {
		return err
	}
	if len(in) > 40 {
		return httpErr(400, "too many fields")
	}
	p, err := mergeProfile(r.Context(), u.ID, in)
	if err != nil {
		return err
	}
	return writeJSON(w, p)
}

func handleAddMemory(w http.ResponseWriter, r *http.Request, u *User) error {
	var in struct{ Fact string }
	if err := readJSON(r, &in); err != nil {
		return err
	}
	if err := addMemory(r.Context(), u.ID, in.Fact); err != nil {
		return err
	}
	return writeJSON(w, map[string]bool{"ok": true})
}

func handleDeleteMemory(w http.ResponseWriter, r *http.Request, u *User) error {
	db, err := DB()
	if err != nil {
		return err
	}
	if r.PathValue("id") == "all" {
		_, err = db.Exec(r.Context(), `DELETE FROM memories WHERE user_id=$1`, u.ID)
	} else {
		_, err = db.Exec(r.Context(), `DELETE FROM memories WHERE id=$1 AND user_id=$2`, r.PathValue("id"), u.ID)
	}
	if err != nil {
		return err
	}
	return writeJSON(w, map[string]bool{"ok": true})
}

func handleMe(w http.ResponseWriter, r *http.Request, u *User) error { return writeJSON(w, u) }

func handleSettings(w http.ResponseWriter, r *http.Request, u *User) error {
	var in struct {
		Name   *string        `json:"name"`
		Lang   *string        `json:"lang"`
		Prefs  map[string]any `json:"prefs"`
		OldPin string         `json:"old_pin"`
		NewPin string         `json:"new_pin"`
	}
	if err := readJSON(r, &in); err != nil {
		return err
	}
	db, err := DB()
	if err != nil {
		return err
	}
	ctx := r.Context()
	if in.Name != nil {
		n := strings.TrimSpace(*in.Name)
		if n == "" || len(n) > 80 {
			return httpErr(400, "enter your name")
		}
		if _, err := db.Exec(ctx, `UPDATE users SET name=$1 WHERE id=$2`, n, u.ID); err != nil {
			return err
		}
	}
	if in.Lang != nil {
		if !langs[*in.Lang] {
			return httpErr(400, "unsupported language")
		}
		if _, err := db.Exec(ctx, `UPDATE users SET lang=$1 WHERE id=$2`, *in.Lang, u.ID); err != nil {
			return err
		}
	}
	if in.Prefs != nil {
		if _, err := db.Exec(ctx, `UPDATE users SET prefs = prefs || $1 WHERE id=$2`, in.Prefs, u.ID); err != nil {
			return err
		}
	}
	if in.NewPin != "" {
		var hash string
		if err := db.QueryRow(ctx, `SELECT pin_hash FROM users WHERE id=$1`, u.ID).Scan(&hash); err != nil {
			return err
		}
		if bcrypt.CompareHashAndPassword([]byte(hash), []byte(in.OldPin)) != nil {
			return httpErr(400, "current PIN is wrong")
		}
		if !pinRe.MatchString(in.NewPin) {
			return httpErr(400, "new PIN must be 4 digits")
		}
		h, err := bcrypt.GenerateFromPassword([]byte(in.NewPin), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		if _, err := db.Exec(ctx, `UPDATE users SET pin_hash=$1 WHERE id=$2`, string(h), u.ID); err != nil {
			return err
		}
	}
	nu, err := loadUser(ctx, u.ID)
	if err != nil {
		return err
	}
	return writeJSON(w, nu)
}

func handleDeleteAccount(w http.ResponseWriter, r *http.Request, u *User) error {
	var in struct{ Pin string }
	if err := readJSON(r, &in); err != nil {
		return err
	}
	db, err := DB()
	if err != nil {
		return err
	}
	var hash string
	if err := db.QueryRow(r.Context(), `SELECT pin_hash FROM users WHERE id=$1`, u.ID).Scan(&hash); err != nil {
		return err
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(in.Pin)) != nil {
		return httpErr(400, "PIN is wrong")
	}
	if _, err := db.Exec(r.Context(), `DELETE FROM users WHERE id=$1`, u.ID); err != nil {
		return err
	}
	return handleLogout(w, r)
}
