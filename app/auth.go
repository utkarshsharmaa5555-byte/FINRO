package app

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID    int64          `json:"id"`
	Phone string         `json:"phone"`
	Name  string         `json:"name"`
	Lang  string         `json:"lang"`
	Role  string         `json:"role"`
	Prefs map[string]any `json:"prefs"`
}

const cookieName = "finro_session"

var (
	phoneRe = regexp.MustCompile(`^[6-9]\d{9}$`)
	pinRe   = regexp.MustCompile(`^\d{4}$`)
	langs   = map[string]bool{"en": true, "hi": true, "te": true, "ta": true}
)

func secret() []byte {
	s := os.Getenv("SESSION_SECRET")
	if s == "" {
		s = "dev-only-secret-change-me"
	}
	return []byte(s)
}

func sign(payload string) string {
	m := hmac.New(sha256.New, secret())
	m.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(m.Sum(nil))
}

func setSession(w http.ResponseWriter, uid int64) {
	exp := time.Now().Add(30 * 24 * time.Hour)
	payload := fmt.Sprintf("%d.%d", uid, exp.Unix())
	http.SetCookie(w, &http.Cookie{
		Name: cookieName, Value: payload + "." + sign(payload), Path: "/",
		Expires: exp, HttpOnly: true, Secure: os.Getenv("VERCEL") != "", SameSite: http.SameSiteLaxMode,
	})
}

// sessionUID returns the user id from a valid, unexpired signed cookie.
func sessionUID(r *http.Request) (int64, bool) {
	c, err := r.Cookie(cookieName)
	if err != nil {
		return 0, false
	}
	parts := strings.Split(c.Value, ".")
	if len(parts) != 3 {
		return 0, false
	}
	payload := parts[0] + "." + parts[1]
	if !hmac.Equal([]byte(sign(payload)), []byte(parts[2])) {
		return 0, false
	}
	exp, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || time.Now().Unix() > exp {
		return 0, false
	}
	uid, err := strconv.ParseInt(parts[0], 10, 64)
	return uid, err == nil
}

func loadUser(ctx context.Context, id int64) (*User, error) {
	db, err := DB()
	if err != nil {
		return nil, err
	}
	u := &User{}
	err = db.QueryRow(ctx, `SELECT id, phone, name, lang, role, prefs FROM users WHERE id=$1`, id).
		Scan(&u.ID, &u.Phone, &u.Name, &u.Lang, &u.Role, &u.Prefs)
	return u, err
}

type authedHandler func(w http.ResponseWriter, r *http.Request, u *User) error

func authed(h authedHandler) http.HandlerFunc {
	return handle(func(w http.ResponseWriter, r *http.Request) error {
		uid, ok := sessionUID(r)
		if !ok {
			return httpErr(401, "please log in")
		}
		u, err := loadUser(r.Context(), uid)
		if errors.Is(err, pgx.ErrNoRows) {
			return httpErr(401, "please log in")
		}
		if err != nil {
			return err
		}
		return h(w, r, u)
	})
}

func adminOnly(h authedHandler) http.HandlerFunc {
	return authed(func(w http.ResponseWriter, r *http.Request, u *User) error {
		if u.Role != "admin" {
			return httpErr(403, "admins only")
		}
		return h(w, r, u)
	})
}

func handleLogin(w http.ResponseWriter, r *http.Request) error {
	var in struct{ Phone, Pin string }
	if err := readJSON(r, &in); err != nil {
		return err
	}
	db, err := DB()
	if err != nil {
		return err
	}
	var id int64
	var hash string
	err = db.QueryRow(r.Context(), `SELECT id, pin_hash FROM users WHERE phone=$1`, strings.TrimSpace(in.Phone)).Scan(&id, &hash)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(hash), []byte(in.Pin)) != nil {
		return httpErr(401, "phone or PIN is wrong")
	}
	setSession(w, id)
	u, err := loadUser(r.Context(), id)
	if err != nil {
		return err
	}
	return writeJSON(w, u)
}

func handleSignup(w http.ResponseWriter, r *http.Request) error {
	var in struct{ Phone, Pin, Name, Lang, Business string }
	if err := readJSON(r, &in); err != nil {
		return err
	}
	in.Phone, in.Name = strings.TrimSpace(in.Phone), strings.TrimSpace(in.Name)
	switch {
	case !phoneRe.MatchString(in.Phone):
		return httpErr(400, "enter a 10-digit mobile number")
	case !pinRe.MatchString(in.Pin):
		return httpErr(400, "PIN must be 4 digits")
	case in.Name == "" || len(in.Name) > 80:
		return httpErr(400, "enter your name")
	}
	if !langs[in.Lang] {
		in.Lang = "en"
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Pin), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	db, err := DB()
	if err != nil {
		return err
	}
	var id int64
	err = db.QueryRow(r.Context(), `INSERT INTO users(phone, pin_hash, name, lang) VALUES($1,$2,$3,$4)
		ON CONFLICT (phone) DO NOTHING RETURNING id`, in.Phone, string(hash), in.Name, in.Lang).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return httpErr(409, "this number is already registered — log in instead")
	}
	if err != nil {
		return err
	}
	profile := map[string]any{"owner_name": in.Name}
	if b := strings.TrimSpace(in.Business); b != "" {
		profile["business_name"] = b
	}
	if _, err := db.Exec(r.Context(), `INSERT INTO profiles(user_id, data) VALUES($1,$2)`, id, profile); err != nil {
		return err
	}
	setSession(w, id)
	u, err := loadUser(r.Context(), id)
	if err != nil {
		return err
	}
	return writeJSON(w, u)
}

func handleLogout(w http.ResponseWriter, r *http.Request) error {
	http.SetCookie(w, &http.Cookie{Name: cookieName, Value: "", Path: "/", MaxAge: -1, HttpOnly: true})
	return writeJSON(w, map[string]bool{"ok": true})
}
