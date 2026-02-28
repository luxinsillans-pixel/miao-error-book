-- system_setting
CREATE TABLE system_setting (
  name TEXT NOT NULL,
  value TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  UNIQUE(name)
);

-- user
CREATE TABLE user (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
  updated_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
  row_status TEXT NOT NULL CHECK (row_status IN ('NORMAL', 'ARCHIVED')) DEFAULT 'NORMAL',
  username TEXT NOT NULL UNIQUE,
  role TEXT NOT NULL DEFAULT 'USER',
  email TEXT NOT NULL DEFAULT '',
  nickname TEXT NOT NULL DEFAULT '',
  password_hash TEXT NOT NULL,
  avatar_url TEXT NOT NULL DEFAULT '',
  description TEXT NOT NULL DEFAULT ''
);

-- user_setting
CREATE TABLE user_setting (
  user_id INTEGER NOT NULL,
  key TEXT NOT NULL,
  value TEXT NOT NULL,
  UNIQUE(user_id, key)
);

-- memo
CREATE TABLE memo (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  uid TEXT NOT NULL UNIQUE,
  creator_id INTEGER NOT NULL,
  created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
  updated_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
  row_status TEXT NOT NULL CHECK (row_status IN ('NORMAL', 'ARCHIVED')) DEFAULT 'NORMAL',
  content TEXT NOT NULL DEFAULT '',
  visibility TEXT NOT NULL CHECK (visibility IN ('PUBLIC', 'PROTECTED', 'PRIVATE')) DEFAULT 'PRIVATE',
  pinned INTEGER NOT NULL CHECK (pinned IN (0, 1)) DEFAULT 0,
  payload TEXT NOT NULL DEFAULT '{}'
);

-- memo_relation
CREATE TABLE memo_relation (
  memo_id INTEGER NOT NULL,
  related_memo_id INTEGER NOT NULL,
  type TEXT NOT NULL,
  UNIQUE(memo_id, related_memo_id, type)
);

-- attachment
CREATE TABLE attachment (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  uid TEXT NOT NULL UNIQUE,
  creator_id INTEGER NOT NULL,
  created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
  updated_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
  filename TEXT NOT NULL DEFAULT '',
  blob BLOB DEFAULT NULL,
  type TEXT NOT NULL DEFAULT '',
  size INTEGER NOT NULL DEFAULT 0,
  memo_id INTEGER,
  storage_type TEXT NOT NULL DEFAULT '',
  reference TEXT NOT NULL DEFAULT '',
  payload TEXT NOT NULL DEFAULT '{}'
);

-- activity
CREATE TABLE activity (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  creator_id INTEGER NOT NULL,
  created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
  type TEXT NOT NULL DEFAULT '',
  level TEXT NOT NULL CHECK (level IN ('INFO', 'WARN', 'ERROR')) DEFAULT 'INFO',
  payload TEXT NOT NULL DEFAULT '{}'
);

-- idp
CREATE TABLE idp (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  type TEXT NOT NULL,
  identifier_filter TEXT NOT NULL DEFAULT '',
  config TEXT NOT NULL DEFAULT '{}'
);

-- inbox
CREATE TABLE inbox (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
  sender_id INTEGER NOT NULL,
  receiver_id INTEGER NOT NULL,
  status TEXT NOT NULL,
  message TEXT NOT NULL DEFAULT '{}'
);

-- reaction
CREATE TABLE reaction (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
  creator_id INTEGER NOT NULL,
  content_id TEXT NOT NULL,
  reaction_type TEXT NOT NULL,
  UNIQUE(creator_id, content_id, reaction_type)
);

-- class
CREATE TABLE class (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  uid TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  description TEXT,
  creator_id INTEGER NOT NULL,
  invite_code TEXT UNIQUE,
  visibility TEXT NOT NULL DEFAULT 'PUBLIC',
  settings TEXT DEFAULT NULL,
  created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
  updated_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
  CHECK (visibility IN ('PUBLIC', 'PROTECTED', 'PRIVATE'))
);

-- class_member
CREATE TABLE class_member (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  class_id INTEGER NOT NULL,
  user_id INTEGER NOT NULL,
  role TEXT NOT NULL DEFAULT 'STUDENT',
  joined_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
  invited_by INTEGER,
  UNIQUE (class_id, user_id),
  FOREIGN KEY (class_id) REFERENCES class(id) ON DELETE CASCADE,
  FOREIGN KEY (user_id) REFERENCES user(id) ON DELETE CASCADE,
  FOREIGN KEY (invited_by) REFERENCES user(id) ON DELETE SET NULL,
  CHECK (role IN ('TEACHER', 'ASSISTANT', 'STUDENT', 'PARENT'))
);

-- class_memo_visibility
CREATE TABLE class_memo_visibility (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  class_id INTEGER NOT NULL,
  memo_id INTEGER NOT NULL,
  visibility TEXT NOT NULL DEFAULT 'PUBLIC',
  shared_by INTEGER NOT NULL,
  shared_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
  description TEXT,
  UNIQUE (class_id, memo_id),
  FOREIGN KEY (class_id) REFERENCES class(id) ON DELETE CASCADE,
  FOREIGN KEY (memo_id) REFERENCES memo(id) ON DELETE CASCADE,
  CHECK (visibility IN ('PUBLIC', 'PROTECTED', 'PRIVATE'))
);

-- class_tag_template
CREATE TABLE class_tag_template (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  class_id INTEGER NOT NULL,
  name TEXT NOT NULL,
  color TEXT DEFAULT '#808080',
  description TEXT NOT NULL DEFAULT '',
  created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
  updated_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
  UNIQUE (class_id, name),
  FOREIGN KEY (class_id) REFERENCES class(id) ON DELETE CASCADE
);

-- Create indexes for better performance
CREATE INDEX idx_class_invite_code ON class (invite_code);
CREATE INDEX idx_class_created_ts ON class (created_ts);
CREATE INDEX idx_class_uid ON class (uid);
CREATE INDEX idx_class_creator_id ON class (creator_id);
CREATE INDEX idx_class_visibility ON class (visibility);
CREATE INDEX idx_class_member_class_id ON class_member (class_id);
CREATE INDEX idx_class_member_user_id ON class_member (user_id);
CREATE INDEX idx_class_member_role ON class_member (role);
CREATE INDEX idx_class_memo_visibility_class_id ON class_memo_visibility (class_id);
CREATE INDEX idx_class_memo_visibility_memo_id ON class_memo_visibility (memo_id);
CREATE INDEX idx_class_tag_template_class_id ON class_tag_template (class_id);