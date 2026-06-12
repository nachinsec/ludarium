package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
)

func (s *Store) ListBingoBoards(ctx context.Context, userID int64) ([]BingoBoard, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, title, visibility, data, updated_at
		FROM bingo_boards WHERE user_id = ?
		ORDER BY updated_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []BingoBoard
	for rows.Next() {
		b, err := scanBingo(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (s *Store) GetBingoBoard(ctx context.Context, userID, id int64) (*BingoBoard, error) {
	b, err := scanBingo(s.db.QueryRowContext(ctx, `
		SELECT id, title, visibility, data, updated_at
		FROM bingo_boards WHERE id = ? AND user_id = ?`, id, userID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func (s *Store) CreateBingoBoard(ctx context.Context, userID int64, title string, data json.RawMessage) (*BingoBoard, error) {
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO bingo_boards (user_id, title, data) VALUES (?, ?, ?)`,
		userID, title, string(data))
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return s.GetBingoBoard(ctx, userID, id)
}

func (s *Store) UpdateBingoBoard(ctx context.Context, userID, id int64, title string, data json.RawMessage, visibility string) error {
	res, err := s.db.ExecContext(ctx, `
		UPDATE bingo_boards SET title = ?, data = ?, visibility = ?, updated_at = datetime('now')
		WHERE id = ? AND user_id = ?`, title, string(data), visibility, id, userID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) DeleteBingoBoard(ctx context.Context, userID, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM bingo_boards WHERE id = ? AND user_id = ?`, id, userID)
	return err
}

func scanBingo(sc interface{ Scan(...any) error }) (BingoBoard, error) {
	var b BingoBoard
	var data string
	if err := sc.Scan(&b.ID, &b.Title, &b.Visibility, &data, &b.UpdatedAt); err != nil {
		return b, err
	}
	b.Data = json.RawMessage(data)
	return b, nil
}
