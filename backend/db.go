package main

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DB struct {
	pool *pgxpool.Pool
}

func NewDB(ctx context.Context, databaseURL string) (*DB, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return &DB{pool: pool}, nil
}

func (d *DB) Close() {
	d.pool.Close()
}

func (d *DB) GetMessages(ctx context.Context, limit int, before string) ([]Message, error) {
	var query string
	var args []interface{}

	if before != "" {
		query = `SELECT id, agent_id, agent_name, content, created_at, reply_to
			FROM messages WHERE created_at < (SELECT created_at FROM messages WHERE id = $1)
			ORDER BY created_at DESC LIMIT $2`
		args = []interface{}{before, limit}
	} else {
		query = `SELECT id, agent_id, agent_name, content, created_at, reply_to
			FROM messages ORDER BY created_at DESC LIMIT $1`
		args = []interface{}{limit}
	}

	rows, err := d.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query messages: %w", err)
	}
	defer rows.Close()

	var msgs []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.AgentID, &m.AgentName, &m.Content, &m.CreatedAt, &m.ReplyTo); err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		msgs = append(msgs, m)
	}
	return msgs, nil
}

func (d *DB) GetMessagesSince(ctx context.Context, afterID string) ([]Message, error) {
	query := `SELECT id, agent_id, agent_name, content, created_at, reply_to
		FROM messages WHERE created_at > (SELECT created_at FROM messages WHERE id = $1)
		ORDER BY created_at ASC`

	rows, err := d.pool.Query(ctx, query, afterID)
	if err != nil {
		return nil, fmt.Errorf("query messages since: %w", err)
	}
	defer rows.Close()

	var msgs []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.AgentID, &m.AgentName, &m.Content, &m.CreatedAt, &m.ReplyTo); err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		msgs = append(msgs, m)
	}
	return msgs, nil
}

func (d *DB) InsertHumanMessage(ctx context.Context, content string) (*Message, error) {
	var m Message
	query := `INSERT INTO messages (agent_id, agent_name, content) VALUES ('human', 'Human', $1)
		RETURNING id, agent_id, agent_name, content, created_at, reply_to`
	err := d.pool.QueryRow(ctx, query, content).Scan(&m.ID, &m.AgentID, &m.AgentName, &m.Content, &m.CreatedAt, &m.ReplyTo)
	if err != nil {
		return nil, fmt.Errorf("insert human message: %w", err)
	}

	// Update last_human_message_at for cooldown tracking
	_, err = d.pool.Exec(ctx, `UPDATE orchestrator_state SET last_human_message_at = now() WHERE id = 1`)
	if err != nil {
		return nil, fmt.Errorf("update cooldown: %w", err)
	}

	return &m, nil
}

func (d *DB) GetCooldownRemaining(ctx context.Context) (int, error) {
	var lastHuman *time.Time
	err := d.pool.QueryRow(ctx, `SELECT last_human_message_at FROM orchestrator_state WHERE id = 1`).Scan(&lastHuman)
	if err != nil {
		return 0, err
	}
	if lastHuman == nil {
		return 0, nil
	}
	remaining := time.Hour - time.Since(*lastHuman)
	if remaining <= 0 {
		return 0, nil
	}
	return int(remaining.Seconds()), nil
}

func (d *DB) GetMessageCount(ctx context.Context) (int, error) {
	var count int
	err := d.pool.QueryRow(ctx, `SELECT COUNT(*) FROM messages`).Scan(&count)
	return count, err
}

func (d *DB) GetOrchestratorRunning(ctx context.Context) (bool, error) {
	var running bool
	err := d.pool.QueryRow(ctx, `SELECT is_running FROM orchestrator_state WHERE id = 1`).Scan(&running)
	return running, err
}
