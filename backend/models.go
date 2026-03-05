package main

import "time"

type Message struct {
	ID        string    `json:"id"`
	AgentID   string    `json:"agent_id"`
	AgentName string    `json:"agent_name"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	ReplyTo   *string   `json:"reply_to,omitempty"`
}

type AgentInfo struct {
	Slug  string `json:"slug"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

type StatusResponse struct {
	IsRunning       bool        `json:"is_running"`
	MessageCount    int         `json:"message_count"`
	Agents          []AgentInfo `json:"agents"`
	CooldownSeconds int         `json:"cooldown_seconds"`
}

type PostMessageRequest struct {
	Content string `json:"content"`
}

type ErrorResponse struct {
	Error      string `json:"error"`
	RetryAfter int    `json:"retry_after,omitempty"`
}

var agentList = []AgentInfo{
	{Slug: "diogenes", Name: "Diogenes", Color: "#E8A838"},
	{Slug: "hypatia", Name: "Hypatia", Color: "#7EB8DA"},
	{Slug: "tesla", Name: "Tesla", Color: "#B088F9"},
	{Slug: "curie", Name: "Marie Curie", Color: "#5DE8A0"},
	{Slug: "cioran", Name: "Cioran", Color: "#F25C54"},
	{Slug: "turing", Name: "Turing", Color: "#6EC8C8"},
	{Slug: "ada", Name: "Ada Lovelace", Color: "#F2A2C0"},
	{Slug: "camus", Name: "Camus", Color: "#D4D4D4"},
	{Slug: "sagan", Name: "Carl Sagan", Color: "#4A90D9"},
	{Slug: "hawking", Name: "Stephen Hawking", Color: "#1CA3EC"},
	{Slug: "jung", Name: "Carl Jung", Color: "#C77DBA"},
	{Slug: "freud", Name: "Sigmund Freud", Color: "#D4A574"},
	{Slug: "lynch", Name: "David Lynch", Color: "#E84040"},
	{Slug: "dali", Name: "Salvador Dalí", Color: "#FFD700"},
}
