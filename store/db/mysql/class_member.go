package mysql

import (
	"context"
	"strings"

	"github.com/pkg/errors"
	"github.com/usememos/memos/store"
)

func (d *DB) CreateClassMember(ctx context.Context, create *store.ClassMember) (*store.ClassMember, error) {
	fields := []string{"`class_id`", "`user_id`", "`role`"}
	placeholder := []string{"?", "?", "?"}
	args := []any{create.ClassID, create.UserID, create.Role}

	stmt := "INSERT INTO `class_member` (" + strings.Join(fields, ", ") + ") VALUES (" + strings.Join(placeholder, ", ") + ")"
	result, err := d.db.ExecContext(ctx, stmt, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute statement")
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get last insert id")
	}

	id32 := int32(id)
	list, err := d.ListClassMembers(ctx, &store.FindClassMember{ID: &id32})
	if err != nil || len(list) == 0 {
		return nil, errors.Wrap(err, "failed to find created class member")
	}

	return list[0], nil
}

func (d *DB) ListClassMembers(ctx context.Context, find *store.FindClassMember) ([]*store.ClassMember, error) {
	where, args := []string{"1 = 1"}, []any{}

	if find.ID != nil {
		where, args = append(where, "`id` = ?"), append(args, *find.ID)
	}
	if find.ClassID != nil {
		where, args = append(where, "`class_id` = ?"), append(args, *find.ClassID)
	}
	if find.UserID != nil {
		where, args = append(where, "`user_id` = ?"), append(args, *find.UserID)
	}
	if find.Role != nil {
		where, args = append(where, "`role` = ?"), append(args, *find.Role)
	}
	if len(find.ClassIDList) > 0 {
		placeholders := make([]string, len(find.ClassIDList))
		for i := range find.ClassIDList {
			placeholders[i] = "?"
			args = append(args, find.ClassIDList[i])
		}
		where = append(where, "`class_id` IN ("+strings.Join(placeholders, ",")+")")
	}
	if len(find.UserIDList) > 0 {
		placeholders := make([]string, len(find.UserIDList))
		for i := range find.UserIDList {
			placeholders[i] = "?"
			args = append(args, find.UserIDList[i])
		}
		where = append(where, "`user_id` IN ("+strings.Join(placeholders, ",")+")")
	}

	orderBy := "`joined_ts` DESC"
	query := "SELECT `id`, `class_id`, `user_id`, `role`, UNIX_TIMESTAMP(`joined_ts`) FROM `class_member` WHERE " + strings.Join(where, " AND ") + " ORDER BY " + orderBy

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

	list := []*store.ClassMember{}
	for rows.Next() {
		classMember := &store.ClassMember{}
		if err := rows.Scan(
			&classMember.ID,
			&classMember.ClassID,
			&classMember.UserID,
			&classMember.Role,
			&classMember.JoinedTs,
		); err != nil {
			return nil, err
		}
		list = append(list, classMember)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return list, nil
}

func (d *DB) UpdateClassMember(ctx context.Context, update *store.UpdateClassMember) error {
	set, args := []string{}, []any{}
	if update.Role != nil {
		set, args = append(set, "`role` = ?"), append(args, *update.Role)
	}

	if len(set) == 0 {
		return errors.New("no fields to update")
	}

	args = append(args, update.ID)
	stmt := "UPDATE `class_member` SET " + strings.Join(set, ", ") + " WHERE `id` = ?"
	_, err := d.db.ExecContext(ctx, stmt, args...)
	if err != nil {
		return errors.Wrap(err, "failed to execute statement")
	}

	return nil
}

func (d *DB) DeleteClassMember(ctx context.Context, delete *store.DeleteClassMember) error {
	stmt := "DELETE FROM `class_member` WHERE `id` = ?"
	result, err := d.db.ExecContext(ctx, stmt, delete.ID)
	if err != nil {
		return errors.Wrap(err, "failed to execute statement")
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return errors.New("class member not found")
	}

	return nil
}