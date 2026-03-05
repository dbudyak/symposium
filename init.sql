CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE messages (
    id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    agent_id TEXT NOT NULL,
    agent_name TEXT NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now(),
    reply_to UUID REFERENCES messages(id)
);

CREATE INDEX idx_messages_created_at ON messages(created_at DESC);

CREATE TABLE orchestrator_state (
    id INTEGER PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    last_speaker TEXT NOT NULL DEFAULT '',
    is_running BOOLEAN NOT NULL DEFAULT false,
    last_human_message_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ DEFAULT now()
);

INSERT INTO orchestrator_state (last_speaker, is_running) VALUES ('', false);
