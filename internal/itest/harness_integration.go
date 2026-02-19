//go:build integration

package itest

import (
	"context"
	"database/sql"
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
)

type Harness struct {
	PostgresURL string
	RedisAddr   string
	NATSURL     string
	ids         []string
}

func RequireDocker(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("integration skipped: docker CLI not found")
	}
	if err := exec.Command("docker", "info").Run(); err != nil {
		t.Skipf("integration skipped: docker unavailable: %v", err)
	}
}

func Start(t *testing.T) *Harness {
	t.Helper()
	RequireDocker(t)
	h := &Harness{}
	run := func(args ...string) string {
		out, err := exec.Command("docker", args...).CombinedOutput()
		if err != nil {
			t.Skipf("integration skipped: docker %v failed: %v: %s", args, err, string(out))
		}
		return strings.TrimSpace(string(out))
	}
	pg := run("run", "-d", "-e", "POSTGRES_PASSWORD=postgres", "-e", "POSTGRES_USER=postgres", "-e", "POSTGRES_DB=pcgb", "-p", "54329:5432", "postgres:16-alpine")
	rd := run("run", "-d", "-p", "6389:6379", "redis:7-alpine")
	nats := run("run", "-d", "-p", "42229:4222", "nats:2.10-alpine", "-js")
	h.ids = []string{pg, rd, nats}
	h.PostgresURL = "postgres://postgres:postgres@127.0.0.1:54329/pcgb?sslmode=disable"
	h.RedisAddr = "127.0.0.1:6389"
	h.NATSURL = "nats://127.0.0.1:42229"

	t.Cleanup(func() {
		for _, id := range h.ids {
			_ = exec.Command("docker", "rm", "-f", id).Run()
		}
	})

	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		db, err := sql.Open("pgx", h.PostgresURL)
		if err == nil && db.Ping() == nil {
			_ = db.Close()
			break
		}
		time.Sleep(300 * time.Millisecond)
	}
	return h
}

func RunSQL(t *testing.T, dsn string, sqlText string) {
	t.Helper()
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, stmt := range strings.Split(sqlText, ";") {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("exec sql %q: %v", stmt, err)
		}
	}
}

func Redis(t *testing.T, addr string) *redis.Client {
	t.Helper()
	r := redis.NewClient(&redis.Options{Addr: addr})
	t.Cleanup(func() { _ = r.Close() })
	return r
}
func NATS(t *testing.T, url string) *nats.Conn {
	t.Helper()
	nc, err := nats.Connect(url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(nc.Close)
	return nc
}
func WaitContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 10*time.Second)
}
func DSN(port int) string {
	return fmt.Sprintf("postgres://postgres:postgres@127.0.0.1:%d/pcgb?sslmode=disable", port)
}
