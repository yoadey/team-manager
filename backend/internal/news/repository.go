package news

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yoadey/team-manager/backend/internal/db/gen"
)

// Repository handles all news-related DB operations.
type Repository struct {
	pool *pgxpool.Pool
	q    *dbgen.Queries
}

// NewRepository creates a new Repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool, q: dbgen.New(pool)}
}

// ListCursor is the keyset position for news pagination (matches the
// ORDER BY pinned DESC, created_at DESC, id DESC ordering).
type ListCursor struct {
	Pinned    bool      `json:"p"`
	CreatedAt time.Time `json:"c"`
	ID        uuid.UUID `json:"i"`
}

func toNewsRow(id, teamID, authorID uuid.UUID, title, body string, pinned bool, createdAt time.Time, authorName, authorColor string, hasPhoto bool) *NewsRow {
	return &NewsRow{
		Id:          id,
		TeamId:      teamID,
		AuthorId:    authorID,
		Title:       title,
		Body:        body,
		Pinned:      pinned,
		CreatedAt:   createdAt,
		AuthorName:  &authorName,
		AuthorColor: &authorColor,
		HasPhoto:    hasPhoto,
	}
}

// ListByTeam returns up to limit news items for a team — pinned first, then
// newest first — starting after cur (nil = first page). It is a keyset query:
// no OFFSET, so deep pages stay fast.
func (r *Repository) ListByTeam(ctx context.Context, teamID uuid.UUID, limit int, cur *ListCursor) ([]*NewsRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	params := dbgen.ListNewsByTeamParams{TeamID: teamID, Limit: int32(limit)} //nolint:gosec // G115: limit is always pre-clamped to pagination.MaxLimit (500) by callers before reaching the repository
	if cur != nil {
		params.CursorID = &cur.ID
		params.CursorPinned = pgtype.Bool{Bool: cur.Pinned, Valid: true}
		params.CursorCreatedAt = pgtype.Timestamptz{Time: cur.CreatedAt, Valid: true}
	}
	rows, err := r.q.ListNewsByTeam(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("news.Repository.ListByTeam: %w", err)
	}

	result := make([]*NewsRow, 0, len(rows))
	for _, row := range rows {
		result = append(result, toNewsRow(row.ID, row.TeamID, row.AuthorID, row.Title, row.Body, row.Pinned, row.CreatedAt, row.AuthorName, row.AuthorColor, row.HasPhoto))
	}
	return result, nil
}

// CountByTeam returns the number of news items the team has, used to enforce
// maxNewsPerTeam before an insert.
func (r *Repository) CountByTeam(ctx context.Context, teamID uuid.UUID) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	count, err := r.q.CountNewsByTeam(ctx, teamID)
	if err != nil {
		return 0, fmt.Errorf("news.Repository.CountByTeam: %w", err)
	}
	return int(count), nil
}

// Create inserts a new news item and returns the enriched row.
//
// Known, accepted tradeoff: if the INSERT commits but the findByID reload
// right after it fails (a transient DB/network blip -- there's no
// concurrent-delete race to guard against here, unlike the finances package's
// otherwise-similar reload-after-write pattern), that error surfaces as a
// generic 500 even though the news item now exists. A client that retries
// after seeing that 500 will create a duplicate -- there's no idempotency
// key or client-supplied ID to detect it. Left unfixed since the failure
// window (a second query on the same pool right after a successful insert)
// is narrow and this hasn't been a real deployment issue.
func (r *Repository) Create(ctx context.Context, teamID, authorID uuid.UUID, title, body string, pinned bool) (*NewsRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	id, err := r.q.CreateNews(ctx, dbgen.CreateNewsParams{
		TeamID: teamID, AuthorID: authorID, Title: title, Body: body, Pinned: pinned,
	})
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
	n, err := r.q.UpdateNews(ctx, dbgen.UpdateNewsParams{
		ID:     id,
		TeamID: teamID,
		Title:  optionalText(title),
		Body:   optionalText(body),
		Pinned: optionalBool(pinned),
	})
	if err != nil {
		return nil, fmt.Errorf("news.Repository.Update: %w", err)
	}
	if n == 0 {
		return nil, pgx.ErrNoRows
	}
	return r.findByID(ctx, id)
}

// Delete removes a news item by ID that belongs to teamID. Returns
// pgx.ErrNoRows if no news item with id exists within teamID.
func (r *Repository) Delete(ctx context.Context, id, teamID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	n, err := r.q.DeleteNews(ctx, dbgen.DeleteNewsParams{ID: id, TeamID: teamID})
	if err != nil {
		return fmt.Errorf("news.Repository.Delete: %w", err)
	}
	if n == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *Repository) findByID(ctx context.Context, id uuid.UUID) (*NewsRow, error) {
	row, err := r.q.GetNewsByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("news.Repository.findByID: %w", err)
	}
	return toNewsRow(row.ID, row.TeamID, row.AuthorID, row.Title, row.Body, row.Pinned, row.CreatedAt, row.AuthorName, row.AuthorColor, row.HasPhoto), nil
}

// optionalText converts a nullable string patch field into the pgtype.Text a
// generated UPDATE ... COALESCE query expects (Valid: false leaves the
// column unchanged, matching the previous hand-written COALESCE($n, col)).
func optionalText(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *s, Valid: true}
}

// optionalBool converts a nullable bool patch field into the pgtype.Bool a
// generated UPDATE ... COALESCE query expects.
func optionalBool(b *bool) pgtype.Bool {
	if b == nil {
		return pgtype.Bool{}
	}
	return pgtype.Bool{Bool: *b, Valid: true}
}
