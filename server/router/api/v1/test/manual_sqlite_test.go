package test

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

type ClassVisibility string

const (
	ClassVisibilityPublic    ClassVisibility = "PUBLIC"
	ClassVisibilityProtected ClassVisibility = "PROTECTED"
	ClassVisibilityPrivate   ClassVisibility = "PRIVATE"
)

func TestManualSQLiteCheckConstraint(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Create table with CHECK constraint
	_, err = db.Exec(`
	CREATE TABLE class_memo_visibility (
	  id INTEGER PRIMARY KEY AUTOINCREMENT,
	  class_id INTEGER NOT NULL,
	  memo_id INTEGER NOT NULL,
	  visibility TEXT NOT NULL DEFAULT 'PUBLIC',
	  CHECK (visibility IN ('PUBLIC', 'PROTECTED', 'PRIVATE'))
	);
	`)
	require.NoError(t, err)

	// Test 1: Insert valid value "PUBLIC" using custom type
	_, err = db.Exec("INSERT INTO `class_memo_visibility` (`class_id`, `memo_id`, `visibility`) VALUES (?, ?, ?)", 1, 1, ClassVisibilityPublic)
	require.NoError(t, err, "Should succeed with ClassVisibilityPublic")

	// Test 2: Insert valid value "PROTECTED"
	_, err = db.Exec(`INSERT INTO class_memo_visibility (class_id, memo_id, visibility) VALUES (?, ?, ?)`, 1, 2, "PROTECTED")
	require.NoError(t, err, "Should succeed with PROTECTED")

	// Test 3: Insert valid value "PRIVATE"
	_, err = db.Exec(`INSERT INTO class_memo_visibility (class_id, memo_id, visibility) VALUES (?, ?, ?)`, 1, 3, "PRIVATE")
	require.NoError(t, err, "Should succeed with PRIVATE")

	// Test 4: Insert invalid value "INVALID"
	_, err = db.Exec(`INSERT INTO class_memo_visibility (class_id, memo_id, visibility) VALUES (?, ?, ?)`, 1, 4, "INVALID")
	require.Error(t, err, "Should fail with INVALID")
}
