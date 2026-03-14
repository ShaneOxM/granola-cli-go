// Package storage provides storage operations for granola-cli.
// It handles SQLite database operations for meetings, embeddings, and chunks.
package storage

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/ShaneOxM/granola-cli-go/internal/embeddings"
)

func isSQLiteError(err error, msg string) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), msg)
}

// EmbeddingStats represents embedding statistics
type EmbeddingStats struct {
	Total       int
	Embedded    int
	Pending     int
	TotalChunks int
	Coverage    int
}

// InitEmbeddings adds embedding-related tables and columns
func (d *DB) InitEmbeddings() error {
	// Add embedding status columns to meetings table
	// Note: SQLite doesn't support IF NOT EXISTS on ALTER TABLE, so we catch errors
	_, err := d.db.Exec(`ALTER TABLE meetings ADD COLUMN embedding_status TEXT DEFAULT 'none'`)
	if err != nil && !isSQLiteError(err, "duplicate column") {
		return err
	}

	_, err = d.db.Exec(`ALTER TABLE meetings ADD COLUMN embedding_provider TEXT`)
	if err != nil && !isSQLiteError(err, "duplicate column") {
		return err
	}

	_, err = d.db.Exec(`ALTER TABLE meetings ADD COLUMN embedding_model TEXT`)
	if err != nil && !isSQLiteError(err, "duplicate column") {
		return err
	}

	_, err = d.db.Exec(`ALTER TABLE meetings ADD COLUMN chunk_count INTEGER DEFAULT 0`)
	if err != nil && !isSQLiteError(err, "duplicate column") {
		return err
	}

	_, err = d.db.Exec(`ALTER TABLE meetings ADD COLUMN embedded_at INTEGER`)
	if err != nil && !isSQLiteError(err, "duplicate column") {
		return err
	}

	// Create embedding_chunks table
	_, err = d.db.Exec(`
		CREATE TABLE IF NOT EXISTS embedding_chunks (
			id TEXT PRIMARY KEY,
			meeting_id TEXT NOT NULL,
			chunk_index INTEGER NOT NULL,
			chunk_text TEXT NOT NULL,
			embedding BLOB NOT NULL,
			dimensions INTEGER NOT NULL,
			provider TEXT NOT NULL,
			model TEXT NOT NULL,
			start_time TEXT,
			end_time TEXT,
			speakers TEXT,
			created_at INTEGER NOT NULL,
			FOREIGN KEY (meeting_id) REFERENCES meetings(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return err
	}

	// Create indexes
	_, err = d.db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_chunks_meeting ON embedding_chunks(meeting_id)
	`)
	if err != nil {
		return err
	}

	_, err = d.db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_chunks_provider ON embedding_chunks(provider, model)
	`)
	if err != nil {
		return err
	}

	_, err = d.db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_chunks_created ON embedding_chunks(created_at)
	`)
	if err != nil {
		return err
	}

	_, err = d.db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_meetings_embedding ON meetings(embedding_status)
	`)
	if err != nil {
		return err
	}

	return nil
}

// SaveEmbeddingChunk saves an embedding chunk to the database
func (d *DB) SaveEmbeddingChunk(chunk *EmbeddingChunk) error {
	speakersJSON, err := json.Marshal(chunk.Speakers)
	if err != nil {
		return fmt.Errorf("failed to marshal speakers: %w", err)
	}

	_, err = d.db.Exec(`
		INSERT OR REPLACE INTO embedding_chunks 
		(id, meeting_id, chunk_index, chunk_text, embedding, dimensions, provider, model, start_time, end_time, speakers, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, chunk.ID, chunk.MeetingID, chunk.ChunkIndex, chunk.ChunkText, chunk.Embedding,
		chunk.Dimensions, chunk.Provider, chunk.Model, chunk.StartTime, chunk.EndTime,
		speakersJSON, chunk.CreatedAt)

	return err
}

// GetChunksByMeeting retrieves all chunks for a meeting
func (d *DB) GetChunksByMeeting(meetingID string) ([]*EmbeddingChunk, error) {
	rows, err := d.db.Query(`
		SELECT id, meeting_id, chunk_index, chunk_text, embedding, dimensions, 
		       provider, model, start_time, end_time, speakers, created_at
		FROM embedding_chunks 
		WHERE meeting_id = ?
		ORDER BY chunk_index
	`, meetingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chunks []*EmbeddingChunk
	for rows.Next() {
		var chunk EmbeddingChunk
		var speakersJSON []byte

		err := rows.Scan(&chunk.ID, &chunk.MeetingID, &chunk.ChunkIndex, &chunk.ChunkText,
			&chunk.Embedding, &chunk.Dimensions, &chunk.Provider, &chunk.Model,
			&chunk.StartTime, &chunk.EndTime, &speakersJSON, &chunk.CreatedAt)
		if err != nil {
			return nil, err
		}

		if len(speakersJSON) > 0 {
			if err := json.Unmarshal(speakersJSON, &chunk.Speakers); err != nil {
				return nil, fmt.Errorf("failed to unmarshal speakers for chunk %s: %w", chunk.ID, err)
			}
		}

		chunks = append(chunks, &chunk)
	}

	return chunks, rows.Err()
}

// GetMeetingsNeedingEmbeddings retrieves meetings that need embeddings
func (d *DB) GetMeetingsNeedingEmbeddings(limit int) ([]*Meeting, error) {
	rows, err := d.db.Query(`
		SELECT id, title, created_at, updated_at, workspace_id, attendees, creator
		FROM meetings 
		WHERE embedding_status = 'none' OR embedding_status IS NULL
		ORDER BY created_at ASC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var meetings []*Meeting
	for rows.Next() {
		var m Meeting
		err := rows.Scan(&m.ID, &m.Title, &m.CreatedAt, &m.UpdatedAt,
			&m.WorkspaceID, &m.Attendees, &m.Creator)
		if err != nil {
			return nil, err
		}
		meetings = append(meetings, &m)
	}

	return meetings, rows.Err()
}

// UpdateMeetingEmbeddingStatus updates the embedding status for a meeting
func (d *DB) UpdateMeetingEmbeddingStatus(meetingID string, status string, provider string, model string, chunkCount int) error {
	_, err := d.db.Exec(`
		UPDATE meetings 
		SET embedding_status = ?, embedding_provider = ?, embedding_model = ?, chunk_count = ?, embedded_at = ?
		WHERE id = ?
	`, status, provider, model, chunkCount, time.Now().Unix(), meetingID)

	return err
}

// GetEmbeddingStats returns embedding statistics
func (d *DB) GetEmbeddingStats() EmbeddingStats {
	stats := EmbeddingStats{}

	// Get total count
	err := d.db.QueryRow(`SELECT COUNT(*) FROM meetings`).Scan(&stats.Total)
	if err != nil {
		return stats
	}

	// Get embedded count
	err = d.db.QueryRow(`SELECT COUNT(DISTINCT meeting_id) FROM embedding_chunks`).Scan(&stats.Embedded)
	if err != nil {
		return stats
	}

	// Get total chunks
	err = d.db.QueryRow(`SELECT COUNT(*) FROM embedding_chunks`).Scan(&stats.TotalChunks)
	if err != nil {
		return stats
	}

	stats.Pending = stats.Total - stats.Embedded
	if stats.Pending < 0 {
		stats.Pending = 0
	}

	// Calculate coverage
	if stats.Total > 0 {
		stats.Coverage = int((float64(stats.Embedded) / float64(stats.Total)) * 100)
	}

	return stats
}

// SearchChunks searches for similar embeddings
func (d *DB) SearchChunks(queryEmbedding []float32, minScore float64, limit int) ([]*EmbeddingChunk, error) {
	// TODO: Implement vector search for better performance
	// Current implementation loads all chunks into memory (O(n))
	rows, err := d.db.Query(`
		SELECT id, meeting_id, chunk_index, chunk_text, embedding, dimensions, 
		       provider, model, start_time, end_time, speakers, created_at
		FROM embedding_chunks
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var allChunks []*EmbeddingChunk
	for rows.Next() {
		var chunk EmbeddingChunk
		var speakersJSON []byte

		err := rows.Scan(&chunk.ID, &chunk.MeetingID, &chunk.ChunkIndex, &chunk.ChunkText,
			&chunk.Embedding, &chunk.Dimensions, &chunk.Provider, &chunk.Model,
			&chunk.StartTime, &chunk.EndTime, &speakersJSON, &chunk.CreatedAt)
		if err != nil {
			return nil, err
		}

		if len(speakersJSON) > 0 {
			if err := json.Unmarshal(speakersJSON, &chunk.Speakers); err != nil {
				return nil, fmt.Errorf("failed to unmarshal speakers for chunk %s: %w", chunk.ID, err)
			}
		}

		allChunks = append(allChunks, &chunk)
	}

	// Calculate similarities in memory
	// This is inefficient for large datasets but works for now
	var similarChunks []*EmbeddingChunk
	for _, chunk := range allChunks {
		embedding := embeddings.BlobToEmbedding(chunk.Embedding, chunk.Dimensions)
		score := embeddings.CosineSimilarity(queryEmbedding, embedding)
		if score >= minScore {
			chunk.Score = score
			similarChunks = append(similarChunks, chunk)
		}
	}

	// Sort by score (descending)
	sort.Slice(similarChunks, func(i, j int) bool {
		return similarChunks[i].Score > similarChunks[j].Score
	})

	// Limit results
	if len(similarChunks) > limit {
		similarChunks = similarChunks[:limit]
	}

	return similarChunks, nil
}

// ResetAllEmbeddings clears all embeddings
func (d *DB) ResetAllEmbeddings() error {
	_, err := d.db.Exec(`DELETE FROM embedding_chunks`)
	if err != nil {
		return err
	}

	_, err = d.db.Exec(`
		UPDATE meetings 
		SET embedding_status = 'none', embedding_provider = NULL, 
		    embedding_model = NULL, chunk_count = 0, embedded_at = NULL
	`)
	return err
}

// GetMeetingEmbeddingStatus returns the embedding status for a meeting
func (d *DB) GetMeetingEmbeddingStatus(meetingID string) (string, string, string, int, error) {
	var status, provider, model string
	var chunkCount int

	err := d.db.QueryRow(`
		SELECT embedding_status, embedding_provider, embedding_model, chunk_count
		FROM meetings WHERE id = ?
	`, meetingID).Scan(&status, &provider, &model, &chunkCount)

	if err != nil {
		return "", "", "", 0, err
	}

	return status, provider, model, chunkCount, nil
}

// GetMeetingTitle returns the title for a meeting ID
func (d *DB) GetMeetingTitle(meetingID string) (string, error) {
	var title string
	err := d.db.QueryRow(`
		SELECT title FROM meetings WHERE id = ?
	`, meetingID).Scan(&title)

	if err != nil {
		return "", err
	}

	return title, nil
}

// GetAllChunks retrieves all embedding chunks
func (d *DB) GetAllChunks() ([]*EmbeddingChunk, error) {
	rows, err := d.db.Query(`
		SELECT id, meeting_id, chunk_index, chunk_text, embedding, dimensions, 
		       provider, model, start_time, end_time, speakers, created_at
		FROM embedding_chunks
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chunks []*EmbeddingChunk
	for rows.Next() {
		var chunk EmbeddingChunk
		var speakersJSON []byte

		err := rows.Scan(&chunk.ID, &chunk.MeetingID, &chunk.ChunkIndex, &chunk.ChunkText,
			&chunk.Embedding, &chunk.Dimensions, &chunk.Provider, &chunk.Model,
			&chunk.StartTime, &chunk.EndTime, &speakersJSON, &chunk.CreatedAt)
		if err != nil {
			return nil, err
		}

		if len(speakersJSON) > 0 {
			if err := json.Unmarshal(speakersJSON, &chunk.Speakers); err != nil {
				return nil, fmt.Errorf("failed to unmarshal speakers for chunk %s: %w", chunk.ID, err)
			}
		}

		chunks = append(chunks, &chunk)
	}

	return chunks, rows.Err()
}

// EmbeddingChunk represents an embedding chunk from the database
type EmbeddingChunk struct {
	ID         string
	MeetingID  string
	ChunkIndex int
	ChunkText  string
	Embedding  []byte
	Dimensions int
	Provider   string
	Model      string
	StartTime  string
	EndTime    string
	Speakers   []string
	CreatedAt  int64
	Score      float64 // Used for search results
}
