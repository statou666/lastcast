package main

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// ---------- users & sessions ----------

type User struct {
	ID   int64  `json:"id"`
	Name string `json:"username"`
	Role string `json:"role"`
}

const (
	sessionCookie = "lc_session"
	sessionTTL    = 30 * 24 * time.Hour
	pbkdf2Iter    = 120000
)

type ctxKeyUser struct{}

func currentUser(r *http.Request) *User {
	u, _ := r.Context().Value(ctxKeyUser{}).(*User)
	return u
}

func userID(r *http.Request) int64 {
	if u := currentUser(r); u != nil {
		return u.ID
	}
	return 0
}

func userCount() int {
	var n int
	_ = store.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&n)
	return n
}

func validName(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) < 2 || len(s) > 32 {
		return false
	}
	for _, r := range s {
		if !(r == '_' || r == '-' || r == '.' || r == '@' ||
			(r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') ||
			r >= 0x80) {
			return false
		}
	}
	return true
}

// pbkdf2SHA256 is a small stdlib-only PBKDF2 (HMAC-SHA256).
func pbkdf2SHA256(password, salt []byte, iter, keyLen int) []byte {
	hLen := sha256.Size
	blocks := (keyLen + hLen - 1) / hLen
	out := make([]byte, 0, blocks*hLen)
	for block := 1; block <= blocks; block++ {
		mac := hmac.New(sha256.New, password)
		mac.Write(salt)
		mac.Write([]byte{byte(block >> 24), byte(block >> 16), byte(block >> 8), byte(block)})
		u := mac.Sum(nil)
		t := make([]byte, len(u))
		copy(t, u)
		for i := 1; i < iter; i++ {
			mac = hmac.New(sha256.New, password)
			mac.Write(u)
			u = mac.Sum(nil)
			for j := range t {
				t[j] ^= u[j]
			}
		}
		out = append(out, t...)
	}
	return out[:keyLen]
}

func hashPassword(password, saltHex string) string {
	salt, _ := hex.DecodeString(saltHex)
	return hex.EncodeToString(pbkdf2SHA256([]byte(password), salt, pbkdf2Iter, 32))
}

func newSalt() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func createUser(name, password, role string) (int64, error) {
	name = strings.TrimSpace(name)
	if !validName(name) {
		return 0, errors.New("username must be 2-32 characters")
	}
	if len(password) < 6 {
		return 0, errors.New("password must be at least 6 characters")
	}
	if role != "admin" {
		role = "user"
	}
	salt := newSalt()
	res, err := store.db.Exec(`INSERT INTO users(username,pass_hash,salt,role,api_token,created_at) VALUES(?,?,?,?,?,?)`,
		name, hashPassword(password, salt), salt, role, randHex(24), nowUnix())
	if err != nil {
		return 0, errors.New("username already exists")
	}
	return res.LastInsertId()
}

func verifyUser(name, password string) *User {
	var id int64
	var hash, salt, role, uname string
	if err := store.db.QueryRow(`SELECT id,username,pass_hash,salt,role FROM users WHERE username=?`, name).
		Scan(&id, &uname, &hash, &salt, &role); err != nil {
		return nil
	}
	got := hashPassword(password, salt)
	if subtle.ConstantTimeCompare([]byte(got), []byte(hash)) != 1 {
		return nil
	}
	return &User{ID: id, Name: uname, Role: role}
}

func sessionUser(token string) *User {
	sum := sha256.Sum256([]byte(token))
	var u User
	err := store.db.QueryRow(`SELECT u.id,u.username,u.role FROM sessions s JOIN users u ON u.id=s.user_id
		WHERE s.token=? AND s.expires_at>?`, hex.EncodeToString(sum[:]), nowUnix()).
		Scan(&u.ID, &u.Name, &u.Role)
	if err != nil {
		return nil
	}
	return &u
}

func createSession(userID int64) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := hex.EncodeToString(b)
	sum := sha256.Sum256([]byte(token))
	now := nowUnix()
	_, _ = store.db.Exec(`DELETE FROM sessions WHERE expires_at<?`, now)
	_, err := store.db.Exec(`INSERT INTO sessions(token,user_id,created_at,expires_at) VALUES(?,?,?,?)`,
		hex.EncodeToString(sum[:]), userID, now, now+int64(sessionTTL/time.Second))
	return token, err
}

func destroySession(r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil {
		sum := sha256.Sum256([]byte(c.Value))
		_, _ = store.db.Exec(`DELETE FROM sessions WHERE token=?`, hex.EncodeToString(sum[:]))
	}
}

func setSessionCookie(w http.ResponseWriter, r *http.Request, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		MaxAge:   int(sessionTTL / time.Second),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https",
	})
}

func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: "", Path: "/", MaxAge: -1, HttpOnly: true})
}

// claimOrphans assigns pre-auth data (user_id=0) to the first account.
func claimOrphans(uid int64) {
	for _, table := range []string{"feeds", "favorites", "progress", "queue", "downloads", "events", "devices"} {
		_, _ = store.db.Exec(fmt.Sprintf(`UPDATE %s SET user_id=? WHERE user_id=0`, table), uid)
	}
}

// ---------- ownership helpers ----------

func feedOwned(uid, feedID int64) bool {
	var n int
	_ = store.db.QueryRow(`SELECT COUNT(*) FROM feeds WHERE id=? AND user_id=?`, feedID, uid).Scan(&n)
	return n > 0
}

func episodeOwned(uid, episodeID int64) bool {
	var n int
	_ = store.db.QueryRow(`SELECT COUNT(*) FROM episodes e JOIN feeds f ON f.id=e.feed_id WHERE e.id=? AND f.user_id=?`, episodeID, uid).Scan(&n)
	return n > 0
}

func episodeOwner(episodeID int64) int64 {
	var uid int64
	_ = store.db.QueryRow(`SELECT COALESCE(f.user_id,0) FROM episodes e JOIN feeds f ON f.id=e.feed_id WHERE e.id=?`, episodeID).Scan(&uid)
	return uid
}

func usernameOf(uid int64) string {
	var name string
	_ = store.db.QueryRow(`SELECT username FROM users WHERE id=?`, uid).Scan(&name)
	return name
}

func isAdmin(r *http.Request) bool {
	u := currentUser(r)
	return u != nil && u.Role == "admin"
}

// ---------- middleware ----------

func publicPath(p string) bool {
	switch p {
	case "/login", "/setup", "/healthz", "/api/login", "/api/setup", "/api/logout",
		"/style.css", "/app.js", "/favicon.svg", "/manifest.json",
		"/feeds.html", "/settings.html":
		return true
	}
	for _, pre := range []string{"/icons/", "/api/2/", "/gpodder/"} {
		if strings.HasPrefix(p, pre) {
			return true
		}
	}
	return false
}

func wantsJSON(r *http.Request) bool {
	p := r.URL.Path
	return strings.HasPrefix(p, "/api/") || strings.HasPrefix(p, "/stream/") ||
		strings.HasPrefix(p, "/media/") || strings.HasPrefix(p, "/speak/") || strings.HasPrefix(p, "/downloads/")
}

func withAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if publicPath(r.URL.Path) && r.URL.Path != "/api/logout" {
			next.ServeHTTP(w, r)
			return
		}
		// first run: no accounts yet → setup
		if userCount() == 0 {
			if strings.HasPrefix(r.URL.Path, "/api/") || wantsJSON(r) {
				writeJSONStatus(w, http.StatusForbidden, map[string]any{"error": "setup required", "setup": true})
				return
			}
			http.Redirect(w, r, "/setup", http.StatusSeeOther)
			return
		}
		var u *User
		if tu := tokenUser(r); tu != nil {
			u = tu
		} else if c, err := r.Cookie(sessionCookie); err == nil {
			u = sessionUser(c.Value)
		}
		if u == nil {
			if wantsJSON(r) {
				writeJSONStatus(w, http.StatusUnauthorized, map[string]any{"error": "unauthorized"})
				return
			}
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKeyUser{}, u)))
	})
}

func writeJSONStatus(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

// ---------- handlers ----------

func handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAuthPage(w, "login", "")
		return
	}
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONStatus(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}
	u := verifyUser(strings.TrimSpace(body.Username), body.Password)
	if u == nil {
		time.Sleep(400 * time.Millisecond) // slow down guessing
		writeJSONStatus(w, http.StatusUnauthorized, map[string]any{"error": "wrong username or password"})
		return
	}
	token, err := createSession(u.ID)
	if err != nil {
		writeJSONStatus(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	setSessionCookie(w, r, token)
	writeJSON(w, map[string]any{"ok": true, "user": u})
}

func handleSetup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAuthPage(w, "setup", "")
		return
	}
	if userCount() > 0 {
		writeJSONStatus(w, http.StatusForbidden, map[string]any{"error": "already set up"})
		return
	}
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONStatus(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}
	id, err := createUser(body.Username, body.Password, "admin")
	if err != nil {
		writeJSONStatus(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	claimOrphans(id)
	token, err := createSession(id)
	if err != nil {
		writeJSONStatus(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	setSessionCookie(w, r, token)
	writeJSON(w, map[string]any{"ok": true})
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	destroySession(r)
	clearSessionCookie(w)
	if r.Method == http.MethodPost {
		writeJSON(w, map[string]any{"ok": true})
		return
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func handleMe(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, currentUser(r))
}

// handleUsers: admin-only account management (list / create / delete).
func handleUsers(w http.ResponseWriter, r *http.Request) {
	me := currentUser(r)
	if me == nil || me.Role != "admin" {
		writeJSONStatus(w, http.StatusForbidden, map[string]any{"error": "admin only"})
		return
	}
	switch r.Method {
	case http.MethodGet:
		rows, err := store.db.Query(`SELECT id,username,role,created_at FROM users ORDER BY id`)
		if err != nil {
			writeJSON(w, map[string]any{"error": err.Error()})
			return
		}
		defer rows.Close()
		out := []map[string]any{}
		for rows.Next() {
			var id, created int64
			var name, role string
			if rows.Scan(&id, &name, &role, &created) == nil {
				out = append(out, map[string]any{"id": id, "username": name, "role": role, "created_at": created})
			}
		}
		writeJSON(w, out)
	case http.MethodPost:
		var body struct {
			Username string `json:"username"`
			Password string `json:"password"`
			Role     string `json:"role"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, map[string]any{"error": "invalid json"})
			return
		}
		id, err := createUser(body.Username, body.Password, body.Role)
		if err != nil {
			writeJSON(w, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, map[string]any{"ok": true, "id": id})
	case http.MethodDelete:
		id := int64(atoi(r.PathValue("id")))
		if id == 0 || id == me.ID {
			writeJSON(w, map[string]any{"error": "cannot delete this account"})
			return
		}
		var name string
		if err := store.db.QueryRow(`SELECT username FROM users WHERE id=?`, id).Scan(&name); err != nil {
			writeJSON(w, map[string]any{"error": "not found"})
			return
		}
		_, _ = store.db.Exec(`DELETE FROM favorites WHERE user_id=?`, id)
		_, _ = store.db.Exec(`DELETE FROM progress WHERE user_id=?`, id)
		_, _ = store.db.Exec(`DELETE FROM queue WHERE user_id=?`, id)
		_, _ = store.db.Exec(`DELETE FROM settings WHERE user_id=?`, id)
		_, _ = store.db.Exec(`DELETE FROM devices WHERE user_id=?`, id)
		_, _ = store.db.Exec(`DELETE FROM downloads WHERE user_id=?`, id)
		_, _ = store.db.Exec(`DELETE FROM episodes WHERE feed_id IN (SELECT id FROM feeds WHERE user_id=?)`, id)
		_, _ = store.db.Exec(`DELETE FROM feeds WHERE user_id=?`, id)
		_, _ = store.db.Exec(`DELETE FROM sessions WHERE user_id=?`, id)
		_, _ = store.db.Exec(`DELETE FROM users WHERE id=?`, id)
		writeJSON(w, map[string]any{"ok": true, "deleted": id})
	default:
		writeJSON(w, map[string]any{"error": "method not supported"})
	}
}

// ---------- login / setup page ----------

func writeAuthPage(w http.ResponseWriter, mode, errMsg string) {
	setup := mode == "setup"
	title := "Sign in · Lastcast"
	action := "/api/login"
	sub := "Sign in to your library"
	extra := ""
	if setup {
		title = "Create admin · Lastcast"
		action = "/api/setup"
		sub = "First run — create the administrator account"
		extra = `<label>Confirm password</label><input id="c" type="password" autocomplete="new-password" required>`
	}
	h1 := "Welcome back"
	btn := "Sign in"
	ac := "current-password"
	if setup {
		h1, btn, ac = "Create admin", "Create account", "new-password"
	}
	htmlPage := `<!doctype html><html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>` + title + `</title>
<link rel="icon" href="/favicon.svg" type="image/svg+xml">
<link rel="apple-touch-icon" href="/icons/apple-touch-icon.png">
<style>
:root{--bg:#0b0a12;--panel:#151221;--line:#2b2444;--ink:#eceaf6;--mut:#9a94b8;--accent:#ff6b35}
*{box-sizing:border-box}
body{margin:0;min-height:100dvh;display:grid;place-items:center;background:radial-gradient(900px 500px at 80% -10%,rgba(255,107,53,.12),transparent 60%),var(--bg);color:var(--ink);font:15px/1.5 -apple-system,"Segoe UI","PingFang SC",system-ui,sans-serif;padding:24px}
.card{width:min(400px,100%);background:var(--panel);border:1px solid var(--line);border-radius:20px;padding:28px;box-shadow:0 20px 60px rgba(0,0,0,.45)}
.brand{display:flex;align-items:center;gap:12px;margin-bottom:18px}
.brand img{width:44px;height:44px;border-radius:12px}
.brand b{letter-spacing:.14em;font-size:15px}
h1{font-size:19px;margin:0 0 4px}
p.sub{color:var(--mut);margin:0 0 20px;font-size:13px}
label{display:block;color:var(--mut);font-size:12px;letter-spacing:.08em;text-transform:uppercase;margin:14px 0 6px}
input{width:100%;padding:11px 12px;border-radius:10px;border:1px solid var(--line);background:#1d1830;color:var(--ink);font:inherit}
input:focus{outline:none;border-color:var(--accent);box-shadow:0 0 0 3px rgba(255,107,53,.15)}
button{width:100%;margin-top:20px;padding:12px;border-radius:10px;border:0;background:linear-gradient(135deg,#ff6b35,#ff9a62);color:#fff;font:inherit;font-weight:700;cursor:pointer}
.err{color:#fb7185;font-size:13px;margin-top:14px;min-height:18px}
</style></head><body>
<form class="card" id="f">
  <div class="brand"><img src="/icons/icon-192.png" alt=""><b>LASTCAST</b></div>
  <h1>` + h1 + `</h1>
  <p class="sub">` + sub + `</p>
  <label>Username</label><input id="u" autocomplete="username" required autofocus>
  <label>Password</label><input id="p" type="password" autocomplete="` + ac + `" required>
  ` + extra + `
  <button type="submit">` + btn + `</button>
  <div class="err" id="e">` + errMsg + `</div>
</form>
<script>
document.getElementById("f").addEventListener("submit", async (ev) => {
  ev.preventDefault();
  const u = document.getElementById("u").value.trim();
  const p = document.getElementById("p").value;
  const c = document.getElementById("c");
  const e = document.getElementById("e");
  if (c && c.value !== p) { e.textContent = "Passwords do not match"; return; }
  e.textContent = "…";
  try {
    const r = await fetch("` + action + `", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ username: u, password: p }) });
    const j = await r.json();
    if (j.ok) location.replace("/");
    else e.textContent = j.error || "Failed";
  } catch (err) { e.textContent = String(err); }
});
</script></body></html>`
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(htmlPage))
}

// handlePassword lets a signed-in user change their own password.
func handlePassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONStatus(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	u := currentUser(r)
	if u == nil {
		writeJSONStatus(w, http.StatusUnauthorized, map[string]any{"error": "unauthorized"})
		return
	}
	var body struct {
		Old string `json:"old"`
		New string `json:"new"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONStatus(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}
	if verifyUser(u.Name, body.Old) == nil {
		writeJSONStatus(w, http.StatusForbidden, map[string]any{"error": "current password is wrong"})
		return
	}
	if len(body.New) < 6 {
		writeJSONStatus(w, http.StatusBadRequest, map[string]any{"error": "new password must be at least 6 characters"})
		return
	}
	salt := newSalt()
	if _, err := store.db.Exec(`UPDATE users SET pass_hash=?, salt=? WHERE id=?`, hashPassword(body.New, salt), salt, u.ID); err != nil {
		writeJSONStatus(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	// keep this session, drop the user's other sessions
	if c, err := r.Cookie(sessionCookie); err == nil {
		sum := sha256.Sum256([]byte(c.Value))
		_, _ = store.db.Exec(`DELETE FROM sessions WHERE user_id=? AND token<>?`, u.ID, hex.EncodeToString(sum[:]))
	}
	writeJSON(w, map[string]any{"ok": true})
}
