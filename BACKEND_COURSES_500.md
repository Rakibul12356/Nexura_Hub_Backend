# Backend 500s blocking the frontend (updated)

Frontend is calling the live API correctly. These **500 Internal Server Error** responses come from the backend, not from Axios/React.

Live API: `https://nexura-hub-backend.onrender.com`  
Frontend base: `https://nexura-hub-backend.onrender.com/api/v1`

Checked: **2026-09-16** (browser console + public curl)

---

## Failing endpoints (from frontend console)

| Method | Path | Where it breaks | Notes |
|---|---|---|---|
| GET | `/api/v1/courses` | `/`, `/courses`, dashboard sidebar | Public. Curl also 500. |
| GET | `/api/v1/courses?isFeatured=true&isPublished=true&limit=8` | Home featured strip | Same 500 as list. |
| GET | `/api/v1/auth/me` | Session hydrator after login | **500**, not 401/404. Token is sent. |
| GET | `/api/v1/dashboard/courses` | Instructor `/dashboard` | Logged-in instructor/admin. |
| GET | `/api/v1/admin/wallet` | Admin overview | Logged-in admin. |

### Still OK

| Endpoint | Result |
|---|---|
| `GET /health` | **200** `{ "status": "ok", "service": "Nexura Hub API Server" }` |
| `GET /api/v1/categories` | **200** UUID list |
| `GET /api/v1/courses/categories` | **200** |

---

## 1. Course lists — confirmed SQL error

Public curl (no cookies):

```bash
curl -i "https://nexura-hub-backend.onrender.com/api/v1/courses"
curl -i "https://nexura-hub-backend.onrender.com/api/v1/courses?isFeatured=true&isPublished=true&limit=8"
```

**HTTP 500** body (same for every course-list variant):

```json
{
  "success": false,
  "statusCode": 500,
  "message": "pq: cannot cast type text[] to jsonb at position 7:74 (42846)",
  "error": { "code": "INTERNAL_ERROR" }
}
```

Postgres SQLSTATE **`42846`**: cannot cast `text[]` → `jsonb`.

Also 500 with:

- `?page=1&limit=20`
- `?category=web-development`

Query params are **not** the cause. The SELECT/scan of course rows is.

### Likely cause

Course SQL casts a **`text[]` column** to **`jsonb`**.

Most likely: **`learning_points`** (API contract: `learningPoints: string[]`).

Broken pattern:

```sql
SELECT learning_points::jsonb   -- fails if column is text[]
```

or scanning `text[]` into `json.RawMessage` / GORM `datatypes.JSON`.

`GET /dashboard/courses` almost certainly hits the **same course SELECT** — fix once, both public catalog and instructor dashboard list should recover.

### Fix (pick one)

**A. Keep DB as `text[]`**

- Scan into `pq.StringArray` / `[]string`
- Do not `::jsonb` that column
- JSON can still be `"learningPoints": ["..."]`

**B. Migrate column to jsonb**

```sql
ALTER TABLE courses
  ALTER COLUMN learning_points TYPE jsonb
  USING to_jsonb(learning_points);
```

Redeploy after.

Check siblings too: `learning_outcomes`, `tags`, any `text[]` on `courses`.

### Expected 200 for `GET /api/v1/courses`

```json
{
  "success": true,
  "statusCode": 200,
  "message": "OK",
  "data": [
    {
      "id": "uuid",
      "slug": "reactive-accelerator",
      "title": "Reactive Accelerator",
      "category": "Web Development",
      "categoryId": "uuid",
      "thumbnail": "https://...",
      "image": "https://...",
      "price": 4999,
      "discountPrice": 3999,
      "isPublished": true,
      "isFeatured": true,
      "totalChapters": 2,
      "progress": 0,
      "creatorType": "instructor",
      "learningPoints": ["Build production-grade apps"],
      "instructor": {
        "id": "uuid",
        "name": "Tapas Adhikary",
        "designation": "Senior Software Engineer & Educator",
        "avatar": "https://...",
        "rating": 4.9,
        "studentsCount": 2,
        "coursesCount": 1,
        "reviewsCount": 0
      }
    }
  ],
  "meta": { "page": 1, "limit": 20, "total": 1, "totalPages": 1 }
}
```

Boolean query values are strings (`isFeatured=true`). Parse safely; do not 500 on strconv.

IDs must be **UUID strings**.

---

## 2. `GET /api/v1/auth/me` → 500

Frontend `SessionHydrator` calls this on boot when a token cookie exists.

- FRONTEND_CHANGES.md said **old token → 404**. Browser now gets **500**.
- That is a server crash in the `/auth/me` handler (user row scan, UUID vs int, missing column, etc.), not “please logout”.
- Invalid token should be **401**. Unknown user should be **401/404**. Never 500.

Please log the pq/Go error for `/auth/me` the same way courses already leak `pq: ...` in `message`.

---

## 3. `GET /api/v1/dashboard/courses` → 500

Instructor dashboard (`/dashboard`) after login as `tapas@nexurahub.com`.

Same course table/query as public list is the first thing to check (`learning_points` `text[]`/`jsonb`). If that is fixed and this still 500s, check instructor-only joins (`wallets`, `enrollments` counts).

Expected: **200** `data` = instructor’s `Course[]`.

---

## 4. `GET /api/v1/admin/wallet` → 500

Admin overview (`/admin`) after login as `admin@nexurahub.com`.

Dummy-payment wallet table is likely missing, UUID mismatch, or jsonb/array scan like courses.

Expected (FRONTEND_API.md):

```json
{
  "success": true,
  "data": {
    "ownerType": "admin",
    "balance": 0,
    "currency": "BDT",
    "isDummy": true,
    "lifetimeCredits": 0,
    "recent": []
  }
}
```

Empty wallet is fine. **500 is not.**

---

## Frontend is not the bug

- Base URL includes `/api/v1`
- Socket URL does **not** include `/api/v1`
- Unwrap is `response.data.data`
- Home query matches FRONTEND_API.md: `GET /courses?isFeatured=true&isPublished=true&limit=8`
- `GET /categories` already works
- Public `GET /courses` 500s in curl with no auth

Do not ask frontend to change these URLs.

---

## Please verify after deploy

```bash
curl -s -o /dev/null -w "%{http_code}" \
  "https://nexura-hub-backend.onrender.com/api/v1/courses?isFeatured=true&isPublished=true&limit=8"
# 200

# with Bearer token of student / instructor / admin:
# GET /auth/me            → 200
# GET /dashboard/courses  → 200 (instructor)
# GET /admin/wallet       → 200 (admin)
```

After **§1** is fixed, Home + `/courses` work. After **§2–4**, login session + instructor dashboard + admin overview stop spamming 500.
