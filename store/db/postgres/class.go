package postgres

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

	fields := []string{"uid", "name", "description", "creator_id", "visibility", "invite_code", "settings"}
	args := []any{create.UID, create.Name, create.Description, create.CreatorID, create.Visibility, create.InviteCode, settingsString}

	stmt := "INSERT INTO class (" + strings.Join(fields, ", ") + ") VALUES (" + placeholders(len(args)) + ") RETURNING id, created_ts, updated_ts"
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
		where, args = append(where, "id = "+placeholder(len(args)+1)), append(args, *find.ID)
	}
	if find.UID != nil {
		where, args = append(where, "uid = "+placeholder(len(args)+1)), append(args, *find.UID)
	}
	if len(find.UIDList) > 0 {
		placeholders := make([]string, len(find.UIDList))
		for i := range find.UIDList {
			placeholders[i] = placeholder(len(args)+1)
			args = append(args, find.UIDList[i])
		}
		where = append(where, "uid IN ("+strings.Join(placeholders, ",")+")")
	}
	if find.CreatorID != nil {
		where, args = append(where, "creator_id = "+placeholder(len(args)+1)), append(args, *find.CreatorID)
	}
	if find.Visibility != nil {
		where, args = append(where, "visibility = "+placeholder(len(args)+1)), append(args, *find.Visibility)
	}
	if find.InviteCode != nil {
		where, args = append(where, "invite_code = "+placeholder(len(args)+1)), append(args, *find.InviteCode)
	}
	if find.MemberID != nil {
		// Join with class_member table to filter classes where user is a member
		where = append(where, "id IN (SELECT class_id FROM class_member WHERE user_id = "+placeholder(len(args)+1)+")")
		args = append(args, *find.MemberID)
	}

	// Handle filters (advanced)
	for _, filter := range find.Filters {
		where = append(where, filter)
	}

	orderBy := "created_ts DESC"
	if find.OrderBy != "" {
		orderBy = find.OrderBy
	}

	query := "SELECT id, uid, name, description, creator_id, visibility, invite_code, settings, created_ts, updated_ts FROM class WHERE " + strings.Join(where, " AND ") + " ORDER BY " + orderBy

	if find.Limit != nil {
		query += " LIMIT " + placeholder(len(args)+1)
		args = append(args, *find.Limit)
	}
	if find.Offset != nil {
		query += " OFFSET " + placeholder(len(args)+1)
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
		set, args = append(set, "uid = "+placeholder(len(args)+1)), append(args, *update.UID)
	}
	if update.Name != nil {
		set, args = append(set, "name = "+placeholder(len(args)+1)), append(args, *update.Name)
	}
	if update.Description != nil {
		set, args = append(set, "description = "+placeholder(len(args)+1)), append(args, *update.Description)
	}
	if update.Visibility != nil {
		set, args = append(set, "visibility = "+placeholder(len(args)+1)), append(args, *update.Visibility)
	}
	if update.InviteCode != nil {
		set, args = append(set, "invite_code = "+placeholder(len(args)+1)), append(args, *update.InviteCode)
	}
	if update.Settings != nil {
		bytes, err := protojson.Marshal(update.Settings)
		if err != nil {
			return errors.Wrap(err, "failed to marshal class settings")
		}
		set, args = append(set, "settings = "+placeholder(len(args)+1)), append(args, string(bytes))
	}

	if len(set) == 0 {
		return errors.New("no fields to update")
	}

	args = append(args, update.ID)
	stmt := "UPDATE class SET " + strings.Join(set, ", ") + ", updated_ts = CURRENT_TIMESTAMP WHERE id = " + placeholder(len(args))
	_, err := d.db.ExecContext(ctx, stmt, args...)
	if err != nil {
		return errors.Wrap(err, "failed to execute statement")
	}

	return nil
}

func (d *DB) DeleteClass(ctx context.Context, delete *store.DeleteClass) error {
	// Delete class (foreign key constraints should handle cascade deletion if configured)
	stmt := "DELETE FROM class WHERE id = " + placeholder(1)
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

// TODO: Implement the following methods for class_member, class_memo_visibility, and class_tag_template
func (d *DB) CreateClassMember(ctx context.Context, create *store.ClassMember) (*store.ClassMember, error) {
	return nil, errors.New("not implemented")
}

func (d *DB) ListClassMembers(ctx context.Context, find *store.FindClassMember) ([]*store.ClassMember, error) {
	return nil, errors.New("not implemented")
}

func (d *DB) UpdateClassMember(ctx context.Context, update *store.UpdateClassMember) error {
	return errors.New("not implemented")
}

func (d *DB) DeleteClassMember(ctx context.Context, delete *store.DeleteClassMember) error {
	return errors.New("not implemented")
}

func (d *DB) CreateClassMemoVisibility(ctx context.Context, create *store.ClassMemoVisibility) (*store.ClassMemoVisibility, error) {
	return nil, errors.New("not implemented")
}

func (d *DB) ListClassMemoVisibilities(ctx context.Context, find *store.FindClassMemoVisibility) ([]*store.ClassMemoVisibility, error) {
	return nil, errors.New("not implemented")
}

func (d *DB) UpdateClassMemoVisibility(ctx context.Context, update *store.UpdateClassMemoVisibility) error {
	return errors.New("not implemented")
}

func (d *DB) DeleteClassMemoVisibility(ctx context.Context, delete *store.DeleteClassMemoVisibility) error {
	return errors.New("not implemented")
}

func (d *DB) CreateClassTagTemplate(ctx context.Context, create *store.ClassTagTemplate) (*store.ClassTagTemplate, error) {
	return nil, errors.New("not implemented")
}

func (d *DB) ListClassTagTemplates(ctx context.Context, find *store.FindClassTagTemplate) ([]*store.ClassTagTemplate, error) {
	return nil, errors.New("not implemented")
}

func (d *DB) UpdateClassTagTemplate(ctx context.Context, update *store.UpdateClassTagTemplate) error {
	return errors.New("not implemented")
}

func (d *DB) DeleteClassTagTemplate(ctx context.Context, delete *store.DeleteClassTagTemplate) error {
	return errors.New("not implemented")
}