-- Create class tables for miao-error-book feature
-- This migration adds support for class management, member roles, memo visibility control, and tag templates.

-- Class table: stores class information
CREATE TABLE IF NOT EXISTS class (
  id SERIAL PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  description TEXT,
  invite_code VARCHAR(50) UNIQUE NOT NULL,
  settings JSONB DEFAULT NULL,
  created_ts TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_ts TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Class member table: stores membership and role information
CREATE TABLE IF NOT EXISTS class_member (
  id SERIAL PRIMARY KEY,
  class_id INTEGER NOT NULL REFERENCES class(id) ON DELETE CASCADE,
  user_id INTEGER NOT NULL REFERENCES user(id) ON DELETE CASCADE,
  role VARCHAR(20) NOT NULL DEFAULT 'STUDENT',
  joined_ts TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE (class_id, user_id),
  CHECK (role IN ('TEACHER', 'ASSISTANT', 'STUDENT', 'PARENT'))
);

-- Class memo visibility table: controls memo visibility within a class
CREATE TABLE IF NOT EXISTS class_memo_visibility (
  id SERIAL PRIMARY KEY,
  class_id INTEGER NOT NULL REFERENCES class(id) ON DELETE CASCADE,
  memo_id INTEGER NOT NULL REFERENCES memo(id) ON DELETE CASCADE,
  visibility VARCHAR(20) NOT NULL DEFAULT 'PUBLIC',
  created_ts TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_ts TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE (class_id, memo_id),
  CHECK (visibility IN ('PUBLIC', 'ANONYMOUS', 'TEACHER_ONLY'))
);

-- Class tag template table: predefined tags for a class
CREATE TABLE IF NOT EXISTS class_tag_template (
  id SERIAL PRIMARY KEY,
  class_id INTEGER NOT NULL REFERENCES class(id) ON DELETE CASCADE,
  name VARCHAR(255) NOT NULL,
  color VARCHAR(20) DEFAULT '#808080',
  created_ts TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_ts TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE (class_id, name)
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