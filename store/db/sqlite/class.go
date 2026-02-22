package sqlite

import (
	"context"
	"database/sql"
	"strings"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	storepb "github.com/usememos/memos/proto/gen/store"
	"github.com/usememos/memos/store"
)

func (d *DB) CreateClass(ctx context.Context, create *store.Class) (*store.Class, error) {
	settingsString := "{}"
	if create.Settings != nil {
		bytes, err := protojson.Marshal(create.Settings)
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal class settings")
		}
		settingsString = string(bytes)
	}

	fields := []string{"`uid`", "`name`", "`description`", "`creator_id`", "`visibility`", "`invite_code`", "`settings`"}
	placeholder := []string{"?", "?", "?", "?", "?", "?", "?"}
	args := []any{create.UID, create.Name, create.Description, create.CreatorID, create.Visibility, create.InviteCode, settingsString}

	stmt := "INSERT INTO class (" + strings.Join(fields, ", ") + ") VALUES (" + strings.Join(placeholder, ", ") + ") RETURNING `id`, `created_ts`, `updated_ts`"
	if err := d.db.QueryRowContext(ctx, stmt, args...).Scan(
		&create.ID,
		&create.CreatedTs,
		&create.UpdatedTs,
	); err != nil {
		return nil, errors.Wrap(err, "failed to execute statement")
	}

	return create, nil
}

func (d *DB) ListClasses(ctx context.Context, find *store.FindClass) ([]*store.Class, error) {
	where, args := []string{"1 = 1"}, []any{}

	if find.ID != nil {
		where, args = append(where, "`id` = ?"), append(args, *find.ID)
	}
	if find.UID != nil {
		where, args = append(where, "`uid` = ?"), append(args, *find.UID)
	}
	if len(find.UIDList) > 0 {
		placeholders := make([]string, len(find.UIDList))
		for i := range find.UIDList {
			placeholders[i] = "?"
			args = append(args, find.UIDList[i])
		}
		where = append(where, "`uid` IN ("+strings.Join(placeholders, ",")+")")
	}
	if find.CreatorID != nil {
		where, args = append(where, "`creator_id` = ?"), append(args, *find.CreatorID)
	}
	if find.Visibility != nil {
		where, args = append(where, "`visibility` = ?"), append(args, *find.Visibility)
	}
	if find.InviteCode != nil {
		where, args = append(where, "`invite_code` = ?"), append(args, *find.InviteCode)
	}
	if find.MemberID != nil {
		// Join with class_member table to filter classes where user is a member
		where = append(where, "`id` IN (SELECT `class_id` FROM `class_member` WHERE `user_id` = ?)")
		args = append(args, *find.MemberID)
	}

	// Handle filters (advanced)
	for _, filter := range find.Filters {
		where = append(where, filter)
	}

	orderBy := "`created_ts` DESC"
	if find.OrderBy != "" {
		orderBy = find.OrderBy
	}

	query := "SELECT `id`, `uid`, `name`, `description`, `creator_id`, `visibility`, `invite_code`, `settings`, `created_ts`, `updated_ts` FROM `class` WHERE " + strings.Join(where, " AND ") + " ORDER BY " + orderBy

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

	list := []*store.Class{}
	for rows.Next() {
		class := &store.Class{}
		var settingsBytes []byte
		if err := rows.Scan(
			&class.ID,
			&class.UID,
			&class.Name,
			&class.Description,
			&class.CreatorID,
			&class.Visibility,
			&class.InviteCode,
			&settingsBytes,
			&class.CreatedTs,
			&class.UpdatedTs,
		); err != nil {
			return nil, err
		}

		if len(settingsBytes) > 0 && string(settingsBytes) != "{}" {
			settings := &storepb.ClassSettings{}
			if err := protojsonUnmarshaler.Unmarshal(settingsBytes, settings); err != nil {
				return nil, errors.Wrap(err, "failed to unmarshal class settings")
			}
			class.Settings = settings
		}

		list = append(list, class)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return list, nil
}

func (d *DB) UpdateClass(ctx context.Context, update *store.UpdateClass) error {
	set, args := []string{}, []any{}
	if update.UID != nil {
		set, args = append(set, "`uid` = ?"), append(args, *update.UID)
	}
	if update.Name != nil {
		set, args = append(set, "`name` = ?"), append(args, *update.Name)
	}
	if update.Description != nil {
		set, args = append(set, "`description` = ?"), append(args, *update.Description)
	}
	if update.Visibility != nil {
		set, args = append(set, "`visibility` = ?"), append(args, *update.Visibility)
	}
	if update.InviteCode != nil {
		set, args = append(set, "`invite_code` = ?"), append(args, *update.InviteCode)
	}
	if update.Settings != nil {
		bytes, err := protojson.Marshal(update.Settings)
		if err != nil {
			return errors.Wrap(err, "failed to marshal class settings")
		}
		set, args = append(set, "`settings` = ?"), append(args, string(bytes))
	}

	if len(set) == 0 {
		return errors.New("no fields to update")
	}

	args = append(args, update.ID)
	stmt := "UPDATE `class` SET " + strings.Join(set, ", ") + ", `updated_ts` = CURRENT_TIMESTAMP WHERE `id` = ?"
	_, err := d.db.ExecContext(ctx, stmt, args...)
	if err != nil {
		return errors.Wrap(err, "failed to execute statement")
	}

	return nil
}

func (d *DB) DeleteClass(ctx context.Context, delete *store.DeleteClass) error {
	// Delete class (foreign key constraints should handle cascade deletion if configured)
	stmt := "DELETE FROM `class` WHERE `id` = ?"
	result, err := d.db.ExecContext(ctx, stmt, delete.ID)
	if err != nil {
		return errors.Wrap(err, "failed to execute statement")
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return errors.New("class not found")
	}

	return nil
}

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
		// invited_by column not present in table, leave as nil
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