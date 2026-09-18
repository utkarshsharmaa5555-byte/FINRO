package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	qrcode "github.com/skip2/go-qrcode"
)

var vpaRe = regexp.MustCompile(`^[a-zA-Z0-9.\-_]{2,256}@[a-zA-Z]{2,64}$`)

// upiLink builds an NPCI-style UPI deep link. amountPaise 0 means "customer enters amount".
func upiLink(vpa, payee string, amountPaise int64) (string, error) {
	vpa = strings.TrimSpace(vpa)
	payee = strings.TrimSpace(payee)
	if !vpaRe.MatchString(vpa) {
		return "", errors.New("UPI ID looks wrong — it should look like name@bank")
	}
	if payee == "" || len(payee) > 60 {
		return "", errors.New("enter a shop name (up to 60 letters)")
	}
	if amountPaise < 0 || amountPaise > 10_00_000_00 {
		return "", errors.New("amount must be between ₹0 and ₹1,00,000")
	}
	v := url.Values{}
	v.Set("pa", vpa)
	v.Set("pn", payee)
	v.Set("cu", "INR")
	if amountPaise > 0 {
		v.Set("am", fmt.Sprintf("%d.%02d", amountPaise/100, amountPaise%100))
	}
	// url.Values encodes spaces as '+', which some UPI apps show literally; use %20.
	return "upi://pay?" + strings.ReplaceAll(v.Encode(), "+", "%20"), nil
}

type QR struct {
	ID          int64  `json:"id"`
	VPA         string `json:"vpa"`
	Payee       string `json:"payee"`
	AmountPaise *int64 `json:"amount_paise"`
	Link        string `json:"upi_link"`
}

func createQR(ctx context.Context, uid int64, vpa, payee string, amountPaise int64) (*QR, error) {
	link, err := upiLink(vpa, payee, amountPaise)
	if err != nil {
		return nil, httpErr(400, err.Error())
	}
	db, err := DB()
	if err != nil {
		return nil, err
	}
	q := &QR{VPA: strings.TrimSpace(vpa), Payee: strings.TrimSpace(payee), Link: link}
	if amountPaise > 0 {
		q.AmountPaise = &amountPaise
	}
	if err := db.QueryRow(ctx, `INSERT INTO qr_codes(user_id, vpa, payee, amount_paise, upi_link) VALUES($1,$2,$3,$4,$5) RETURNING id`,
		uid, q.VPA, q.Payee, q.AmountPaise, link).Scan(&q.ID); err != nil {
		return nil, err
	}
	// Remember the VPA on the profile so drafts and the standee can use it.
	db.Exec(ctx, `UPDATE profiles SET data = data || jsonb_build_object('upi_vpa', $2::text), updated_at=now() WHERE user_id=$1`, uid, q.VPA)
	return q, nil
}

func handleCreateQR(w http.ResponseWriter, r *http.Request, u *User) error {
	var in struct {
		VPA         string `json:"vpa"`
		Payee       string `json:"payee"`
		AmountPaise int64  `json:"amount_paise"`
	}
	if err := readJSON(r, &in); err != nil {
		return err
	}
	if in.AmountPaise <= 0 {
		return httpErr(400, "amount must be greater than ₹0")
	}
	q, err := createQR(r.Context(), u.ID, in.VPA, in.Payee, in.AmountPaise)
	if err != nil {
		return err
	}
	return writeJSON(w, q)
}

func handleListQR(w http.ResponseWriter, r *http.Request, u *User) error {
	db, err := DB()
	if err != nil {
		return err
	}
	rows, err := db.Query(r.Context(), `SELECT id, vpa, payee, amount_paise, upi_link FROM qr_codes WHERE user_id=$1 ORDER BY id DESC LIMIT 20`, u.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	out := []QR{}
	for rows.Next() {
		var q QR
		if err := rows.Scan(&q.ID, &q.VPA, &q.Payee, &q.AmountPaise, &q.Link); err != nil {
			return err
		}
		out = append(out, q)
	}
	return writeJSON(w, out)
}

// handleQRPNG renders the user's own QR as PNG (size query param, 256–1024).
func handleQRPNG(w http.ResponseWriter, r *http.Request, u *User) error {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	db, err := DB()
	if err != nil {
		return err
	}
	var link string
	if err := db.QueryRow(r.Context(), `SELECT upi_link FROM qr_codes WHERE id=$1 AND user_id=$2`, id, u.ID).Scan(&link); err != nil {
		return httpErr(404, "QR not found")
	}
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	size = min(max(size, 256), 1024)
	png, err := qrcode.Encode(link, qrcode.High, size)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "private, max-age=86400")
	_, err = w.Write(png)
	return err
}
