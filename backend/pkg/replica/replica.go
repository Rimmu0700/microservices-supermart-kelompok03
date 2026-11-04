// Package replica menyediakan replikasi data referensi antar service.
//
// Sisi PEMILIK data memanggil RegisterExporter untuk membuka endpoint internal
//
//	GET /internal/replica/:table
//
// Sisi PEMAKAI data menjalankan Syncer yang menarik endpoint itu secara berkala lalu
// meng-upsert hasilnya ke tabel replika di database service-nya sendiri.
// Dengan begitu query JOIN lama tetap jalan, tetapi tidak ada lagi akses lintas database.
package replica

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/jmoiron/sqlx"
)

// TokenHeader adalah header rahasia untuk panggilan internal antar service.
const TokenHeader = "X-Internal-Token"

// Token membaca token internal dari env INTERNAL_TOKEN.
func Token() string {
	if v := os.Getenv("INTERNAL_TOKEN"); v != "" {
		return v
	}
	return "supermart-internal-token-2026"
}

// RequireToken menolak request internal tanpa token yang benar.
func RequireToken() fiber.Handler {
	return func(c fiber.Ctx) error {
		if c.Get(TokenHeader) != Token() {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"success": false, "message": "invalid internal token"})
		}
		return c.Next()
	}
}

// ---------------------------------------------------------------- sisi PEMILIK

// Export menyatakan satu tabel yang boleh diekspor beserta kolom yang disembunyikan.
type Export struct {
	Table   string
	Exclude []string
}

// RegisterExporter membuka GET /internal/replica/:table (dilindungi token internal).
func RegisterExporter(app *fiber.App, db *sqlx.DB, exports []Export) {
	allowed := make(map[string]map[string]bool, len(exports))
	for _, e := range exports {
		hidden := make(map[string]bool, len(e.Exclude))
		for _, col := range e.Exclude {
			hidden[col] = true
		}
		allowed[e.Table] = hidden
	}

	app.Get("/internal/replica/:table", func(c fiber.Ctx) error {
		table := c.Params("table")
		hidden, ok := allowed[table]
		if !ok {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "message": "table not exported"})
		}

		rows, err := db.Queryx("SELECT * FROM `" + table + "`")
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": err.Error()})
		}
		defer rows.Close()

		out := make([]map[string]interface{}, 0)
		for rows.Next() {
			row := make(map[string]interface{})
			if err := rows.MapScan(row); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": err.Error()})
			}
			for k, v := range row {
				if hidden[k] {
					delete(row, k)
					continue
				}
				row[k] = normalize(v)
			}
			out = append(out, row)
		}
		if err := rows.Err(); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": err.Error()})
		}
		return c.JSON(fiber.Map{"table": table, "rows": out})
	}, RequireToken())
}

// normalize mengubah nilai hasil MapScan agar aman di-JSON-kan dan dibaca kembali MySQL.
func normalize(v interface{}) interface{} {
	switch t := v.(type) {
	case []byte:
		return string(t)
	case time.Time:
		return t.Format("2006-01-02 15:04:05.999999")
	default:
		return v
	}
}

// ---------------------------------------------------------------- sisi PEMAKAI

// Source menyatakan satu tabel replika dan alamat service pemiliknya.
type Source struct {
	Table string // nama tabel (sama di pemilik dan di replika)
	URL   string // mis. http://auth-service:3000
}

// Syncer menarik data dari service pemilik dan meng-upsert ke database lokal.
type Syncer struct {
	DB       *sqlx.DB
	Sources  []Source
	Interval time.Duration
	Client   *http.Client

	failing map[string]bool
}

// Run melakukan sinkronisasi pertama segera, lalu mengulang tiap Interval sampai ctx selesai.
func (s *Syncer) Run(ctx context.Context) {
	if s.Interval <= 0 {
		s.Interval = 5 * time.Second
	}
	if s.Client == nil {
		s.Client = &http.Client{Timeout: 10 * time.Second}
	}
	s.failing = make(map[string]bool)

	log.Printf("[Replica] Aktif untuk %d tabel (interval %v)", len(s.Sources), s.Interval)
	s.syncAll(ctx)

	t := time.NewTicker(s.Interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.syncAll(ctx)
		}
	}
}

func (s *Syncer) syncAll(ctx context.Context) {
	for _, src := range s.Sources {
		key := src.Table + "@" + src.URL
		n, err := s.syncTable(ctx, src)
		if err != nil {
			if !s.failing[key] { // log hanya saat mulai gagal, agar tidak membanjiri log
				log.Printf("[Replica] %s dari %s belum tersedia: %v", src.Table, src.URL, err)
			}
			s.failing[key] = true
			continue
		}
		if s.failing[key] {
			log.Printf("[Replica] %s dari %s pulih (%d baris)", src.Table, src.URL, n)
		}
		s.failing[key] = false
	}
}

func (s *Syncer) syncTable(ctx context.Context, src Source) (int, error) {
	url := strings.TrimRight(src.URL, "/") + "/internal/replica/" + src.Table
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set(TokenHeader, Token())

	resp, err := s.Client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 300))
		return 0, fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}

	var payload struct {
		Rows []map[string]interface{} `json:"rows"`
	}
	dec := json.NewDecoder(resp.Body)
	dec.UseNumber() // jaga ketelitian angka
	if err := dec.Decode(&payload); err != nil {
		return 0, err
	}
	if len(payload.Rows) == 0 {
		return 0, nil
	}

	tx, err := s.DB.BeginTxx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	for _, row := range payload.Rows {
		if err := upsert(ctx, tx, src.Table, row); err != nil {
			return 0, fmt.Errorf("upsert %s: %w", src.Table, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return len(payload.Rows), nil
}

func upsert(ctx context.Context, tx *sqlx.Tx, table string, row map[string]interface{}) error {
	cols := make([]string, 0, len(row))
	for k := range row {
		cols = append(cols, k)
	}
	sort.Strings(cols)

	quoted := make([]string, len(cols))
	marks := make([]string, len(cols))
	updates := make([]string, len(cols))
	args := make([]interface{}, len(cols))
	for i, c := range cols {
		quoted[i] = "`" + c + "`"
		marks[i] = "?"
		updates[i] = fmt.Sprintf("`%s` = VALUES(`%s`)", c, c)
		args[i] = row[c]
	}
	q := fmt.Sprintf("INSERT INTO `%s` (%s) VALUES (%s) ON DUPLICATE KEY UPDATE %s",
		table, strings.Join(quoted, ", "), strings.Join(marks, ", "), strings.Join(updates, ", "))
	_, err := tx.ExecContext(ctx, q, args...)
	return err
}
