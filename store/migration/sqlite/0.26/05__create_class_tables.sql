-- Create class tables for miao-error-book feature
-- This migration adds support for class management, member roles, memo visibility control, and tag templates.

-- Class table: stores class information
CREATE TABLE IF NOT EXISTS class (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  description TEXT,
  invite_code TEXT UNIQUE NOT NULL,
  settings TEXT DEFAULT NULL,
  created_ts TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_ts TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Class member table: stores membership and role information
CREATE TABLE IF NOT EXISTS class_member (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  class_id INTEGER NOT NULL,
  user_id INTEGER NOT NULL,
  role TEXT NOT NULL DEFAULT 'STUDENT',
  joined_ts TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE (class_id, user_id),
  FOREIGN KEY (class_id) REFERENCES class(id) ON DELETE CASCADE,
  FOREIGN KEY (user_id) REFERENCES user(id) ON DELETE CASCADE,
  CHECK (role IN ('TEACHER', 'ASSISTANT', 'STUDENT', 'PARENT'))
);

-- Class memo visibility table: controls memo visibility within a class
CREATE TABLE IF NOT EXISTS class_memo_visibility (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  class_id INTEGER NOT NULL,
  memo_id INTEGER NOT NULL,
  visibility TEXT NOT NULL DEFAULT 'PUBLIC',
  created_ts TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_ts TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE (class_id, memo_id),
  FOREIGN KEY (class_id) REFERENCES class(id) ON DELETE CASCADE,
  FOREIGN KEY (memo_id) REFERENCES memo(id) ON DELETE CASCADE,
  CHECK (visibility IN ('PUBLIC', 'ANONYMOUS', 'TEACHER_ONLY'))
);

-- Class tag template table: predefined tags for a class
CREATE TABLE IF NOT EXISTS class_tag_template (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  class_id INTEGER NOT NULL,
  name TEXT NOT NULL,
  color TEXT DEFAULT '#808080',
  created_ts TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_ts TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE (class_id, name),
  FOREIGN KEY (class_id) REFERENCES class(id) ON DELETE CASCADE
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_class_invite_code ON class (invite_code);
CREATE INDEX IF NOT EXISTS idx_class_created_ts ON class (created_ts);
CREATE INDEX IF NOT EXISTS idx_class_member_class_id ON class_member (class_id);
CREATE INDEX IF NOT EXISTS idx_class_member_user_id ON class_member (user_id);
CREATE INDEX IF NOT EXISTS idx_class_member_role ON class_member (role);
CREATE INDEX IF NOT EXISTS idx_class_memo_visibility_class_id ON class_memo_visibility (class_id);
CREATE INDEX IF NOT EXISTS idx_class_memo_visibility_memo_id ON class_memo_visibility (memo_id);
CREATE INDEX IF NOT EXISTS idx_class_tag_template_class_id ON class_tag_template (class_id);