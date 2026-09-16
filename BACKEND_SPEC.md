# Nexura Hub — Backend API & PostgreSQL Specification

> **Audience:** Backend team  
> **Frontend:** React 19 + TypeScript (this repo)  
> **Database:** PostgreSQL  
> **Realtime:** Socket.IO  
> **API version:** `/api/v1`  
> **Source of truth:** `src/` pages, types, services, and Socket client

This document is generated from a full frontend analysis. Implement every endpoint, payload, and Socket event listed here so the existing UI can connect without frontend changes.

---

## 0. How to use this file

1. Create PostgreSQL tables from **Section 4**.
2. Implement **Auth** first (Section 6).
3. Implement **Chat REST + Socket.IO** next (Section 7) — frontend already emits these events.
4. Then follow **page-by-page** (Section 8) so every screen has data.
5. Match **response wrapper** and **JWT header** exactly (Sections 2–3).
6. Payments are **dummy only** (Section 9): `POST /payments/dummy` credits the **admin wallet**. No Stripe/SSLCommerz/Shurjopay.

---

## 1. System overview

Nexura Hub is an LMS with 3 roles:

| Role | Frontend routes | Can chat? |
|---|---|---|
| `student` | Public site, `/account/*`, `/player/*`, `/messages` | Yes |
| `instructor` | Everything a student can + `/dashboard/*` | Yes |
| `admin` | `/admin/*` + instructor studio `/dashboard/*` | **No** (frontend disconnects socket) |

### Commission model (hard business rule)

| Course creator | Student pays | Admin gets | Instructor gets |
|---|---|---|---|
| `instructor` | 100% of sale price (after coupon) | **5%** | **95%** |
| `admin` | 100% of sale price (after coupon) | **100%** | **0%** |

- `totalRevenue` / GMV = sum of all completed student payments.
- `adminNetCommission` = 5% of instructor-course sales + 100% of admin-course sales.
- `instructorPayouts` = 95% of instructor-course sales.

Currency is BDT (`৳`). Store money as `NUMERIC(12,2)`.

### Dummy payment (v1 — no real gateway)

**Do not integrate Stripe, SSLCommerz, or Shurjopay.** Checkout is a **dummy POST** to the backend.

- Student pays by calling `POST /payments/dummy`.
- Backend instantly marks the payment `paid` (no card, no redirect, no webhook).
- Money is credited to **dummy wallets** in PostgreSQL:
  - **Admin wallet** always receives the platform cut (5% instructor courses, 100% admin courses).
  - **Instructor wallet** receives 95% on instructor-created courses.
- Admin Revenue page reads the same ledger — dummy taka shows as real GMV/commission inside the app only.
- Frontend may still send `gateway: stripe|sslcommerz|shurjopay` as a **label for the invoice**. Ignore it for processing; always treat as `dummy`.

See **Section 9** for the full dummy payment contract.

### Login redirects (frontend already does this)

| Role | After login | After register |
|---|---|---|
| `admin` | `/admin` | — (admin cannot self-register) |
| `instructor` | `/dashboard` | `/dashboard` |
| `student` | `/account/enrolled-courses` | `/courses` |

---

## 2. API conventions

### Base URL

```
https://<host>/api/v1
```

Frontend default: `VITE_API_URL` or `https://nexura-hub-backend.onrender.com/api/v1`

Socket URL (no `/api/v1`):

```
VITE_SOCKET_URL || https://nexura-hub-backend.onrender.com
```

### Auth header

Every protected request:

```
Authorization: Bearer <access_token>
```

Access token cookie name on frontend: `nexurahub_token` (7 days).  
Refresh token cookie: `nexurahub_refresh_token` (30 days).  
Also return tokens in JSON so frontend can store them.

### Standard JSON envelope

Frontend unwraps `response.data.data` first, then `response.data`.

**Success**

```json
{
  "success": true,
  "statusCode": 200,
  "message": "OK",
  "data": {}
}
```

**List (either shape is accepted; prefer this)**

```json
{
  "success": true,
  "statusCode": 200,
  "message": "OK",
  "data": [],
  "meta": {
    "page": 1,
    "limit": 20,
    "total": 120,
    "totalPages": 6
  }
}
```

Frontend also accepts list wrappers: `{ courses: [] }`, `{ categories: [] }`, `{ lives: [] }`, `{ quizSets: [] }`, `{ enrollments: [] }`, `{ items: [] }`, `{ result: [] }`.

**Error**

```json
{
  "success": false,
  "statusCode": 400,
  "message": "Human readable error",
  "error": {
    "code": "VALIDATION_ERROR",
    "details": []
  }
}
```

Use HTTP status:

| Status | When |
|---|---|
| 200 / 201 | Success |
| 400 | Validation |
| 401 | Missing/expired token — frontend clears cookies and redirects to `/login` |
| 403 | Authenticated but wrong role |
| 404 | Not found |
| 409 | Duplicate (email, coupon code) |
| 422 | Business rule (already enrolled, coupon expired, quiz fail) |
| 500 | Server error — frontend shows toast |

### IDs

Frontend types are `string | number`. Use **UUID** (`uuid`) in PostgreSQL and return as string.

### Pagination / query params

Common query keys used or expected:

```
?page=1&limit=20&search=&sort=createdAt:desc
&category=&status=published|draft
&role=student|instructor|admin
&creatorType=instructor|admin
```

### File uploads

`multipart/form-data`, field name: `file`

Responses:

```json
{ "success": true, "data": { "url": "https://cdn.../file.jpg", "size": "2.4 MB" } }
```

---

## 3. Role-based access (RBAC)

Implement middleware: `auth()` + `restrictTo('admin'|'instructor'|'student')`.

| Capability | student | instructor | admin |
|---|---|---|---|
| Browse published courses | ✓ | ✓ | ✓ |
| Buy / enroll | ✓ | ✓ | ✓ |
| Watch enrolled lessons | ✓ | ✓ | ✓ |
| Lesson notes, Q&A, reviews, certificate | ✓ | ✓ | ✓ |
| Direct + group chat | ✓ | ✓ | ✗ |
| Create/edit own courses, modules, lessons | ✗ | ✓ (own) | ✓ (all) |
| Live classes | ✗ | ✓ (own) | ✓ (all) |
| Quiz sets | ✗ | ✓ (own) | ✓ (all) |
| Instructor analytics | ✗ | ✓ (own) | ✓ (all / own studio) |
| Platform users, GMV, coupons CRUD | ✗ | ✗ | ✓ |
| Change user role / suspend | ✗ | ✗ | ✓ |
| Feature / unpublish any course | ✗ | own publish only | ✓ |
| Create admin-owned courses | ✗ | ✗ | ✓ (`creatorType=admin`) |

**Registration:** public form only allows `student` or `instructor`. Never accept `admin` from `/auth/register`. Seed the first admin manually.

**Suspended users:** `status=suspended` → reject login with 403.

---

## 4. PostgreSQL schema

Enable extensions:

```sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
```

### 4.1 Enums

```sql
CREATE TYPE user_role AS ENUM ('student', 'instructor', 'admin');
CREATE TYPE user_status AS ENUM ('active', 'suspended', 'pending');
CREATE TYPE creator_type AS ENUM ('instructor', 'admin');
CREATE TYPE payment_status AS ENUM ('pending', 'paid', 'failed', 'refunded');
CREATE TYPE payment_gateway AS ENUM ('dummy', 'stripe', 'sslcommerz', 'shurjopay');
CREATE TYPE wallet_owner_type AS ENUM ('admin', 'instructor');
CREATE TYPE wallet_entry_type AS ENUM ('credit', 'debit');
CREATE TYPE txn_status AS ENUM ('completed', 'pending', 'refunded');
CREATE TYPE discount_type AS ENUM ('percentage', 'flat');
CREATE TYPE resource_type AS ENUM ('github', 'link', 'pdf', 'richtext');
CREATE TYPE conversation_type AS ENUM ('direct', 'group');
CREATE TYPE attachment_type AS ENUM ('image', 'file', 'code');
CREATE TYPE notification_type AS ENUM ('live', 'discussion', 'quiz', 'system', 'enrollment', 'payment');
CREATE TYPE live_status AS ENUM ('scheduled', 'live', 'completed', 'cancelled');
```

### 4.2 Tables

#### `users`

```sql
CREATE TABLE users (
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
  designation   VARCHAR(160),          -- instructor public title
  email_verified_at TIMESTAMPTZ,
  last_seen_at  TIMESTAMPTZ,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at    TIMESTAMPTZ
);
```

JSON shape frontend `User`:

```json
{
  "id": "uuid",
  "firstName": "Rahim",
  "lastName": "Ahmed",
  "email": "rahim@example.com",
  "role": "student",
  "avatar": "https://...",
  "bio": "...",
  "occupation": "Software Engineer",
  "phone": "+8801...",
  "website": "https://..."
}
```

#### `refresh_tokens`

```sql
CREATE TABLE refresh_tokens (
  id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash  TEXT NOT NULL,
  expires_at  TIMESTAMPTZ NOT NULL,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

#### `categories`

```sql
CREATE TABLE categories (
  id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  title      VARCHAR(120) NOT NULL,
  slug       VARCHAR(140) NOT NULL UNIQUE,   -- frontend `value`
  thumbnail  TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

JSON:

```json
{
  "id": "uuid",
  "title": "Web Development",
  "thumbnail": "https://...",
  "value": "web-development",
  "label": "Web Development"
}
```

Seed at least: Web Development, Design, Data Science, Mobile, Marketing, Business, Photography, Music.

#### `courses`

```sql
CREATE TABLE courses (
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

CREATE INDEX idx_courses_published ON courses(is_published) WHERE deleted_at IS NULL;
CREATE INDEX idx_courses_instructor ON courses(instructor_id);
CREATE INDEX idx_courses_category ON courses(category_id);
```

JSON `Course` (list + detail):

```json
{
  "id": "uuid",
  "slug": "reactive-accelerator",
  "title": "Reactive Accelerator",
  "subtitle": "...",
  "description": "HTML or markdown",
  "category": "Web Development",
  "categoryId": "uuid",
  "thumbnail": "https://...",
  "image": "https://...",
  "price": 4999,
  "discountPrice": 3999,
  "isPublished": true,
  "isFeatured": true,
  "totalChapters": 12,
  "progress": 40,
  "creatorType": "instructor",
  "learningPoints": ["Build production-grade apps", "..."],
  "instructor": {
    "id": "uuid",
    "name": "Tapas Adhikary",
    "designation": "Senior Software Engineer",
    "avatar": "https://...",
    "bio": "...",
    "rating": 4.9,
    "studentsCount": 3400,
    "coursesCount": 10,
    "reviewsCount": 1250
  },
  "modules": [],
  "quizSets": [],
  "reviews": []
}
```

Admin course list extra computed fields (can be on list items):

```json
{
  "enrollmentsCount": 120,
  "totalRevenue": 599880,
  "adminEarnings": 29994
}
```

#### `modules`

```sql
CREATE TABLE modules (
  id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  course_id    UUID NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
  title        VARCHAR(255) NOT NULL,
  description  TEXT,
  position     INT NOT NULL DEFAULT 0,
  is_published BOOLEAN NOT NULL DEFAULT FALSE,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_modules_course ON modules(course_id, position);
```

#### `lessons`

```sql
CREATE TABLE lessons (
  id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  module_id    UUID NOT NULL REFERENCES modules(id) ON DELETE CASCADE,
  title        VARCHAR(255) NOT NULL,
  description  TEXT,
  video_url    TEXT,                 -- HLS .m3u8 or mp4 / youtube embed
  duration     VARCHAR(20),          -- "12:40"
  is_free      BOOLEAN NOT NULL DEFAULT FALSE,
  is_published BOOLEAN NOT NULL DEFAULT FALSE,
  position     INT NOT NULL DEFAULT 0,
  quiz_set_id  UUID REFERENCES quiz_sets(id) ON DELETE SET NULL,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_lessons_module ON lessons(module_id, position);
```

Public course detail: hide `video_url` of paid unpublished-preview lessons unless `is_free=true` or user is enrolled.

JSON `Lesson`:

```json
{
  "id": "uuid",
  "title": "Course Overview",
  "description": "...",
  "videoUrl": "https://cdn.../lesson.m3u8",
  "duration": "12:40",
  "isFree": true,
  "isPublished": true,
  "position": 1,
  "completed": false,
  "quizSetId": "uuid",
  "quizSetTitle": "React Fundamentals Assessment",
  "hasQuiz": true,
  "questions": [],
  "resources": []
}
```

#### `lesson_resources`

```sql
CREATE TABLE lesson_resources (
  id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  lesson_id  UUID NOT NULL REFERENCES lessons(id) ON DELETE CASCADE,
  title      VARCHAR(255) NOT NULL,
  type       resource_type NOT NULL DEFAULT 'link',
  url        TEXT,
  size       VARCHAR(40),
  file_name  VARCHAR(255),
  content    TEXT,                   -- richtext notes
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

#### `quiz_sets`

```sql
CREATE TABLE quiz_sets (
  id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  instructor_id UUID NOT NULL REFERENCES users(id),
  title        VARCHAR(255) NOT NULL,
  description  TEXT,
  total_marks  INT NOT NULL DEFAULT 20,
  is_published BOOLEAN NOT NULL DEFAULT FALSE,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

#### `quiz_questions`

```sql
CREATE TABLE quiz_questions (
  id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  quiz_set_id UUID NOT NULL REFERENCES quiz_sets(id) ON DELETE CASCADE,
  title       VARCHAR(500) NOT NULL,
  description TEXT,
  points      INT NOT NULL DEFAULT 5,
  position    INT NOT NULL DEFAULT 0
);
```

#### `quiz_options`

```sql
CREATE TABLE quiz_options (
  id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  question_id UUID NOT NULL REFERENCES quiz_questions(id) ON DELETE CASCADE,
  label       VARCHAR(500) NOT NULL,
  is_correct  BOOLEAN NOT NULL DEFAULT FALSE,
  position    INT NOT NULL DEFAULT 0
);
```

Never send `isCorrect` to students until after submit. Instructor/admin may see it.

#### `quiz_attempts`

```sql
CREATE TABLE quiz_attempts (
  id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  quiz_set_id UUID NOT NULL REFERENCES quiz_sets(id),
  lesson_id   UUID REFERENCES lessons(id),
  user_id     UUID NOT NULL REFERENCES users(id),
  course_id   UUID REFERENCES courses(id),
  answers     JSONB NOT NULL,        -- { "questionId": "optionId" }
  score       NUMERIC(8,2) NOT NULL,
  total       NUMERIC(8,2) NOT NULL,
  passed      BOOLEAN NOT NULL,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX uniq_quiz_pass ON quiz_attempts(user_id, lesson_id) WHERE passed = TRUE;
```

Pass rule (frontend): all questions correct **or** score >= 50%. Recommend: **score >= 50%** to pass, configurable later.

#### `coupons`

```sql
CREATE TABLE coupons (
  id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  code              VARCHAR(40) NOT NULL UNIQUE,   -- uppercase
  discount_type     discount_type NOT NULL,
  discount_value    NUMERIC(12,2) NOT NULL,
  expiry_date       DATE NOT NULL,
  max_redemptions   INT NOT NULL DEFAULT 200,
  redemption_count  INT NOT NULL DEFAULT 0,
  is_active         BOOLEAN NOT NULL DEFAULT TRUE,
  created_by        UUID REFERENCES users(id),
  created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

JSON:

```json
{
  "id": "uuid",
  "code": "NEXURA20",
  "discountType": "percentage",
  "discountValue": 20,
  "expiryDate": "2026-12-31",
  "maxRedemptions": 500,
  "redemptionCount": 142,
  "isActive": true
}
```

#### `coupon_redemptions`

```sql
CREATE TABLE coupon_redemptions (
  id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  coupon_id    UUID NOT NULL REFERENCES coupons(id),
  user_id      UUID NOT NULL REFERENCES users(id),
  course_id    UUID NOT NULL REFERENCES courses(id),
  payment_id   UUID REFERENCES payments(id),
  created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (coupon_id, user_id, course_id)
);
```

#### `wallets` (dummy money — admin + instructors)

There is **no real bank transfer**. Each admin has one platform wallet. Each instructor has one earnings wallet. Dummy checkout credits these balances.

```sql
CREATE TABLE wallets (
  id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  owner_type   wallet_owner_type NOT NULL,
  user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  balance      NUMERIC(14,2) NOT NULL DEFAULT 0,
  currency     VARCHAR(8) NOT NULL DEFAULT 'BDT',
  created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (owner_type, user_id)
);
```

On first admin seed: create `wallets` row `owner_type='admin'` with `balance=0`.  
On instructor register: create `owner_type='instructor'` wallet with `balance=0`.

#### `wallet_ledger`

```sql
CREATE TABLE wallet_ledger (
  id             UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  wallet_id      UUID NOT NULL REFERENCES wallets(id) ON DELETE CASCADE,
  payment_id     UUID REFERENCES payments(id),
  entry_type     wallet_entry_type NOT NULL,
  amount         NUMERIC(12,2) NOT NULL CHECK (amount > 0),
  balance_after  NUMERIC(14,2) NOT NULL,
  note           VARCHAR(255),
  created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_wallet_ledger_wallet ON wallet_ledger(wallet_id, created_at DESC);
```

#### `payments`

Dummy checkout creates a row with `gateway='dummy'`, `is_dummy=TRUE`, `status='paid'` in the **same request**. No pending→webhook cycle.

```sql
CREATE TABLE payments (
  id               UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  public_txn_id    VARCHAR(64) NOT NULL UNIQUE,  -- TXN-DUMMY-12345678
  user_id          UUID NOT NULL REFERENCES users(id),
  course_id        UUID NOT NULL REFERENCES courses(id),
  coupon_id        UUID REFERENCES coupons(id),
  original_price   NUMERIC(12,2) NOT NULL,
  discount_amount  NUMERIC(12,2) NOT NULL DEFAULT 0,
  amount_paid      NUMERIC(12,2) NOT NULL,
  gateway          payment_gateway NOT NULL DEFAULT 'dummy',
  gateway_label    VARCHAR(40),              -- optional UI label: Stripe / SSLCommerz / Shurjopay
  gateway_ref      VARCHAR(160),
  is_dummy         BOOLEAN NOT NULL DEFAULT TRUE,
  status           payment_status NOT NULL DEFAULT 'paid',
  created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  paid_at          TIMESTAMPTZ DEFAULT NOW()
);
```

#### `enrollments`

```sql
CREATE TABLE enrollments (
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
CREATE INDEX idx_enrollments_course ON enrollments(course_id);
CREATE INDEX idx_enrollments_user ON enrollments(user_id);
```

After successful **dummy** payment (all in one DB transaction):

1. Insert `payments` (`is_dummy=true`, `status=paid`).
2. Insert `enrollments`.
3. Insert `transactions` ledger row (admin commission + instructor share).
4. Credit **admin wallet** with `admin_commission_amount`.
5. If `creator_type=instructor`, credit **instructor wallet** with `instructor_earnings`.
6. Increment coupon `redemption_count` if used.
7. Auto-join student into course group chat + create instructor DM if missing.
8. Notify instructor (`type=enrollment`) and admin (`type=payment`).

JSON `StudentEnrollment`:

```json
{
  "id": "uuid",
  "studentName": "Sadia Rahman",
  "studentEmail": "sadia@example.com",
  "studentAvatar": "https://...",
  "courseTitle": "Reactive Accelerator",
  "enrolledDate": "2024-02-14",
  "progress": 45,
  "paymentStatus": "paid"
}
```

JSON `EnrolledCourse` (extends Course):

```json
{
  "...course fields...": true,
  "enrolledDate": "2024-01-10",
  "completedModules": 2,
  "totalModules": 8,
  "completedQuizzes": 3,
  "totalQuizzes": 6,
  "quizScore": 80,
  "otherScore": 10,
  "totalScore": 90,
  "progress": 62
}
```

#### `transactions` (admin ledger)

```sql
CREATE TABLE transactions (
  id                       UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  payment_id               UUID NOT NULL REFERENCES payments(id),
  course_id                UUID NOT NULL REFERENCES courses(id),
  course_title             VARCHAR(255) NOT NULL,
  creator_type             creator_type NOT NULL,
  instructor_id            UUID REFERENCES users(id),
  instructor_name          VARCHAR(160) NOT NULL,
  student_id               UUID NOT NULL REFERENCES users(id),
  student_name             VARCHAR(160) NOT NULL,
  student_email            VARCHAR(160) NOT NULL,
  price                    NUMERIC(12,2) NOT NULL,
  admin_commission_rate    NUMERIC(5,4) NOT NULL,  -- 0.05 or 1.0000
  admin_commission_amount  NUMERIC(12,2) NOT NULL,
  instructor_earnings      NUMERIC(12,2) NOT NULL,
  payment_method           VARCHAR(40) NOT NULL DEFAULT 'Dummy',
  status                   txn_status NOT NULL DEFAULT 'completed',
  created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_txn_created ON transactions(created_at DESC);
```

JSON `PlatformTransaction`:

```json
{
  "id": "TXN-DUMMY-18402911",
  "courseId": "uuid",
  "courseTitle": "Reactive Accelerator",
  "creatorType": "instructor",
  "instructorName": "Tapas Adhikary",
  "studentName": "Rahim Ahmed",
  "studentEmail": "rahim@example.com",
  "price": 4999,
  "adminCommissionRate": 0.05,
  "adminCommissionAmount": 249.95,
  "instructorEarnings": 4749.05,
  "date": "2026-09-15",
  "status": "completed",
  "paymentMethod": "Dummy"
}
```

Use `public_txn_id` as `id` in this JSON so admin table search works. Always `"paymentMethod": "Dummy"` (frontend invoice may still print the selected UI gateway label).

#### `lesson_progress`

```sql
CREATE TABLE lesson_progress (
  id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_id      UUID NOT NULL REFERENCES users(id),
  lesson_id    UUID NOT NULL REFERENCES lessons(id) ON DELETE CASCADE,
  course_id    UUID NOT NULL REFERENCES courses(id),
  completed    BOOLEAN NOT NULL DEFAULT FALSE,
  last_position_sec INT NOT NULL DEFAULT 0,
  completed_at TIMESTAMPTZ,
  UNIQUE (user_id, lesson_id)
);
```

Recompute `enrollments.progress` after each complete:

```
progress = completed_published_lessons / total_published_lessons * 100
```

When progress hits 100, set `enrollments.completed_at`.

#### `lesson_notes`

```sql
CREATE TABLE lesson_notes (
  id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_id    UUID NOT NULL REFERENCES users(id),
  lesson_id  UUID NOT NULL REFERENCES lessons(id) ON DELETE CASCADE,
  timestamp_sec INT NOT NULL DEFAULT 0,
  text       TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

JSON:

```json
{
  "id": "uuid",
  "timestamp": 45,
  "text": "Review Virtual DOM diffing",
  "createdAt": "07:12 PM"
}
```

#### `discussions` (lesson Q&A)

```sql
CREATE TABLE discussions (
  id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  lesson_id  UUID NOT NULL REFERENCES lessons(id) ON DELETE CASCADE,
  course_id  UUID NOT NULL REFERENCES courses(id),
  user_id    UUID NOT NULL REFERENCES users(id),
  content    TEXT NOT NULL,
  upvotes    INT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

#### `discussion_replies`

```sql
CREATE TABLE discussion_replies (
  id             UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  discussion_id  UUID NOT NULL REFERENCES discussions(id) ON DELETE CASCADE,
  user_id        UUID NOT NULL REFERENCES users(id),
  content        TEXT NOT NULL,
  created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

#### `discussion_votes`

```sql
CREATE TABLE discussion_votes (
  discussion_id UUID NOT NULL REFERENCES discussions(id) ON DELETE CASCADE,
  user_id       UUID NOT NULL REFERENCES users(id),
  PRIMARY KEY (discussion_id, user_id)
);
```

JSON thread:

```json
{
  "id": "uuid",
  "userName": "Rakibul Hasan",
  "userAvatar": "https://...",
  "userRole": "student",
  "content": "Does HLS auto-adjust bitrate?",
  "createdAt": "2 hours ago",
  "upvotes": 4,
  "replies": [
    {
      "id": "uuid",
      "userName": "Tapas Adhikary",
      "userAvatar": "https://...",
      "userRole": "instructor",
      "content": "Yes, hls.js handles ABR.",
      "createdAt": "1 hour ago"
    }
  ]
}
```

#### `reviews`

```sql
CREATE TABLE reviews (
  id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  course_id  UUID NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
  user_id    UUID NOT NULL REFERENCES users(id),
  rating     SMALLINT NOT NULL CHECK (rating BETWEEN 1 AND 5),
  comment    TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (course_id, user_id)
);
```

JSON `CourseReview`:

```json
{
  "id": "uuid",
  "studentName": "Sadia Rahman",
  "studentAvatar": "https://...",
  "rating": 5,
  "comment": "Outstanding walkthrough...",
  "date": "2024-02-14"
}
```

#### `live_classes`

```sql
CREATE TABLE live_classes (
  id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  instructor_id UUID NOT NULL REFERENCES users(id),
  course_id    UUID REFERENCES courses(id) ON DELETE SET NULL,
  title        VARCHAR(255) NOT NULL,
  description  TEXT,
  date         VARCHAR(40) NOT NULL,    -- frontend currently sends display string "15 Nov 2024"
  time         VARCHAR(40) NOT NULL,    -- "08:00 PM"
  starts_at    TIMESTAMPTZ,             -- store real timestamp too
  duration     VARCHAR(20),
  meeting_link TEXT,
  is_completed BOOLEAN NOT NULL DEFAULT FALSE,
  status       live_status NOT NULL DEFAULT 'scheduled',
  created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

JSON `LiveClass`:

```json
{
  "id": "uuid",
  "title": "Live Career Roadmap Q&A",
  "description": "...",
  "date": "15 Nov 2024",
  "time": "08:00 PM",
  "duration": "90 min",
  "meetingLink": "https://meet.google.com/...",
  "isCompleted": false
}
```

Notify enrolled students ~15 minutes before `starts_at` (`notification_type=live`).

#### `certificates`

```sql
CREATE TABLE certificates (
  id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  public_id       VARCHAR(32) NOT NULL UNIQUE,  -- NEX-123456
  user_id         UUID NOT NULL REFERENCES users(id),
  course_id       UUID NOT NULL REFERENCES courses(id),
  student_name    VARCHAR(160) NOT NULL,
  course_title    VARCHAR(255) NOT NULL,
  issued_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (user_id, course_id)
);
```

Issue only when enrollment progress = 100 (all published lessons completed + mandatory quizzes passed).

Public verify: `GET /certificates/verify/:publicId` (no auth).

#### `notifications`

```sql
CREATE TABLE notifications (
  id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  type       notification_type NOT NULL DEFAULT 'system',
  title      VARCHAR(255) NOT NULL,
  message    TEXT NOT NULL,
  is_read    BOOLEAN NOT NULL DEFAULT FALSE,
  link       VARCHAR(255),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_notif_user ON notifications(user_id, is_read, created_at DESC);
```

JSON:

```json
{
  "id": "uuid",
  "type": "live",
  "title": "Live Class Starting Soon",
  "message": "Mastering RTK Query live stream starts in 15 minutes!",
  "timestamp": "5m ago",
  "isRead": false
}
```

#### Chat tables

```sql
CREATE TABLE conversations (
  id             UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  type           conversation_type NOT NULL,
  name           VARCHAR(255) NOT NULL,
  avatar         TEXT,
  course_id      UUID REFERENCES courses(id) ON DELETE CASCADE,
  instructor_id  UUID REFERENCES users(id),
  created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE conversation_members (
  conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
  user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  unread_count    INT NOT NULL DEFAULT 0,
  last_read_at    TIMESTAMPTZ,
  joined_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (conversation_id, user_id)
);

CREATE TABLE messages (
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
CREATE INDEX idx_messages_conv ON messages(conversation_id, created_at);

CREATE TABLE message_reactions (
  message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
  user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  emoji      VARCHAR(16) NOT NULL,
  PRIMARY KEY (message_id, user_id, emoji)
);
```

Direct conversation uniqueness: for two users, store a sorted pair key.

```sql
CREATE TABLE direct_pairs (
  conversation_id UUID PRIMARY KEY REFERENCES conversations(id) ON DELETE CASCADE,
  user_a          UUID NOT NULL REFERENCES users(id),
  user_b          UUID NOT NULL REFERENCES users(id),
  UNIQUE (user_a, user_b),
  CHECK (user_a < user_b)
);
```

JSON `ChatConversation`:

```json
{
  "id": "uuid",
  "type": "direct",
  "name": "Tapas Adhikary (Instructor)",
  "avatar": "https://...",
  "courseId": "uuid",
  "courseTitle": "Reactive Accelerator",
  "instructorId": "uuid",
  "members": [
    {
      "id": "uuid",
      "name": "Tapas Adhikary",
      "avatar": "https://...",
      "role": "instructor",
      "isOnline": true
    }
  ],
  "lastMessage": { },
  "unreadCount": 2,
  "updatedAt": "07:15 PM"
}
```

JSON `ChatMessage`:

```json
{
  "id": "uuid",
  "senderId": "uuid",
  "senderName": "Rahim Ahmed",
  "senderAvatar": "https://...",
  "senderRole": "student",
  "content": "Hello",
  "timestamp": "07:15 PM",
  "isRead": true,
  "imageUrl": null,
  "replyTo": {
    "id": "uuid",
    "senderName": "Tapas",
    "content": "Welcome"
  },
  "attachment": {
    "type": "image",
    "url": "https://...",
    "name": "shot.png"
  },
  "reactions": [
    { "emoji": "👍", "count": 2, "users": ["uuid1", "uuid2"] }
  ]
}
```

---

## 5. Auth JSON (login / register / me)

### Request `POST /auth/login`

```json
{ "email": "m@example.com", "password": "secret1", "rememberMe": true }
```

Password min 6 chars.

### Request `POST /auth/register`

```json
{
  "firstName": "John",
  "lastName": "Doe",
  "email": "m@example.com",
  "password": "secret1",
  "confirmPassword": "secret1",
  "role": "student"
}
```

`role` ∈ `student` | `instructor` only.

### Response

```json
{
  "success": true,
  "data": {
    "user": { "id": "...", "firstName": "...", "lastName": "...", "email": "...", "role": "student" },
    "token": "<jwt>",
    "refreshToken": "<jwt>"
  }
}
```

JWT payload:

```json
{ "sub": "user-uuid", "role": "student", "email": "..." }
```

---

## 6. Auth endpoints

| Method | Path | Auth | Body / Query | Response |
|---|---|---|---|---|
| POST | `/auth/register` | Public | RegisterCredentials | `{ user, token }` |
| POST | `/auth/login` | Public | `{ email, password }` | `{ user, token }` |
| POST | `/auth/logout` | User | — | `{ message }` |
| GET | `/auth/me` | User | — | `User` |
| PUT | `/auth/profile` | User | Partial User: firstName, lastName, email, bio, occupation, phone, website, avatar | `User` |
| POST | `/auth/change-password` | User | `{ currentPassword, newPassword }` | `{ message }` |
| POST | `/auth/forgot-password` | Public | `{ email }` | `{ message }` |
| POST | `/auth/reset-password` | Public | `{ token, newPassword }` | `{ message }` |
| POST | `/auth/refresh-token` | Public | `{ refreshToken }` | `{ token, refreshToken }` |
| POST | `/auth/verify-email` | Public | `{ token }` | `{ message }` |
| POST | `/upload/avatar` | User | multipart `file` | `{ url }` |

Profile fields used on `/account`: firstName, lastName, email, occupation, bio, phone, website, avatar.

---

## 7. Chat system (implement first after auth)

Admin users must **not** join chat. If `role=admin` connects, disconnect immediately.

### 7.1 REST

| Method | Path | Auth | Purpose |
|---|---|---|---|
| GET | `/chat/conversations` | student, instructor | Inbox list with lastMessage + unreadCount |
| GET | `/chat/conversations/:id` | member | Conversation + members |
| GET | `/chat/conversations/:id/messages` | member | Paginated messages `?page&limit=50` (oldest→newest or newest first + reverse on client) |
| POST | `/chat/conversations/direct` | student, instructor | `{ userId }` — get-or-create DM |
| POST | `/chat/groups` | instructor | `{ courseId, groupName }` — create course group; instructor is first member |
| POST | `/chat/groups/:id/join` | enrolled student | Join course group (also called internally on enroll) |
| POST | `/chat/conversations/:id/read` | member | Reset unreadCount to 0 |
| POST | `/chat/messages` | member | `{ conversationId, content, imageUrl?, replyToId? }` persist + broadcast |
| DELETE | `/chat/messages/:id` | sender (within 15 min recommended) | Unsend; set `is_deleted` or hard delete |
| POST | `/chat/messages/:id/reactions` | member | `{ emoji }` toggle |
| POST | `/chat/upload-image` | student, instructor | multipart image, max **5MB** |

Create group body:

```json
{ "courseId": "uuid", "groupName": "React Accelerator - Batch 1" }
```

On create, insert system/instructor welcome message:

```
Welcome to the official group chat for {courseTitle}!
```

On enroll auto-join, if group missing create `{courseTitle} Community Group` and message:

```
Welcome {studentName} to the {courseTitle} community group!
```

Also create DM with instructor + message:

```
Welcome to {courseTitle}! Feel free to send me any direct questions.
```

One group per course is enough (`UNIQUE(course_id)` where type=group).

### 7.2 Socket.IO

Connect:

```
io(SOCKET_URL, {
  transports: ["websocket", "polling"],
  auth: { userId, userRole }
})
```

Prefer also accepting `Authorization: Bearer` in handshake.

After connect, put user in personal room `user:{userId}` and mark online.

#### Client → Server

| Event | Payload |
|---|---|
| `join_room` | `{ conversationId }` |
| `leave_room` | `{ conversationId }` |
| `send_message` | `{ conversationId, message: ChatMessage }` |
| `typing_status` | `{ conversationId, userId, userName, isTyping }` |
| `send_reaction` | `{ conversationId, messageId, emoji, userId }` |

Persist `send_message` on server (do not trust client `message.id`). Generate UUID, save, then broadcast the saved message.

#### Server → Client

| Event | Payload |
|---|---|
| `receive_message` | `{ conversationId, message }` |
| `user_typing` | `{ conversationId, userId, userName, isTyping }` |
| `user_presence` | `{ userId, isOnline }` |
| `receive_reaction` | `{ conversationId, messageId, emoji, userId }` |

Rules:

- Only members of the conversation receive `receive_message`.
- Do not echo typing to the sender (optional; frontend handles duplicates).
- On disconnect, emit `user_presence { isOnline: false }`.
- Unsend: either add event `message_deleted` `{ conversationId, messageId }` (recommended) or reuse receive with empty content.

Image messages: frontend currently sends base64 `imageUrl` from FileReader. Backend should:

1. Accept data-URL **or** uploaded CDN URL.
2. If data-URL, store file and replace with HTTPS URL before broadcast (do not persist huge base64 in Postgres).

---

## 8. Page-by-page frontend map

Legend: **P** = public, **S** = student (any logged-in), **I** = instructor, **A** = admin.

---

### 8.1 Auth pages

#### `GET /login` — LoginPage

- **Role:** P  
- **Data:** none  
- **Endpoints:** `POST /auth/login`

#### `GET /register` — RegisterPage

- **Role:** P  
- Query `?role=instructor` preselects instructor.  
- **Endpoints:** `POST /auth/register`

---

### 8.2 Public / student storefront

#### `GET /` — HomePage

- **Role:** P  
- **Needs:**
  - Categories (id, title, thumbnail, value)
  - Featured published courses (isFeatured=true or latest)
- **Endpoints:**
  - `GET /categories`
  - `GET /courses?isFeatured=true&isPublished=true&limit=8`

#### `GET /courses` — CoursesPage

- **Role:** P  
- Filters (frontend currently client-side; support server-side too):
  - `search`, `category` (multi), `price=free|paid`, `sort=price-asc|price-desc|popular`
- **Endpoints:**
  - `GET /courses`
  - `GET /categories`

List item fields: `id, slug, title, category, thumbnail, image, price, instructor.name, isPublished, isFeatured`.

#### `GET /courses/:courseId` — CourseDetailPage

- **Role:** P (`:courseId` may be UUID **or slug**)  
- **Needs:** full course + modules + lessons (titles, isFree, duration) + instructor + reviews + learningPoints  
- Hide paid `videoUrl` unless enrolled / isFree.  
- **Endpoints:**
  - `GET /courses/:id`
  - `GET /enrollments/:courseId/status` (if logged in)
  - `GET /courses/:id/reviews`
  - Dummy checkout: `POST /payments/dummy` (Section 9 — no real gateway)

Tabs: Overview, Curriculum, Instructor, Reviews.

#### `GET /inst-profile` — InstructorProfilePage

Frontend currently has no `:id`. Implement:

- `GET /instructors/:id`  
- `GET /courses?instructorId=:id&isPublished=true`

JSON instructor public profile:

```json
{
  "id": "uuid",
  "name": "Tapas Adhikary",
  "designation": "Senior Software Engineer & Educator",
  "avatar": "...",
  "bio": "...",
  "coursesCount": 10,
  "studentsCount": 2000,
  "reviewsCount": 1500,
  "rating": 4.9,
  "courses": []
}
```

Recommend frontend later uses `/inst-profile/:instructorId`. Until then, return a featured instructor or `?id=`.

#### `GET /enroll-success` — EnrollSuccessPage

- **Role:** P (should be S after pay)  
- Static UI. Optional: `GET /enrollments/latest` to deep-link player.

#### `GET /account` — AccountProfilePage

- **Role:** S (any logged-in)  
- **Endpoints:** `GET /auth/me`, `PUT /auth/profile`, `POST /auth/change-password`

#### `GET /account/enrolled-courses` — EnrolledCoursesPage

- **Role:** S  
- **Endpoints:** `GET /user/enrolled-courses`

---

### 8.3 Chat pages

#### `GET /messages` and `GET /dashboard/messages` — ChatPage

- **Role:** student + instructor. Admin redirected to `/admin`.  
- **Needs:** conversations, messages, presence, typing  
- **Endpoints:** Chat REST (Section 7) + Socket.IO  
- Instructor can open **Create Group** modal → `POST /chat/groups` using own courses (`GET /dashboard/courses` or `/courses?mine=true`)

---

### 8.4 Course player

#### `GET /player/:courseSlug/:lessonId` — CoursePlayerPage

- **Role:** logged-in + **enrolled** (or `isFree` lesson preview)  
- **Needs:**
  - Course with modules/lessons + progress
  - Current lesson videoUrl (HLS `.m3u8` preferred)
  - Resources, quiz questions (without `isCorrect`)
  - Notes, discussion, review CTA, certificate when 100%
- **Endpoints:**
  - `GET /courses/:slug` (include progress if enrolled)
  - `GET /lessons/:lessonId`
  - `PATCH /lessons/:lessonId/complete`
  - `POST /quizzes/:quizId/submit`
  - `GET|POST|DELETE /lessons/:lessonId/notes`
  - `GET|POST /lessons/:lessonId/discussions` + replies + upvote
  - `POST /courses/:id/reviews`
  - `GET /certificates/me/:courseId` + `POST /certificates/:courseId`

Watermark: frontend overlays `user.email` / `user.id` — no extra API.

---

### 8.5 Instructor Studio (`/dashboard/*`)

Access: `instructor` **or** `admin`.

#### `GET /dashboard` — DashboardOverviewPage

- **Needs:** stats + recent enrollments + course count  
- **Endpoints:**
  - `GET /dashboard/stats`
  - `GET /dashboard/enrollments?limit=10`
  - `GET /dashboard/courses`

`DashboardStats`:

```json
{
  "totalCourses": 4,
  "totalEnrollments": 128,
  "totalRevenue": 185000,
  "totalStudents": 96
}
```

For instructor: **own** courses only. Revenue = 95% of own paid sales (or show gross — frontend label is “Lifetime gross earnings”; prefer **instructor 95% earnings** and document it). Recommend returning both:

```json
{
  "totalCourses": 4,
  "totalEnrollments": 128,
  "totalRevenue": 185000,
  "instructorEarnings": 175750,
  "totalStudents": 96
}
```

#### `GET /dashboard/courses` — DashboardCoursesPage

- Table: thumbnail, title, price, published, enrollments  
- Actions: edit, enrollments, delete  
- **Endpoints:** `GET /dashboard/courses`, `DELETE /courses/:id`

#### `GET /dashboard/courses/add` — AddCoursePage

- Body: `{ title, description }` then server fills draft defaults.  
- **Endpoint:** `POST /courses`  
- Response course with `id` — frontend navigates to `/dashboard/courses/:id` (edit).  
- Set `creatorType` from actor role (`admin` → admin, else instructor).  
- `instructor_id` = current user.  
- Generate unique `slug` from title.

#### `GET /dashboard/courses/:courseId` — EditCoursePage

- **Needs:** course + modules + categories  
- **Endpoints:**
  - `GET /courses/:id`
  - `GET /categories`
  - `PUT /courses/:id` (title, description, thumbnail/image, category, price, isPublished, modules)
  - `PATCH /courses/:id/publish` / `unpublish`
  - `DELETE /courses/:id`
  - `POST /courses/:id/modules`
  - `PUT /courses/:courseId/modules/reorder` `{ moduleIds: [] }`
  - `POST /upload/image`

Publish guard: require title, description, image, category, price, ≥1 module.

#### `GET /dashboard/courses/:courseId/modules/:moduleId` — EditModulePage

- **Needs:** module, lessons, resources, attached quiz set, available quiz sets  
- **Endpoints:**
  - `GET /modules/:moduleId`
  - `PUT /modules/:moduleId`
  - `DELETE /modules/:moduleId`
  - `POST /modules/:moduleId/lessons`
  - `PUT /lessons/:lessonId`
  - `DELETE /lessons/:lessonId`
  - `PUT /modules/:moduleId/lessons/reorder` `{ lessonIds: [] }`
  - `POST /lessons/:lessonId/resources`
  - `DELETE /lessons/:lessonId/resources/:resourceId`
  - `GET /dashboard/quiz-sets` (to attach)
  - `POST /upload/video` | `/upload/pdf` | `/upload/file`

Lesson fields editable: title, description, videoUrl, isFree, isPublished, quizSetId, resources, inline questions.

#### `GET /dashboard/courses/:courseId/enrollments` — CourseEnrollmentsPage

- **Endpoint:** `GET /courses/:courseId/enrollments?search=`
- Columns: studentName, studentEmail, enrolledDate, progress, paymentStatus

#### `GET /dashboard/courses/:courseId/reviews` — CourseReviewsPage

- **Endpoint:** `GET /courses/:id/reviews`
- Columns: studentName, rating, comment, date

#### `GET /dashboard/lives` — DashboardLivesPage

- **Endpoints:** `GET /dashboard/lives`, `DELETE /lives/:id`

#### `GET /dashboard/lives/add` — AddLivePage

- Body: `{ title, description, date, time, meetingLink, courseId? }`  
- **Endpoint:** `POST /dashboard/lives` or `POST /lives`

#### `GET /dashboard/lives/:liveId` — EditLivePage

- **Endpoints:** `GET /lives/:id`, `PUT /lives/:id`, `DELETE /lives/:id`

#### `GET /dashboard/quiz-sets` — DashboardQuizSetsPage

- **Endpoints:** `GET /dashboard/quiz-sets`, `DELETE /quizzes/:id`  
- Columns: title, questions.length, totalMarks, isPublished

#### `GET /dashboard/quiz-sets/add` — AddQuizSetPage

- Body: `{ title, description }`  
- **Endpoint:** `POST /quizzes`  
- Return `{ id }` so frontend can go to edit questions.

#### `GET /dashboard/quiz-sets/:quizSetId` — EditQuizSetPage

- **Endpoints:**
  - `GET /quizzes/:id`
  - `PUT /quizzes/:id`
  - `POST /quizzes/:id/questions`
  - `DELETE /quizzes/:quizId/questions/:qId`

Question body:

```json
{
  "title": "Which hook is used for side effects?",
  "description": "...",
  "points": 5,
  "options": [
    { "label": "useEffect", "isCorrect": true },
    { "label": "useState", "isCorrect": false },
    { "label": "useReducer", "isCorrect": false },
    { "label": "useMemo", "isCorrect": false }
  ]
}
```

Exactly one option should be `isCorrect: true`.

---

### 8.6 Admin Control Hub (`/admin/*`)

Access: **admin only**.

#### `GET /admin` — AdminOverviewPage

- **Needs:** KPIs, monthly growth, category stats, recent 5 transactions, coupons  
- **Endpoints:**
  - `GET /analytics/overview`
  - `GET /admin/wallet` (dummy platform balance — this is where student “tk” lands)
  - `GET /analytics/revenue?limit=5` (or overview includes `recentTransactions`)
  - `GET /users`
  - `GET /courses` (admin sees drafts too)
  - `GET /coupons`

`AdminStats`:

```json
{
  "totalRevenue": 1250000,
  "adminNetCommission": 186000,
  "instructorPayouts": 1064000,
  "dummyWalletBalance": 186000,
  "isDummyPayments": true,
  "totalStudents": 1240,
  "totalInstructors": 18,
  "totalCourses": 42,
  "activeEnrollments": 3100,
  "growthRate": { "revenue": 24.8, "students": 12.1, "courses": 8.0 }
}
```

`dummyWalletBalance` **must equal** `wallets.balance` for the platform admin (same as sum of dummy admin credits).

`MonthlyGrowthData[]`:

```json
{
  "month": "Jan",
  "gmv": 120000,
  "adminRevenue": 18000,
  "instructorEarnings": 102000,
  "students": 80,
  "enrollments": 140
}
```

`CategoryStat[]`:

```json
{
  "name": "Web Development",
  "count": 12,
  "revenue": 450000,
  "color": "#0ea5e9"
}
```

Assign stable colors server-side or let frontend color; include `color`.

#### `GET /admin/courses` — AdminCoursesPage

- Filters: search title/instructor, category, creatorType, published/draft  
- Actions: view, edit (`/dashboard/courses/:id`), publish, feature, delete  
- **Endpoints:**
  - `GET /courses?includeUnpublished=true` (admin)
  - `PUT /courses/:id` `{ isPublished, isFeatured }`
  - `PATCH /courses/:id/publish` | `/unpublish`
  - `DELETE /courses/:id`

#### `GET /admin/users` — AdminUsersPage

- Tabs: all / student / instructor / admin  
- Search name/email  
- Actions: suspend/activate, change role, delete, CSV is client-side  
- **Endpoints:**
  - `GET /users?role=&search=&page=&limit=`
  - `PATCH /users/:id/role` `{ role }`
  - `PATCH /users/:id/status` `{ status: "active"|"suspended" }`
  - `DELETE /users/:id`

`ManagedUser`:

```json
{
  "id": "uuid",
  "firstName": "Rahim",
  "lastName": "Ahmed",
  "email": "rahim@example.com",
  "role": "student",
  "avatar": "...",
  "status": "active",
  "joinDate": "2024-01-10",
  "enrolledCoursesCount": 3,
  "totalSpent": 9998,
  "createdCoursesCount": 0,
  "totalStudentsCount": 0,
  "totalEarnings": 0,
  "adminCommissionGenerated": 0,
  "phone": "...",
  "bio": "..."
}
```

For instructors fill `createdCoursesCount`, `totalEarnings` (95%), `totalStudentsCount`, `adminCommissionGenerated`.  
For students fill `enrolledCoursesCount`, `totalSpent`.

Prevent deleting own admin account. Require at least one remaining admin.

#### `GET /admin/revenue` — AdminRevenuePage

- **Endpoints:** `GET /analytics/revenue`  
- Filters: search txn id / student / course, creatorType  
- CSV is generated on frontend from this list.

Return:

```json
{
  "totalGmv": 1250000,
  "adminNetCommission": 186000,
  "instructorPayouts": 1064000,
  "dummyWalletBalance": 186000,
  "isDummyPayments": true,
  "instructorCourseSales": 1060000,
  "admin5PercentCut": 53000,
  "adminDirectCourseSales": 133000,
  "monthlyGrowth": [],
  "transactions": []
}
```

All of these numbers come from dummy `payments` / `transactions` / `wallets` — not from a bank.

#### Coupons (embedded on Admin Overview)

| Method | Path | Auth |
|---|---|---|
| GET | `/coupons` | admin |
| POST | `/coupons` | admin |
| DELETE | `/coupons/:id` | admin |
| PATCH | `/coupons/:id` | admin (toggle active) |
| POST | `/coupons/validate` | logged-in | `{ code, courseId }` |

Validate response:

```json
{
  "valid": true,
  "code": "NEXURA20",
  "discountType": "percentage",
  "discountValue": 20,
  "discountAmount": 999.8,
  "finalPrice": 3999.2
}
```

Invalid / expired / maxed → 422 `{ message: "Invalid promo coupon code." }`

---

## 9. Dummy payments & enrollment

**v1 is dummy only.** Backend simulates payment. No Stripe, SSLCommerz, Shurjopay, cards, redirects, or webhooks.

Frontend checkout (`PaymentCheckoutModal`) may still show three gateway buttons. Treat `gateway` as an optional invoice label. Processing is always dummy.

### 9.1 What happens on one POST

Student is logged in → `POST /payments/dummy` → backend **in a single PostgreSQL transaction**:

1. Validate user is not already enrolled (`409 Already enrolled`).
2. Load published course + price.
3. Apply coupon if `couponCode` is valid (same rules as `/coupons/validate`).
4. `amount_paid = max(0, original_price - discount_amount)`.
5. Insert `payments`:
   - `gateway = 'dummy'`
   - `is_dummy = true`
   - `status = 'paid'`
   - `public_txn_id = TXN-DUMMY-{8 digits}`
   - `gateway_label` = request `gateway` if sent (`stripe` / `sslcommerz` / `shurjopay`) else `"dummy"`
6. Insert `enrollments` (`payment_status=paid`).
7. Compute split:
   - Instructor course: admin **5%** of `amount_paid`, instructor **95%**.
   - Admin course: admin **100%**, instructor **0**.
8. Insert `transactions` (`paymentMethod: "Dummy"`, `status: completed`).
9. Credit **admin dummy wallet** `+= admin_commission_amount` + ledger row.
10. If instructor course: credit **instructor dummy wallet** `+= instructor_earnings` + ledger row.
11. Coupon redemption increment.
12. Auto-join course chat group + instructor DM.
13. Notify instructor + admin.

If `amount_paid = 0` (free course or 100% coupon): still enroll; skip wallet credits (or credit 0). Prefer `POST /enrollments` for `price=0` courses with no coupon.

**Never call a payment provider.** Success is immediate. Frontend uses the JSON to show “Payment Successful” and generate the PDF invoice.

### 9.2 Endpoints

| Method | Path | Auth | Purpose |
|---|---|---|---|
| POST | `/coupons/validate` | User | `{ code, courseId }` preview discount |
| POST | `/payments/dummy` | User (student/instructor/admin) | **Main dummy checkout — implement this** |
| GET | `/payments/:id` | Payer or Admin | Receipt for PDF invoice |
| GET | `/admin/wallet` | Admin | Dummy platform wallet balance + recent credits |
| GET | `/dashboard/wallet` | Instructor | Dummy instructor earnings wallet |
| POST | `/enrollments` | User | `{ courseId }` — free course (`price=0`) only |
| GET | `/enrollments/:courseId/status` | User | `{ isEnrolled: true }` |
| GET | `/user/enrolled-courses` | User | EnrolledCourse[] |

Do **not** implement:

- `POST /payments/init` with real checkout sessions
- `POST /payments/webhook/stripe`
- `POST /payments/webhook/sslcommerz`
- `POST /payments/webhook/shurjopay`

If frontend still calls `/payments/init`, alias it to the same dummy handler so UI does not break:

`POST /payments/init` → same behavior as `POST /payments/dummy` (instant paid, no `checkoutUrl` / `clientSecret`).

### 9.3 `POST /payments/dummy`

**Request**

```json
{
  "courseId": "uuid",
  "couponCode": "NEXURA20",
  "gateway": "stripe"
}
```

| Field | Required | Notes |
|---|---|---|
| `courseId` | yes | UUID or slug |
| `couponCode` | no | Uppercase match |
| `gateway` | no | Label only. Stored as `gateway_label`. Processing is always dummy |

**Success `201`**

```json
{
  "success": true,
  "statusCode": 201,
  "message": "Dummy payment completed. Course enrolled.",
  "data": {
    "paymentId": "uuid",
    "transactionId": "TXN-DUMMY-18402911",
    "isDummy": true,
    "courseId": "uuid",
    "courseTitle": "Reactive Accelerator",
    "originalPrice": 4999,
    "discountAmount": 999.8,
    "amountPaid": 3999.2,
    "gateway": "dummy",
    "gatewayLabel": "stripe",
    "paymentMethod": "Dummy",
    "paidAt": "2026-09-15T13:00:00.000Z",
    "enrollment": {
      "id": "uuid",
      "isEnrolled": true,
      "paymentStatus": "paid"
    },
    "split": {
      "creatorType": "instructor",
      "adminCommissionRate": 0.05,
      "adminCommissionAmount": 199.96,
      "instructorEarnings": 3799.24
    },
    "adminWallet": {
      "credited": 199.96,
      "balanceAfter": 186199.96
    }
  }
}
```

Frontend invoice PDF can use `transactionId`, `amountPaid`, `courseTitle`, `paidAt`.

**Errors**

| Status | `message` |
|---|---|
| 400 | Course not published / invalid body |
| 401 | Not logged in |
| 409 | Already enrolled |
| 422 | Invalid / expired / maxed coupon |

### 9.4 Dummy wallet rules

| Event | Admin wallet | Instructor wallet |
|---|---|---|
| Student buys instructor course | +5% of `amount_paid` | +95% of `amount_paid` |
| Student buys admin course | +100% of `amount_paid` | no change |
| Free enroll (`amount_paid=0`) | no change | no change |
| Dummy refund (optional later) | −admin cut | −instructor share |

Money **never leaves the database**. `wallets.balance` is the admin “tk” the UI should show.

`GET /admin/wallet`

```json
{
  "ownerType": "admin",
  "balance": 186199.96,
  "currency": "BDT",
  "isDummy": true,
  "lifetimeCredits": 186199.96,
  "recent": [
    {
      "id": "uuid",
      "entryType": "credit",
      "amount": 199.96,
      "balanceAfter": 186199.96,
      "note": "5% dummy commission · TXN-DUMMY-18402911",
      "createdAt": "2026-09-15T13:00:00.000Z"
    }
  ]
}
```

`GET /dashboard/wallet` (instructor): same shape with their 95% dummy balance.

Admin Overview / Revenue KPIs (`adminNetCommission`, GMV) **must match** the sum of dummy `transactions` and admin wallet credits.

### 9.5 Worked example

Course price ৳4999, instructor-owned, coupon `NEXURA20` (20%):

| Item | Amount |
|---|---|
| Original | 4999.00 |
| Discount | 999.80 |
| Student dummy pay | **3999.20** |
| Admin dummy wallet (+) | **199.96** (5%) |
| Instructor dummy wallet (+) | **3799.24** (95%) |

Admin course, no coupon, price ৳2000:

| Item | Amount |
|---|---|
| Student dummy pay | 2000.00 |
| Admin dummy wallet (+) | **2000.00** |
| Instructor | 0 |

### 9.6 Idempotency

Unique `(user_id, course_id)` on `enrollments`. Second dummy pay → `409`.

Use a DB transaction with row lock on `coupons` when incrementing `redemption_count`.

---

## 10. Full REST catalog (grouped)

### Categories

| Method | Path | Auth |
|---|---|---|
| GET | `/categories` | Public |
| GET | `/categories/:id` | Public |
| POST | `/categories` | Admin |
| PUT | `/categories/:id` | Admin |
| DELETE | `/categories/:id` | Admin |

### Courses

| Method | Path | Auth |
|---|---|---|
| GET | `/courses` | Public (published); Admin sees all |
| GET | `/courses/:id` | Public (id or slug) |
| POST | `/courses` | Instructor, Admin |
| PUT | `/courses/:id` | Owner instructor or Admin |
| DELETE | `/courses/:id` | Owner instructor or Admin |
| PATCH | `/courses/:id/publish` | Owner / Admin |
| PATCH | `/courses/:id/unpublish` | Owner / Admin |
| PUT | `/courses/:courseId/modules/reorder` | Owner / Admin |

### Modules

| Method | Path | Auth |
|---|---|---|
| GET | `/courses/:courseId/modules` | Public titles; full if enrolled/owner |
| GET | `/modules/:moduleId` | Owner / Admin / enrolled |
| POST | `/courses/:courseId/modules` | Owner / Admin |
| PUT | `/modules/:moduleId` | Owner / Admin |
| DELETE | `/modules/:moduleId` | Owner / Admin |
| PUT | `/modules/:moduleId/lessons/reorder` | Owner / Admin |

### Lessons

| Method | Path | Auth |
|---|---|---|
| GET | `/lessons/:lessonId` | Enrolled / isFree / owner |
| POST | `/modules/:moduleId/lessons` | Owner / Admin |
| PUT | `/lessons/:lessonId` | Owner / Admin |
| DELETE | `/lessons/:lessonId` | Owner / Admin |
| POST | `/lessons/:lessonId/resources` | Owner / Admin |
| DELETE | `/lessons/:lessonId/resources/:resourceId` | Owner / Admin |
| PATCH | `/lessons/:lessonId/complete` | Enrolled student | `{ completed: true }` |

### Quizzes

| Method | Path | Auth |
|---|---|---|
| GET | `/quizzes` | Instructor (own) / Admin |
| GET | `/quizzes/:quizId` | Owner / Admin; student gets no isCorrect |
| POST | `/quizzes` | Instructor / Admin |
| PUT | `/quizzes/:quizId` | Owner / Admin |
| DELETE | `/quizzes/:quizId` | Owner / Admin |
| POST | `/quizzes/:quizId/questions` | Owner / Admin |
| DELETE | `/quizzes/:quizId/questions/:qId` | Owner / Admin |
| POST | `/quizzes/:quizId/submit` | Enrolled | `{ lessonId, answers: { questionId: optionId } }` |

Submit response:

```json
{ "score": 80, "total": 100, "passed": true, "correctCount": 4, "totalQuestions": 5 }
```

### Users (admin)

| Method | Path | Auth |
|---|---|---|
| GET | `/users` | Admin |
| GET | `/users/:id` | Admin (or self) |
| PATCH | `/users/:id/role` | Admin | `{ role }` |
| PATCH | `/users/:id/status` | Admin | `{ status }` |
| DELETE | `/users/:id` | Admin |

### Dashboard (instructor)

| Method | Path | Auth |
|---|---|---|
| GET | `/dashboard/stats` | Instructor / Admin |
| GET | `/dashboard/courses` | Instructor / Admin (own unless admin viewing studio) |
| GET | `/dashboard/lives` | Instructor / Admin |
| POST | `/lives` | Instructor / Admin |
| GET | `/lives/:id` | Owner / Admin |
| PUT | `/lives/:id` | Owner / Admin |
| DELETE | `/lives/:id` | Owner / Admin |
| GET | `/dashboard/quiz-sets` | Instructor / Admin |
| GET | `/dashboard/enrollments` | Instructor / Admin |
| GET | `/courses/:courseId/enrollments` | Owner / Admin |

### Dummy payments & wallets

| Method | Path | Auth |
|---|---|---|
| POST | `/payments/dummy` | User | Instant dummy pay → enroll + credit admin wallet |
| POST | `/payments/init` | User | Alias of `/payments/dummy` (no real gateway) |
| GET | `/payments/:id` | Payer / Admin | Dummy receipt |
| GET | `/admin/wallet` | Admin | Dummy admin balance (student tk) |
| GET | `/dashboard/wallet` | Instructor | Dummy 95% earnings |

### Analytics (admin)

| Method | Path | Auth |
|---|---|---|
| GET | `/analytics/overview` | Admin | Includes dummy GMV + admin wallet |
| GET | `/analytics/revenue` | Admin | Dummy ledger `?search=&creatorType=&page=` |
| GET | `/analytics/instructor` | Instructor / Admin | own dummy earnings |
| GET | `/analytics/student-progress` | Instructor / Admin |

### Media

| Method | Path | Auth |
|---|---|---|
| POST | `/upload/image` | User |
| POST | `/upload/pdf` | Instructor / Admin |
| POST | `/upload/video` | Instructor / Admin |
| POST | `/upload/file` | Instructor / Admin |
| POST | `/upload/delete` | Owner | `{ fileUrl }` |

Video: return HLS master playlist URL if transcoded (`*.m3u8`). Until transcoder exists, return mp4 URL — `hls.js` will fail over; frontend VideoPlayer also accepts YouTube embeds.

### Notifications

| Method | Path | Auth |
|---|---|---|
| GET | `/notifications` | User |
| PATCH | `/notifications/:id/read` | User |
| POST | `/notifications/read-all` | User |
| GET | `/notifications/unread-count` | User |

Also push via socket event `notification:new` (optional; UI currently polls local state).

### Notes

| Method | Path | Auth |
|---|---|---|
| GET | `/lessons/:lessonId/notes` | Enrolled (own notes only) |
| POST | `/lessons/:lessonId/notes` | `{ timestamp, text }` |
| DELETE | `/lessons/:lessonId/notes/:noteId` | Owner of note |

### Lesson discussion

| Method | Path | Auth |
|---|---|---|
| GET | `/lessons/:lessonId/discussions` | Enrolled / owner |
| POST | `/lessons/:lessonId/discussions` | `{ content }` |
| POST | `/discussions/:id/replies` | `{ content }` |
| POST | `/discussions/:id/upvote` | toggle |

### Reviews

| Method | Path | Auth |
|---|---|---|
| GET | `/courses/:id/reviews` | Public |
| POST | `/courses/:id/reviews` | Enrolled | `{ rating: 1-5, comment }` |
| PUT | `/reviews/:id` | Author |
| DELETE | `/reviews/:id` | Author / Admin |

### Certificates

| Method | Path | Auth |
|---|---|---|
| POST | `/certificates/:courseId` | Enrolled + 100% complete |
| GET | `/certificates/me/:courseId` | User |
| GET | `/certificates/verify/:publicId` | Public |

```json
{
  "publicId": "NEX-123456",
  "studentName": "Rahim Ahmed",
  "courseTitle": "Reactive Accelerator",
  "completionDate": "September 15, 2026",
  "verifyUrl": "https://nexurahub.com/verify/NEX-123456"
}
```

Frontend currently generates PDF/QR client-side. Backend still must persist `public_id` so QR verify works.

### Instructors public

| Method | Path | Auth |
|---|---|---|
| GET | `/instructors/:id` | Public |
| GET | `/instructors/:id/courses` | Public |

---

## 11. Suggested implementation order

1. Postgres + migrations + seed admin, categories, demo instructor.  
2. Auth (register/login/me/profile/password/JWT).  
3. Categories + Courses CRUD + publish.  
4. Modules, lessons, resources, uploads.  
5. **Chat REST + Socket.IO** (conversations, rooms, typing, reactions, presence).  
6. Coupons + **dummy** `POST /payments/dummy` + enrollments + transactions + **admin/instructor wallets**.  
7. Auto-join chat on enroll.  
8. Quiz sets + submit + lesson complete + progress.  
9. Live classes + notifications.  
10. Reviews, notes, discussions.  
11. Instructor dashboard stats.  
12. Admin users / overview / revenue.  
13. Certificates + verify.  
14. HLS video pipeline (can be phase 2).

---

## 12. Seed data (minimum for QA)

- 1 admin: `admin@nexurahub.com` / strong password  
- 1 instructor: `tapas@nexurahub.com`  
- 2 students  
- 8 categories with thumbnails  
- 1 published instructor course with 2 modules × 3 lessons (1 free)  
- 1 admin-owned course  
- Coupons `NEXURA20` (20%) and `FLAT500` (৳500)  
- Admin dummy wallet row (`balance=0`) + instructor dummy wallet row  
- 1 course group conversation  
- After QA: run 1–2 dummy payments so admin wallet + revenue charts are not empty  

---

## 13. Environment variables (backend)

```
PORT=5000
DATABASE_URL=postgresql://user:pass@localhost:5432/nexura_hub
JWT_ACCESS_SECRET=
JWT_REFRESH_SECRET=
JWT_ACCESS_EXPIRES=7d
JWT_REFRESH_EXPIRES=30d
CLIENT_ORIGIN=https://nexurahub.vercel.app
CLOUDINARY_URL=   # or S3
# v1 dummy payments — do NOT set real gateway keys
DUMMY_PAYMENTS=true
```

CORS must allow frontend origin and `Authorization` header. Socket.IO same origin policy / credentials as needed.

---

## 14. Security checklist

- Hash passwords with bcrypt/argon2.  
- Rate-limit `/auth/login` and `/payments/dummy`.  
- Never return `password_hash` or quiz `isCorrect` to students.  
- Signed URLs for private videos (enrolled users only).  
- Admin-only role changes; cannot demote last admin.  
- Validate coupon expiry + maxRedemptions in a transaction (row lock).  
- Dummy pay + wallet credit + enrollment must be **one DB transaction** (no partial enroll without wallet credit).  
- Do not store or log card numbers — there are none in v1.  
- Sanitize rich text in descriptions / discussion.  
- Chat: membership check on every `join_room` and `send_message`.  
- File type + size limits (images 5MB as frontend; videos larger).  

---

## 15. Frontend files this spec maps to

| Area | Path |
|---|---|
| Routes | `src/routes/index.tsx` |
| Endpoint constants | `src/api/endpoints.ts` |
| Axios base | `src/api/axiosInstance.ts` |
| Auth types | `src/types/auth.ts` |
| Course types | `src/types/course.ts` |
| Admin types | `src/types/admin.ts` |
| Chat types | `src/types/chat.ts` |
| Dashboard types | `src/types/dashboard.ts` |
| Socket client | `src/services/socket.ts`, `src/hooks/useSocket.ts` |
| Chat store events | `src/store/useChatStore.ts` |
| Payments / coupons UI | `src/components/common/PaymentCheckoutModal.tsx`, `src/components/admin/CouponManager.tsx` |

If an endpoint exists in `src/api/endpoints.ts`, implement it even if a page still uses mock data — the service layer already calls it.

---

## 16. Quick role × page matrix

| Page | student | instructor | admin |
|---|---|---|---|
| Home, Courses, Course detail, Instructor profile | view | view | view |
| Login / Register | yes | yes | login only |
| Account profile / enrolled | yes | yes | yes |
| Chat `/messages` | yes | yes | blocked |
| Player | enrolled | enrolled | enrolled / preview |
| Instructor dashboard | no | yes | yes |
| Admin hub | no | no | yes |

---

**End of specification.** Implement PostgreSQL + REST + Socket.IO as specified so the Nexura Hub frontend can switch from mock/local state to this backend with no contract changes.
