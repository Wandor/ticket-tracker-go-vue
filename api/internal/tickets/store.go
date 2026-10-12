package tickets

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound is returned when a ticket id does not exist.
var ErrNotFound = errors.New("ticket not found")

const columns = "id, title, description, status, priority, created_at"

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) Create(ctx context.Context, t NewTicket) (Ticket, error) {
	row := s.pool.QueryRow(ctx,
		`INSERT INTO tickets (title, description, status, priority)
		 VALUES ($1, $2, $3, $4)
		 RETURNING `+columns,
		t.Title, t.Description, t.Status, t.Priority)
	return scanTicket(row)
}

func (s *Store) List(ctx context.Context, p ListParams) ([]Ticket, int, error) {
	// A NULL status parameter disables the filter, so one statement covers both cases.
	const filter = "WHERE ($1::text IS NULL OR status = $1)"

	var total int
	if err := s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM tickets "+filter, p.Status).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := s.pool.Query(ctx,
		"SELECT "+columns+" FROM tickets "+filter+
			" ORDER BY created_at DESC, id DESC LIMIT $2 OFFSET $3",
		p.Status, p.Limit, p.Offset())
	if err != nil {
		return nil, 0, err
	}
	tickets, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Ticket, error) {
		return scanTicket(row)
	})
	return tickets, total, err
}

func (s *Store) Update(ctx context.Context, id int64, r UpdateRequest) (Ticket, error) {
	// COALESCE keeps the current value for any field the request omitted.
	row := s.pool.QueryRow(ctx,
		`UPDATE tickets
		    SET status = COALESCE($2, status),
		        priority = COALESCE($3, priority)
		  WHERE id = $1
		  RETURNING `+columns,
		id, r.Status, r.Priority)
	t, err := scanTicket(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Ticket{}, ErrNotFound
	}
	return t, err
}

func (s *Store) CountByStatus(ctx context.Context) (map[Status]int, error) {
	counts := make(map[Status]int, len(Statuses))
	for _, st := range Statuses {
		counts[st] = 0
	}
	rows, err := s.pool.Query(ctx, "SELECT status, COUNT(*) FROM tickets GROUP BY status")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var st Status
		var n int
		if err := rows.Scan(&st, &n); err != nil {
			return nil, err
		}
		counts[st] = n
	}
	return counts, rows.Err()
}

func scanTicket(row pgx.Row) (Ticket, error) {
	var t Ticket
	err := row.Scan(&t.ID, &t.Title, &t.Description, &t.Status, &t.Priority, &t.CreatedAt)
	t.CreatedAt = t.CreatedAt.UTC()
	return t, err
}
