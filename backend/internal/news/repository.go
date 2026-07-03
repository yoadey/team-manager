package news

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles all news-related DB operations.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new Repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

const selectNewsFields = `
	n.id, n.team_id, n.author_id, n.title, n.body, n.pinned, n.created_at,
	u.name AS author_name, u.avatar_color AS author_color,
	(u.photo_data IS NOT NULL) AS has_photo
`

func scanNews(row interface{ Scan(dest ...any) error }) (*NewsRow, error) {
	nr := &NewsRow{}
	err := row.Scan(
		&nr.Id, &nr.TeamId, &nr.AuthorId, &nr.Title, &nr.Body, &nr.Pinned, &nr.CreatedAt,
		&nr.AuthorName, &nr.AuthorColor, &nr.HasPhoto,
	)
	if err != nil {
		return nil, fmt.Errorf("scan: %w", err)
	}
	return nr, nil
}

// ListCursor is the keyset position for news pagination (matches the
// ORDER BY pinned DESC, created_at DESC, id DESC ordering).
type ListCursor struct {
	Pinned    bool      `json:"p"`
	CreatedAt time.Time `json:"c"`
	ID        uuid.UUID `json:"i"`
}

// ListByTeam returns up to limit news items for a team — pinned first, then
// newest first — starting after cur (nil = first page). It is a keyset query:
// no OFFSET, so deep pages stay fast.
func (r *Repository) ListByTeam(ctx context.Context, teamID uuid.UUID, limit int, cur *ListCursor) ([]*NewsRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	args := []any{teamID, limit}
	predicate := ""
	if cur != nil {
		predicate = "AND (n.pinned, n.created_at, n.id) < ($3, $4, $5)"
		args = append(args, cur.Pinned, cur.CreatedAt, cur.ID)
	}
	q := fmt.Sprintf(`
		SELECT %s
		FROM news n
		JOIN users u ON u.id = n.author_id
		WHERE n.team_id = $1 %s
		ORDER BY n.pinned DESC, n.created_at DESC, n.id DESC
		LIMIT $2
	`, selectNewsFields, predicate)
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("news.Repository.ListByTeam: %w", err)
	}
	defer rows.Close()

	var result []*NewsRow
	for rows.Next() {
		nr, err := scanNews(rows)
		if err != nil {
			return nil, fmt.Errorf("news.Repository.ListByTeam scan: %w", err)
		}
		result = append(result, nr)
	}
	return result, rows.Err()
}

// Create inserts a new news item and returns the enriched row.
func (r *Repository) Create(ctx context.Context, teamID, authorID uuid.UUID, title, body string, pinned bool) (*NewsRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var id uuid.UUID
	err := r.pool.QueryRow(
		ctx,
		`INSERT INTO news (team_id, author_id, title, body, pinned) VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		teamID, authorID, title, body, pinned,
	).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("news.Repository.Create: %w", err)
	}
	return r.findByID(ctx, id)
}

// Update modifies a news item that belongs to teamID and returns the enriched
// row. Returns pgx.ErrNoRows if no news item with id exists within teamID.
func (r *Repository) Update(ctx context.Context, id, teamID uuid.UUID, title, body *string, pinned *bool) (*NewsRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	tag, err := r.pool.Exec(
		ctx,
		`UPDATE news SET
			title  = COALESCE($3, title),
			body   = COALESCE($4, body),
			pinned = COALESCE($5, pinned)
		 WHERE id = $1 AND team_id = $2`,
		id, teamID, title, body, pinned,
	)
	if err != nil {
		return nil, fmt.Errorf("news.Repository.Update: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, pgx.ErrNoRows
	}
	return r.findByID(ctx, id)
}

// Delete removes a news item by ID that belongs to teamID. Returns
// pgx.ErrNoRows if no news item with id exists within teamID.
func (r *Repository) Delete(ctx context.Context, id, teamID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	tag, err := r.pool.Exec(ctx, `DELETE FROM news WHERE id = $1 AND team_id = $2`, id, teamID)
	if err != nil {
		return fmt.Errorf("news.Repository.Delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// InsertNotification creates a notification for the news item creation.
func (r *Repository) InsertNotification(ctx context.Context, teamID, actorID uuid.UUID, title string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err := r.pool.Exec(
		ctx,
		`INSERT INTO notifications (team_id, type, actor_id, status, title) VALUES ($1, 'news', $2, 'info', $3)`,
		teamID, actorID, title,
	)
	if err != nil {
		return fmt.Errorf("news.Repository.InsertNotification: %w", err)
	}
	return nil
}

func (r *Repository) findByID(ctx context.Context, id uuid.UUID) (*NewsRow, error) {
	q := fmt.Sprintf(`
		SELECT %s
		FROM news n
		JOIN users u ON u.id = n.author_id
		WHERE n.id = $1
	`, selectNewsFields)
	row := r.pool.QueryRow(ctx, q, id)
	nr, err := scanNews(row)
	if err != nil {
		return nil, fmt.Errorf("news.Repository.findByID: %w", err)
	}
	return nr, nil
}
