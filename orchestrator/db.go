package main

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Message struct {
	ID        string
	AgentID   string
	AgentName string
	Content   string
	CreatedAt time.Time
	ReplyTo   *string
}

type OrchestratorState struct {
	LastSpeaker        string
	IsRunning          bool
	LastHumanMessageAt *time.Time
}

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

func (d *DB) GetRecentMessages(ctx context.Context, limit int) ([]Message, error) {
	query := `SELECT id, agent_id, agent_name, content, created_at, reply_to
		FROM messages ORDER BY created_at DESC LIMIT $1`
	rows, err := d.pool.Query(ctx, query, limit)
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
	// Reverse so oldest first for context building
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	return msgs, nil
}

func (d *DB) HasNewHumanMessage(ctx context.Context, since time.Time) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM messages WHERE agent_id = 'human' AND created_at > $1)`
	var exists bool
	err := d.pool.QueryRow(ctx, query, since).Scan(&exists)
	return exists, err
}

func (d *DB) InsertMessage(ctx context.Context, agentID, agentName, content string) error {
	query := `INSERT INTO messages (agent_id, agent_name, content) VALUES ($1, $2, $3)`
	_, err := d.pool.Exec(ctx, query, agentID, agentName, content)
	return err
}

func (d *DB) GetState(ctx context.Context) (*OrchestratorState, error) {
	query := `SELECT last_speaker, is_running, last_human_message_at FROM orchestrator_state WHERE id = 1`
	var state OrchestratorState
	err := d.pool.QueryRow(ctx, query).Scan(&state.LastSpeaker, &state.IsRunning, &state.LastHumanMessageAt)
	if err == pgx.ErrNoRows {
		return &OrchestratorState{}, nil
	}
	return &state, err
}

func (d *DB) UpdateState(ctx context.Context, lastSpeaker string, isRunning bool) error {
	query := `UPDATE orchestrator_state SET last_speaker = $1, is_running = $2, updated_at = now() WHERE id = 1`
	_, err := d.pool.Exec(ctx, query, lastSpeaker, isRunning)
	return err
}
