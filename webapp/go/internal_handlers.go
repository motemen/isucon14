package main

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"os"
)

// このAPIをインスタンス内から一定間隔で叩かせることで、椅子とライドをマッチングさせる (のは昔の話)
func internalGetMatching(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func doMatching() error {
	hostname, err := os.Hostname()
	if err != nil {
		return err
	}
	if hostname != "ip-192-168-0-11" {
		return nil
	}

	for {
		if err := matching(); err != nil {
			slog.Error("matching failed", slog.Any("error", err))
		}
		// time.Sleep(500 * time.Millisecond)
	}
}

func matching() error {
	ctx := context.Background()
	// MEMO: 一旦最も待たせているリクエストに適当な空いている椅子マッチさせる実装とする。おそらくもっといい方法があるはず…
	ride := &Ride{}
	if err := db.GetContext(ctx, ride, `SELECT * FROM rides WHERE chair_id IS NULL ORDER BY id LIMIT 1`); err != nil {

		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}

	matched := &Chair{}
	if err := db.GetContext(ctx, matched,
		`SELECT * FROM chairs WHERE is_active = TRUE AND is_occupied = FALSE LIMIT 1`); err != nil {

		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}

	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, "UPDATE rides SET chair_id = ? WHERE id = ?", matched.ID, ride.ID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "UPDATE chairs SET is_occupied = TRUE WHERE id = ?", matched.ID); err != nil {
		return err
	}
	return tx.Commit()
}
