package mysql

import (
	"context"
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

	stmt := "INSERT INTO `class` (" + strings.Join(fields, ", ") + ") VALUES (" + strings.Join(placeholder, ", ") + ")"
	result, err := d.db.ExecContext(ctx, stmt, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute statement")
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get last insert id")
	}

	id32 := int32(id)
	list, err := d.ListClasses(ctx, &store.FindClass{ID: &id32})
	if err != nil || len(list) == 0 {
		return nil, errors.Wrap(err, "failed to find created class")
	}

	return list[0], nil
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

	query := "SELECT `id`, `uid`, `name`, `description`, `creator_id`, `visibility`, `invite_code`, `settings`, UNIX_TIMESTAMP(`created_ts`), UNIX_TIMESTAMP(`updated_ts`) FROM `class` WHERE " + strings.Join(where, " AND ") + " ORDER BY " + orderBy

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