package repository

import (
	"context"
	"database/sql"
	"fmt"

	"challenge/internal/models"
)

// Handles persistence for pack sizes using a standard
// database/sql connection via pgx driver
type PackRepository struct {
	db *sql.DB
}

func New(db *sql.DB) *PackRepository {
	return &PackRepository{db: db}
}

// Returns all pack sizes as plain integers, ordered descending
// (largest first). 
func (r *PackRepository) ListSizes(ctx context.Context) ([]int, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT size FROM packs ORDER BY size DESC")
	if err != nil {
		return nil, fmt.Errorf("query pack sizes: %w", err)
	}
	defer rows.Close()

	var sizes []int
	for rows.Next() {
		var size int
		if err := rows.Scan(&size); err != nil {
			return nil, fmt.Errorf("scan pack size: %w", err)
		}
		sizes = append(sizes, size)
	}
	return sizes, rows.Err()
}

// Returns all pack records ordered descending by size. 
func (r *PackRepository) List(ctx context.Context) ([]models.Pack, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT id, size, created_at FROM packs ORDER BY size DESC")
	if err != nil {
		return nil, fmt.Errorf("query packs: %w", err)
	}
	defer rows.Close()

	var packs []models.Pack
	for rows.Next() {
		var p models.Pack
		if err := rows.Scan(&p.ID, &p.Size, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan pack: %w", err)
		}
		packs = append(packs, p)
	}
	return packs, rows.Err()
}

// Removes all existing pack sizes and inserts the provided
// ones in a single tx. 
func (r *PackRepository) Replace(ctx context.Context, sizes []int) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, "DELETE FROM packs"); err != nil {
		return fmt.Errorf("delete packs: %w", err)
	}

	for _, size := range sizes {
		if _, err := tx.ExecContext(ctx, "INSERT INTO packs (size) VALUES ($1)", size); err != nil {
			return fmt.Errorf("insert pack %d: %w", size, err)
		}
	}

	return tx.Commit()
}
