package mysql

import (
	"context"
	"database/sql"
	"strings"

	"github.com/pkg/errors"
	"github.com/usememos/memos/store"
)

func (d *DB) CreateClassTagTemplate(ctx context.Context, create *store.ClassTagTemplate) (*store.ClassTagTemplate, error) {
	fields := []string{"`class_id`", "`name`"}
	placeholder := []string{"?", "?"}
	args := []any{create.ClassID, create.Name}

	// Optional color field
	if create.Color != "" {
		fields = append(fields, "`color`")
		placeholder = append(placeholder, "?")
		args = append(args, create.Color)
	}
	// Description field not present in table

	stmt := "INSERT INTO `class_tag_template` (" + strings.Join(fields, ", ") + ") VALUES (" + strings.Join(placeholder, ", ") + ")"
	result, err := d.db.ExecContext(ctx, stmt, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute statement")
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get last insert id")
	}

	id32 := int32(id)
	list, err := d.ListClassTagTemplates(ctx, &store.FindClassTagTemplate{ID: &id32})
	if err != nil || len(list) == 0 {
		return nil, errors.Wrap(err, "failed to find created class tag template")
	}

	return list[0], nil
}

func (d *DB) ListClassTagTemplates(ctx context.Context, find *store.FindClassTagTemplate) ([]*store.ClassTagTemplate, error) {
	where, args := []string{"1 = 1"}, []any{}

	if find.ID != nil {
		where, args = append(where, "`id` = ?"), append(args, *find.ID)
	}
	if find.ClassID != nil {
		where, args = append(where, "`class_id` = ?"), append(args, *find.ClassID)
	}
	if find.Name != nil {
		where, args = append(where, "`name` = ?"), append(args, *find.Name)
	}
	if len(find.ClassIDList) > 0 {
		placeholders := make([]string, len(find.ClassIDList))
		for i := range find.ClassIDList {
			placeholders[i] = "?"
			args = append(args, find.ClassIDList[i])
		}
		where = append(where, "`class_id` IN ("+strings.Join(placeholders, ",")+")")
	}

	orderBy := "`created_ts` DESC"
	query := "SELECT `id`, `class_id`, `name`, `color`, UNIX_TIMESTAMP(`created_ts`), UNIX_TIMESTAMP(`updated_ts`) FROM `class_tag_template` WHERE " + strings.Join(where, " AND ") + " ORDER BY " + orderBy

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

	list := []*store.ClassTagTemplate{}
	for rows.Next() {
		template := &store.ClassTagTemplate{}
		var color sql.NullString
		if err := rows.Scan(
			&template.ID,
			&template.ClassID,
			&template.Name,
			&color,
			&template.CreatedTs,
			&template.UpdatedTs,
		); err != nil {
			return nil, err
		}
		if color.Valid {
			template.Color = color.String
		}
		// Description field not present in table
		list = append(list, template)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return list, nil
}

func (d *DB) UpdateClassTagTemplate(ctx context.Context, update *store.UpdateClassTagTemplate) error {
	set, args := []string{}, []any{}
	if update.Name != nil {
		set, args = append(set, "`name` = ?"), append(args, *update.Name)
	}
	if update.Color != nil {
		set, args = append(set, "`color` = ?"), append(args, *update.Color)
	}
	// Description field not present in table

	if len(set) == 0 {
		return errors.New("no fields to update")
	}

	args = append(args, update.ID)
	stmt := "UPDATE `class_tag_template` SET " + strings.Join(set, ", ") + ", `updated_ts` = CURRENT_TIMESTAMP WHERE `id` = ?"
	_, err := d.db.ExecContext(ctx, stmt, args...)
	if err != nil {
		return errors.Wrap(err, "failed to execute statement")
	}

	return nil
}

func (d *DB) DeleteClassTagTemplate(ctx context.Context, delete *store.DeleteClassTagTemplate) error {
	stmt := "DELETE FROM `class_tag_template` WHERE `id` = ?"
	result, err := d.db.ExecContext(ctx, stmt, delete.ID)
	if err != nil {
		return errors.Wrap(err, "failed to execute statement")
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return errors.New("class tag template not found")
	}

	return nil
}