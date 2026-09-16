-- Nexura Hub — spec-aligned PostgreSQL schema
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

DO $$ BEGIN CREATE TYPE user_role AS ENUM ('student', 'instructor', 'admin'); EXCEPTION WHEN duplicate_object THEN NULL; END $$;
DO $$ BEGIN CREATE TYPE user_status AS ENUM ('active', 'suspended', 'pending'); EXCEPTION WHEN duplicate_object THEN NULL; END $$;
DO $$ BEGIN CREATE TYPE creator_type AS ENUM ('instructor', 'admin'); EXCEPTION WHEN duplicate_object THEN NULL; END $$;
DO $$ BEGIN CREATE TYPE payment_status AS ENUM ('pending', 'paid', 'failed', 'refunded'); EXCEPTION WHEN duplicate_object THEN NULL; END $$;
DO $$ BEGIN CREATE TYPE payment_gateway AS ENUM ('dummy', 'stripe', 'sslcommerz', 'shurjopay'); EXCEPTION WHEN duplicate_object THEN NULL; END $$;
DO $$ BEGIN CREATE TYPE wallet_owner_type AS ENUM ('admin', 'instructor'); EXCEPTION WHEN duplicate_object THEN NULL; END $$;
DO $$ BEGIN CREATE TYPE wallet_entry_type AS ENUM ('credit', 'debit'); EXCEPTION WHEN duplicate_object THEN NULL; END $$;
DO $$ BEGIN CREATE TYPE txn_status AS ENUM ('completed', 'pending', 'refunded'); EXCEPTION WHEN duplicate_object THEN NULL; END $$;
DO $$ BEGIN CREATE TYPE discount_type AS ENUM ('percentage', 'flat'); EXCEPTION WHEN duplicate_object THEN NULL; END $$;
DO $$ BEGIN CREATE TYPE resource_type AS ENUM ('github', 'link', 'pdf', 'richtext'); EXCEPTION WHEN duplicate_object THEN NULL; END $$;
DO $$ BEGIN CREATE TYPE conversation_type AS ENUM ('direct', 'group'); EXCEPTION WHEN duplicate_object THEN NULL; END $$;
DO $$ BEGIN CREATE TYPE attachment_type AS ENUM ('image', 'file', 'code'); EXCEPTION WHEN duplicate_object THEN NULL; END $$;
DO $$ BEGIN CREATE TYPE notification_type AS ENUM ('live', 'discussion', 'quiz', 'system', 'enrollment', 'payment'); EXCEPTION WHEN duplicate_object THEN NULL; END $$;
DO $$ BEGIN CREATE TYPE live_status AS ENUM ('scheduled', 'live', 'completed', 'cancelled'); EXCEPTION WHEN duplicate_object THEN NULL; END $$;

-- Compat aliases used by older code
DO $$ BEGIN CREATE TYPE creator_type_enum AS ENUM ('instructor', 'admin'); EXCEPTION WHEN duplicate_object THEN NULL; END $$;
DO $$ BEGIN CREATE TYPE transaction_status AS ENUM ('completed', 'pending', 'refunded'); EXCEPTION WHEN duplicate_object THEN NULL; END $$;
DO $$ BEGIN CREATE TYPE chat_type AS ENUM ('direct', 'group'); EXCEPTION WHEN duplicate_object THEN NULL; END $$;

CREATE TABLE IF NOT EXISTS users (
  id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  first_name    VARCHAR(80) NOT NULL,
  last_name     VARCHAR(80) NOT NULL,
  email         VARCHAR(160) NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  role          user_role NOT NULL DEFAULT 'student',
  status        user_status NOT NULL DEFAULT 'active',
  avatar        TEXT,
  bio           TEXT,
  occupation    VARCHAR(160),
  phone         VARCHAR(40),
  website       VARCHAR(255),
  designation   VARCHAR(160),
  email_verified_at TIMESTAMPTZ,
  last_seen_at  TIMESTAMPTZ,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at    TIMESTAMPTZ
);

ALTER TABLE users ADD COLUMN IF NOT EXISTS designation VARCHAR(160);
ALTER TABLE users ADD COLUMN IF NOT EXISTS email_verified_at TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS last_seen_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);
CREATE INDEX IF NOT EXISTS idx_users_deleted ON users(deleted_at) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS refresh_tokens (
  id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash  TEXT NOT NULL,
  expires_at  TIMESTAMPTZ NOT NULL,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS password_resets (
  id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash  TEXT NOT NULL UNIQUE,
  expires_at  TIMESTAMPTZ NOT NULL,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS email_verifications (
  id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash  TEXT NOT NULL UNIQUE,
  expires_at  TIMESTAMPTZ NOT NULL,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS categories (
  id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  title      VARCHAR(120) NOT NULL,
  slug       VARCHAR(140) NOT NULL UNIQUE,
  thumbnail  TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS courses (
  id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  slug            VARCHAR(200) NOT NULL UNIQUE,
  title           VARCHAR(255) NOT NULL,
  subtitle        VARCHAR(500),
  description     TEXT,
  category_id     UUID REFERENCES categories(id),
  instructor_id   UUID NOT NULL REFERENCES users(id),
  creator_type    creator_type NOT NULL DEFAULT 'instructor',
  thumbnail       TEXT,
  price           NUMERIC(12,2) NOT NULL DEFAULT 0,
  discount_price  NUMERIC(12,2),
  is_published    BOOLEAN NOT NULL DEFAULT FALSE,
  is_featured     BOOLEAN NOT NULL DEFAULT FALSE,
  learning_points TEXT[] DEFAULT '{}',
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at      TIMESTAMPTZ
);

ALTER TABLE courses ADD COLUMN IF NOT EXISTS creator_type creator_type NOT NULL DEFAULT 'instructor';
ALTER TABLE courses ADD COLUMN IF NOT EXISTS is_featured BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS idx_courses_published ON courses(is_published) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_courses_instructor ON courses(instructor_id);
CREATE INDEX IF NOT EXISTS idx_courses_category ON courses(category_id);

CREATE TABLE IF NOT EXISTS quiz_sets (
  id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  instructor_id UUID NOT NULL REFERENCES users(id),
  title         VARCHAR(255) NOT NULL,
  description   TEXT,
  total_marks   INT NOT NULL DEFAULT 20,
  is_published  BOOLEAN NOT NULL DEFAULT FALSE,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS modules (
  id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  course_id    UUID NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
  title        VARCHAR(255) NOT NULL,
  description  TEXT,
  position     INT NOT NULL DEFAULT 0,
  is_published BOOLEAN NOT NULL DEFAULT FALSE,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_modules_course ON modules(course_id, position);

CREATE TABLE IF NOT EXISTS lessons (
  id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  module_id    UUID NOT NULL REFERENCES modules(id) ON DELETE CASCADE,
  title        VARCHAR(255) NOT NULL,
  description  TEXT,
  video_url    TEXT,
  duration     VARCHAR(20),
  is_free      BOOLEAN NOT NULL DEFAULT FALSE,
  is_published BOOLEAN NOT NULL DEFAULT FALSE,
  position     INT NOT NULL DEFAULT 0,
  quiz_set_id  UUID REFERENCES quiz_sets(id) ON DELETE SET NULL,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
ALTER TABLE lessons ADD COLUMN IF NOT EXISTS quiz_set_id UUID REFERENCES quiz_sets(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_lessons_module ON lessons(module_id, position);

CREATE TABLE IF NOT EXISTS lesson_resources (
  id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  lesson_id  UUID NOT NULL REFERENCES lessons(id) ON DELETE CASCADE,
  title      VARCHAR(255) NOT NULL,
  type       resource_type NOT NULL DEFAULT 'link',
  url        TEXT,
  size       VARCHAR(40),
  file_name  VARCHAR(255),
  content    TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS quiz_questions (
  id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  quiz_set_id UUID NOT NULL REFERENCES quiz_sets(id) ON DELETE CASCADE,
  title       VARCHAR(500) NOT NULL,
  description TEXT,
  points      INT NOT NULL DEFAULT 5,
  position    INT NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS quiz_options (
  id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  question_id UUID NOT NULL REFERENCES quiz_questions(id) ON DELETE CASCADE,
  label       VARCHAR(500) NOT NULL,
  is_correct  BOOLEAN NOT NULL DEFAULT FALSE,
  position    INT NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS coupons (
  id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  code              VARCHAR(40) NOT NULL UNIQUE,
  discount_type     discount_type NOT NULL,
  discount_value    NUMERIC(12,2) NOT NULL,
  expiry_date       DATE NOT NULL,
  max_redemptions   INT NOT NULL DEFAULT 200,
  redemption_count  INT NOT NULL DEFAULT 0,
  is_active         BOOLEAN NOT NULL DEFAULT TRUE,
  created_by        UUID REFERENCES users(id),
  created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS wallets (
  id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  owner_type   wallet_owner_type NOT NULL,
  user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  balance      NUMERIC(14,2) NOT NULL DEFAULT 0,
  currency     VARCHAR(8) NOT NULL DEFAULT 'BDT',
  created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (owner_type, user_id)
);

CREATE TABLE IF NOT EXISTS payments (
  id               UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  public_txn_id    VARCHAR(64) NOT NULL UNIQUE,
  user_id          UUID NOT NULL REFERENCES users(id),
  course_id        UUID NOT NULL REFERENCES courses(id),
  coupon_id        UUID REFERENCES coupons(id),
  original_price   NUMERIC(12,2) NOT NULL,
  discount_amount  NUMERIC(12,2) NOT NULL DEFAULT 0,
  amount_paid      NUMERIC(12,2) NOT NULL,
  gateway          payment_gateway NOT NULL DEFAULT 'dummy',
  gateway_label    VARCHAR(40),
  gateway_ref      VARCHAR(160),
  is_dummy         BOOLEAN NOT NULL DEFAULT TRUE,
  status           payment_status NOT NULL DEFAULT 'paid',
  created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  paid_at          TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS enrollments (
  id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_id         UUID NOT NULL REFERENCES users(id),
  course_id       UUID NOT NULL REFERENCES courses(id),
  payment_id      UUID REFERENCES payments(id),
  payment_status  payment_status NOT NULL DEFAULT 'paid',
  progress        NUMERIC(5,2) NOT NULL DEFAULT 0,
  enrolled_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  completed_at    TIMESTAMPTZ,
  UNIQUE (user_id, course_id)
);
CREATE INDEX IF NOT EXISTS idx_enrollments_course ON enrollments(course_id);
CREATE INDEX IF NOT EXISTS idx_enrollments_user ON enrollments(user_id);

-- Compat: older schema used student_id
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='enrollments' AND column_name='student_id')
     AND NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='enrollments' AND column_name='user_id') THEN
    ALTER TABLE enrollments RENAME COLUMN student_id TO user_id;
  END IF;
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='enrollments' AND column_name='progress_percentage')
     AND NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='enrollments' AND column_name='progress') THEN
    ALTER TABLE enrollments RENAME COLUMN progress_percentage TO progress;
  END IF;
END $$;

ALTER TABLE enrollments ADD COLUMN IF NOT EXISTS payment_id UUID REFERENCES payments(id);
ALTER TABLE enrollments ADD COLUMN IF NOT EXISTS payment_status payment_status DEFAULT 'paid';
ALTER TABLE enrollments ADD COLUMN IF NOT EXISTS progress NUMERIC(5,2) DEFAULT 0;

CREATE TABLE IF NOT EXISTS coupon_redemptions (
  id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  coupon_id    UUID NOT NULL REFERENCES coupons(id),
  user_id      UUID NOT NULL REFERENCES users(id),
  course_id    UUID NOT NULL REFERENCES courses(id),
  payment_id   UUID REFERENCES payments(id),
  created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (coupon_id, user_id, course_id)
);

CREATE TABLE IF NOT EXISTS wallet_ledger (
  id             UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  wallet_id      UUID NOT NULL REFERENCES wallets(id) ON DELETE CASCADE,
  payment_id     UUID REFERENCES payments(id),
  entry_type     wallet_entry_type NOT NULL,
  amount         NUMERIC(12,2) NOT NULL CHECK (amount > 0),
  balance_after  NUMERIC(14,2) NOT NULL,
  note           VARCHAR(255),
  created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_wallet_ledger_wallet ON wallet_ledger(wallet_id, created_at DESC);

CREATE TABLE IF NOT EXISTS transactions (
  id                       UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  payment_id               UUID REFERENCES payments(id),
  course_id                UUID NOT NULL REFERENCES courses(id),
  course_title             VARCHAR(255) NOT NULL DEFAULT '',
  creator_type             creator_type NOT NULL DEFAULT 'instructor',
  instructor_id            UUID REFERENCES users(id),
  instructor_name          VARCHAR(160) NOT NULL DEFAULT '',
  student_id               UUID NOT NULL REFERENCES users(id),
  student_name             VARCHAR(160) NOT NULL DEFAULT '',
  student_email            VARCHAR(160) NOT NULL DEFAULT '',
  price                    NUMERIC(12,2) NOT NULL DEFAULT 0,
  admin_commission_rate    NUMERIC(5,4) NOT NULL DEFAULT 0,
  admin_commission_amount  NUMERIC(12,2) NOT NULL DEFAULT 0,
  instructor_earnings      NUMERIC(12,2) NOT NULL DEFAULT 0,
  payment_method           VARCHAR(40) NOT NULL DEFAULT 'Dummy',
  status                   txn_status NOT NULL DEFAULT 'completed',
  created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
ALTER TABLE transactions ADD COLUMN IF NOT EXISTS payment_id UUID REFERENCES payments(id);
ALTER TABLE transactions ADD COLUMN IF NOT EXISTS course_title VARCHAR(255) DEFAULT '';
ALTER TABLE transactions ADD COLUMN IF NOT EXISTS instructor_name VARCHAR(160) DEFAULT '';
ALTER TABLE transactions ADD COLUMN IF NOT EXISTS student_name VARCHAR(160) DEFAULT '';
ALTER TABLE transactions ADD COLUMN IF NOT EXISTS student_email VARCHAR(160) DEFAULT '';
ALTER TABLE transactions ADD COLUMN IF NOT EXISTS price NUMERIC(12,2) DEFAULT 0;
CREATE INDEX IF NOT EXISTS idx_txn_created ON transactions(created_at DESC);

CREATE TABLE IF NOT EXISTS quiz_attempts (
  id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  quiz_set_id UUID NOT NULL REFERENCES quiz_sets(id),
  lesson_id   UUID REFERENCES lessons(id),
  user_id     UUID NOT NULL REFERENCES users(id),
  course_id   UUID REFERENCES courses(id),
  answers     JSONB NOT NULL,
  score       NUMERIC(8,2) NOT NULL,
  total       NUMERIC(8,2) NOT NULL,
  passed      BOOLEAN NOT NULL,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS uniq_quiz_pass ON quiz_attempts(user_id, lesson_id) WHERE passed = TRUE;

CREATE TABLE IF NOT EXISTS lesson_progress (
  id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_id      UUID NOT NULL REFERENCES users(id),
  lesson_id    UUID NOT NULL REFERENCES lessons(id) ON DELETE CASCADE,
  course_id    UUID NOT NULL REFERENCES courses(id),
  completed    BOOLEAN NOT NULL DEFAULT FALSE,
  last_position_sec INT NOT NULL DEFAULT 0,
  completed_at TIMESTAMPTZ,
  UNIQUE (user_id, lesson_id)
);
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='lesson_progress' AND column_name='student_id')
     AND NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='lesson_progress' AND column_name='user_id') THEN
    ALTER TABLE lesson_progress RENAME COLUMN student_id TO user_id;
  END IF;
END $$;
ALTER TABLE lesson_progress ADD COLUMN IF NOT EXISTS course_id UUID REFERENCES courses(id);
ALTER TABLE lesson_progress ADD COLUMN IF NOT EXISTS last_position_sec INT NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS lesson_notes (
  id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_id       UUID NOT NULL REFERENCES users(id),
  lesson_id     UUID NOT NULL REFERENCES lessons(id) ON DELETE CASCADE,
  timestamp_sec INT NOT NULL DEFAULT 0,
  text          TEXT NOT NULL,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS discussions (
  id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  lesson_id  UUID NOT NULL REFERENCES lessons(id) ON DELETE CASCADE,
  course_id  UUID NOT NULL REFERENCES courses(id),
  user_id    UUID NOT NULL REFERENCES users(id),
  content    TEXT NOT NULL,
  upvotes    INT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS discussion_replies (
  id             UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  discussion_id  UUID NOT NULL REFERENCES discussions(id) ON DELETE CASCADE,
  user_id        UUID NOT NULL REFERENCES users(id),
  content        TEXT NOT NULL,
  created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS discussion_votes (
  discussion_id UUID NOT NULL REFERENCES discussions(id) ON DELETE CASCADE,
  user_id       UUID NOT NULL REFERENCES users(id),
  PRIMARY KEY (discussion_id, user_id)
);

CREATE TABLE IF NOT EXISTS reviews (
  id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  course_id  UUID NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
  user_id    UUID NOT NULL REFERENCES users(id),
  rating     SMALLINT NOT NULL CHECK (rating BETWEEN 1 AND 5),
  comment    TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (course_id, user_id)
);

CREATE TABLE IF NOT EXISTS live_classes (
  id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  instructor_id UUID NOT NULL REFERENCES users(id),
  course_id     UUID REFERENCES courses(id) ON DELETE SET NULL,
  title         VARCHAR(255) NOT NULL,
  description   TEXT,
  date          VARCHAR(40) NOT NULL,
  time          VARCHAR(40) NOT NULL,
  starts_at     TIMESTAMPTZ,
  duration      VARCHAR(20),
  meeting_link  TEXT,
  is_completed  BOOLEAN NOT NULL DEFAULT FALSE,
  status        live_status NOT NULL DEFAULT 'scheduled',
  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS certificates (
  id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  public_id       VARCHAR(32) NOT NULL UNIQUE,
  user_id         UUID NOT NULL REFERENCES users(id),
  course_id       UUID NOT NULL REFERENCES courses(id),
  student_name    VARCHAR(160) NOT NULL,
  course_title    VARCHAR(255) NOT NULL,
  issued_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (user_id, course_id)
);

CREATE TABLE IF NOT EXISTS notifications (
  id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  type       notification_type NOT NULL DEFAULT 'system',
  title      VARCHAR(255) NOT NULL,
  message    TEXT NOT NULL,
  is_read    BOOLEAN NOT NULL DEFAULT FALSE,
  link       VARCHAR(255),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_notif_user ON notifications(user_id, is_read, created_at DESC);

CREATE TABLE IF NOT EXISTS conversations (
  id             UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  type           conversation_type NOT NULL,
  name           VARCHAR(255) NOT NULL,
  avatar         TEXT,
  course_id      UUID REFERENCES courses(id) ON DELETE CASCADE,
  instructor_id  UUID REFERENCES users(id),
  created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS uniq_course_group ON conversations(course_id) WHERE type = 'group' AND course_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS conversation_members (
  conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
  user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  unread_count    INT NOT NULL DEFAULT 0,
  last_read_at    TIMESTAMPTZ,
  joined_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (conversation_id, user_id)
);
ALTER TABLE conversation_members ADD COLUMN IF NOT EXISTS unread_count INT NOT NULL DEFAULT 0;
ALTER TABLE conversation_members ADD COLUMN IF NOT EXISTS last_read_at TIMESTAMPTZ;

CREATE TABLE IF NOT EXISTS messages (
  id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
  sender_id       UUID NOT NULL REFERENCES users(id),
  content         TEXT NOT NULL DEFAULT '',
  image_url       TEXT,
  reply_to_id     UUID REFERENCES messages(id) ON DELETE SET NULL,
  attachment_type attachment_type,
  attachment_url  TEXT,
  attachment_name VARCHAR(255),
  is_deleted      BOOLEAN NOT NULL DEFAULT FALSE,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_messages_conv ON messages(conversation_id, created_at);

-- Compat view for old chat_messages name
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'chat_messages')
     AND NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'messages') THEN
    ALTER TABLE chat_messages RENAME TO messages;
  END IF;
END $$;

CREATE TABLE IF NOT EXISTS message_reactions (
  message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
  user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  emoji      VARCHAR(16) NOT NULL,
  PRIMARY KEY (message_id, user_id, emoji)
);

CREATE TABLE IF NOT EXISTS direct_pairs (
  conversation_id UUID PRIMARY KEY REFERENCES conversations(id) ON DELETE CASCADE,
  user_a          UUID NOT NULL REFERENCES users(id),
  user_b          UUID NOT NULL REFERENCES users(id),
  UNIQUE (user_a, user_b),
  CHECK (user_a < user_b)
);
