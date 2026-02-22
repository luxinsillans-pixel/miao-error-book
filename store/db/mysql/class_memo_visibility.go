package mysql

import (
	"context"
	"database/sql"
	"strings"

	"github.com/pkg/errors"
	"github.com/usememos/memos/store"
)

func (d *DB) CreateClassMemoVisibility(ctx context.Context, create *store.ClassMemoVisibility) (*store.ClassMemoVisibility, error) {
	fields := []string{"`class_id`", "`memo_id`", "`visibility`"}
	placeholder := []string{"?", "?", "?"}
	args := []any{create.ClassID, create.MemoID, create.Visibility}

	// Optional description field
	if create.Description != "" {
		fields = append(fields, "`description`")
		placeholder = append(placeholder, "?")
		args = append(args, create.Description)
	}

	stmt := "INSERT INTO `class_memo_visibility` (" + strings.Join(fields, ", ") + ") VALUES (" + strings.Join(placeholder, ", ") + ")"
	result, err := d.db.ExecContext(ctx, stmt, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute statement")
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get last insert id")
	}

	id32 := int32(id)
	list, err := d.ListClassMemoVisibilities(ctx, &store.FindClassMemoVisibility{ID: &id32})
	if err != nil || len(list) == 0 {
		return nil, errors.Wrap(err, "failed to find created class memo visibility")
	}

	return list[0], nil
}

func (d *DB) ListClassMemoVisibilities(ctx context.Context, find *store.FindClassMemoVisibility) ([]*store.ClassMemoVisibility, error) {
	where, args := []string{"1 = 1"}, []any{}

	if find.ID != nil {
		where, args = append(where, "`id` = ?"), append(args, *find.ID)
	}
	if find.ClassID != nil {
		where, args = append(where, "`class_id` = ?"), append(args, *find.ClassID)
	}
	if find.MemoID != nil {
		where, args = append(where, "`memo_id` = ?"), append(args, *find.MemoID)
	}
	if find.UserID != nil {
		// Filter by user who shared (shared_by column)
		where = append(where, "`shared_by` = ?")
		args = append(args, *find.UserID)
	}

	orderBy := "`created_ts` DESC"
	query := "SELECT `id`, `class_id`, `memo_id`, `visibility`, `shared_by`, UNIX_TIMESTAMP(`shared_ts`), `description` FROM `class_memo_visibility` WHERE " + strings.Join(where, " AND ") + " ORDER BY " + orderBy

	if find.Limit != nil {
		query += " LIMIT ?"
		args = append(args, *find.Limit)
	}
	if find.Offset != nil {
		query += " OFFSET ?"
		args = append(args, *find.Offset)
	}

	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := []*store.ClassMemoVisibility{}
	for rows.Next() {
		record := &store.ClassMemoVisibility{}
		var sharedBy sql.NullInt32
		var description sql.NullString
		if err := rows.Scan(
			&record.ID,
			&record.ClassID,
			&record.MemoID,
			&record.Visibility,
			&sharedBy,
			&record.SharedTs,
			&description,
		); err != nil {
			return nil, err
		}
		if sharedBy.Valid {
			record.SharedBy = sharedBy.Int32
		}
		if description.Valid {
			record.Description = description.String
		}
		list = append(list, record)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return list, nil
}

func (d *DB) UpdateClassMemoVisibility(ctx context.Context, update *store.UpdateClassMemoVisibility) error {
	set, args := []string{}, []any{}
	if update.Visibility != nil {
		set, args = append(set, "`visibility` = ?"), append(args, *update.Visibility)
	}
	if update.Description != nil {
		set, args = append(set, "`description` = ?"), append(args, *update.Description)
	}

	if len(set) == 0 {
		return errors.New("no fields to update")
	}

	args = append(args, update.ID)
	stmt := "UPDATE `class_memo_visibility` SET " + strings.Join(set, ", ") + ", `updated_ts` = CURRENT_TIMESTAMP WHERE `id` = ?"
	_, err := d.db.ExecContext(ctx, stmt, args...)
	if err != nil {
		return errors.Wrap(err, "failed to execute statement")
	}

	return nil
}

func (d *DB) DeleteClassMemoVisibility(ctx context.Context, delete *store.DeleteClassMemoVisibility) error {
	stmt := "DELETE FROM `class_memo_visibility` WHERE `id` = ?"
	result, err := d.db.ExecContext(ctx, stmt, delete.ID)
	if err != nil {
		return errors.Wrap(err, "failed to execute statement")
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return errors.New("class memo visibility not found")
	}

	return nil
}