//go:build integration

package sessions

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/paul-cloud-game-backend/paul-cloud-game-backend/internal/contracts"
	"github.com/paul-cloud-game-backend/paul-cloud-game-backend/internal/itest"
)

func TestMatchedEventCreatesSession(t *testing.T) {
	h := itest.Start(t)
	sqlRaw, _ := os.ReadFile("../../deploy/sql/migrations/001_init.sql")
	itest.RunSQL(t, h.PostgresURL, string(sqlRaw))

	db, err := sql.Open("pgx", h.PostgresURL)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	nc := itest.NATS(t, h.NATSURL)

	repo := NewPostgresRepository(db)
	svc := NewService(repo, nil, nc, nil)
	sub, err := nc.Subscribe(contracts.SubjectMatchmakingMatch, svc.HandleMatchedEvent)
	if err != nil {
		t.Fatal(err)
	}
	defer sub.Unsubscribe()

	raw, err := contracts.MarshalV1("evt-1", contracts.EventMatchmakingMatched, time.Now().UTC(), "corr-1", nil, contracts.MatchmakingMatchedV1{MatchID: "m1", UserIDs: []string{"u1", "u2"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := nc.Publish(contracts.SubjectMatchmakingMatch, raw); err != nil {
		t.Fatal(err)
	}
	_ = nc.Flush()

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		var count int
		if err := db.QueryRowContext(context.Background(), "select count(*) from sessions").Scan(&count); err == nil && count > 0 {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("expected sessions row created")
}
