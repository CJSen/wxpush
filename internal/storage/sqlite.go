package storage

import (
	"database/sql"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

type Message struct {
	ID         int64
	MsgID      string
	TokenID    string
	Title      string
	Content    string
	UserID     string
	TemplateID string
	BaseURL    string
	CreatedAt  time.Time
}

func NewSQLite(path string) (*Store, error) {
	if err := ensureDir(path); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if err := initSchema(db); err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
}

func ensureDir(path string) error {
	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		return nil
	}
	return os.MkdirAll(dir, 0o755)
}

func initSchema(db *sql.DB) error {
	_, err := db.Exec(`
CREATE TABLE IF NOT EXISTS messages (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  msg_id TEXT NOT NULL UNIQUE,
  token_id TEXT NOT NULL,
  title TEXT NOT NULL,
  content TEXT NOT NULL,
  userid TEXT NOT NULL,
  template_id TEXT NOT NULL,
  base_url TEXT NOT NULL,
  created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_messages_msg_token ON messages(msg_id, token_id);
`)
	return err
}

func (s *Store) InsertMessage(msg Message) error {
	_, err := s.db.Exec(`
INSERT INTO messages (msg_id, token_id, title, content, userid, template_id, base_url, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		msg.MsgID, msg.TokenID, msg.Title, msg.Content, msg.UserID, msg.TemplateID, msg.BaseURL, msg.CreatedAt.Format(time.RFC3339),
	)
	return err
}

func (s *Store) GetMessage(msgID, tokenID string) (*Message, error) {
	row := s.db.QueryRow(`
SELECT id, msg_id, token_id, title, content, userid, template_id, base_url, created_at
FROM messages
WHERE msg_id = ? AND token_id = ?`,
		msgID, tokenID,
	)
	var msg Message
	var createdAt string
	if err := row.Scan(&msg.ID, &msg.MsgID, &msg.TokenID, &msg.Title, &msg.Content, &msg.UserID, &msg.TemplateID, &msg.BaseURL, &createdAt); err != nil {
		return nil, err
	}
	if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
		msg.CreatedAt = t
	}
	return &msg, nil
}
