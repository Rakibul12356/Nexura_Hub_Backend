# Login → dashboard → bounced back to `/login`

This is a **backend auth bug**. Frontend login works. A later authenticated request returns **401**, the Axios interceptor clears cookies, and the user is sent back to `/login`.

Live API: `https://nexura-hub-backend.onrender.com`  
Frontend: `http://localhost:5173` → `https://nexura-hub-backend.onrender.com/api/v1`

Checked in browser: **2026-09-16**

QA account: `tapas@nexurahub.com` / `password123` (instructor → `/dashboard`)

Related: [`BACKEND_COURSES_500.md`](./BACKEND_COURSES_500.md) (`GET /courses`, `GET /auth/me` 500).

---

## What the frontend did (browser)

1. `POST /api/v1/auth/login` **succeeded** (~1.9s).
2. Cookies were set: `nexurahub_token`, `nexurahub_refresh_token`, `nexurahub_user`.
3. React navigated to **`/dashboard`** (instructor studio rendered: New Course / Schedule Live / Create Quiz).
4. Immediately after, the app called:
   - `GET /api/v1/auth/me`
   - `GET /api/v1/dashboard/courses`
   - `GET /api/v1/dashboard/stats`
   - `GET /api/v1/dashboard/enrollments`
   - `GET /api/v1/courses`
5. Session cookies were **wiped**.
6. Browser landed on **`/login`** again (full reload via `window.location.href = "/login"`).

`500` on those URLs does **not** log the user out. Only **401** (and a failed refresh) does.

So after a fresh login, **at least one** of `/auth/me`, `/dashboard/*`, or `/auth/refresh-token` is returning **401** (or refresh itself 401/500).

Do **not** map internal SQL / scan errors to 401. That kicks a valid session out.

---

## Why frontend kicks them out (by design)

`src/api/axiosInstance.ts`:

1. Any non-login request with **401** → `POST /auth/refresh-token` `{ refreshToken }`.
2. Refresh fail / missing refresh cookie → clear `nexurahub_*` cookies → toast **Session expired** → `window.location.href = "/login"`.

`GET /auth/me` is **not** treated as a public auth route. If it 401s right after login, the session dies.

`ProtectedRoute` also sends `/dashboard` → `/login` when cookies were cleared.

This is correct frontend behavior. **Do not ask frontend to ignore 401.**

---

## Likely backend causes (fix in this order)

### 1. `GET /auth/me` rejects a token login just issued

Previously this was **500**:

```json
{ "success": false, "statusCode": 500, "error": { "code": "INTERNAL_ERROR" } }
```

If the handler now returns **401** because the user row scan fails (UUID, missing column, jsonb/array), that **looks like an expired session** to the frontend.

**Required:** a token from `POST /auth/login` must get **200** on `GET /auth/me` in the same second.

```http
GET /api/v1/auth/me
Authorization: Bearer <token from login>
```

**200**

```json
{
  "success": true,
  "statusCode": 200,
  "data": {
    "id": "uuid",
    "firstName": "Tapas",
    "lastName": "Adhikary",
    "email": "tapas@nexurahub.com",
    "role": "instructor",
    "avatar": null,
    "bio": null,
    "occupation": null,
    "phone": null,
    "website": null
  }
}
```

- Invalid/missing token → **401** (ok).
- User exists but DB scan crashes → **500** with pq message (bad, but better than 401). Prefer **fix the scan** so it is **200**.
- Old token after UUID migration → **401/404**, not 500. FRONTEND_CHANGES.md already said this.

### 2. Instructor dashboard routes 401 a valid instructor JWT

Right after login the dashboard fires:

| Method | Path | Who |
|---|---|---|
| GET | `/api/v1/dashboard/courses` | instructor |
| GET | `/api/v1/dashboard/stats` | instructor |
| GET | `/api/v1/dashboard/enrollments` | instructor |

If **any** of these is 401, the interceptor logs the user out even if `/auth/me` is 200 or 500.

`GET /dashboard/courses` was already **500** (`text[]` → `jsonb`). If middleware converts that to 401, same bounce.

**Required:** valid instructor token → **200** (empty list is fine). SQL errors → **500**, never **401**.

### 3. `POST /auth/refresh-token` fails

When (1) or (2) 401s, frontend immediately posts:

```json
{ "refreshToken": "<refresh jwt from login>" }
```

If this 401/500, cookies are cleared.

**Required 200:**

```json
{
  "success": true,
  "data": {
    "token": "<new access jwt>",
    "refreshToken": "<new refresh jwt>",
    "user": { "id": "uuid", "email": "tapas@nexurahub.com", "role": "instructor" }
  }
}
```

Login `data.refreshToken` must be the same string this endpoint accepts (`refreshToken` body key, not `refresh_token` only).

### 4. Login `data` shape

Must match FRONTEND_API.md (frontend unwraps `response.data.data`):

```json
{
  "success": true,
  "data": {
    "user": { "id": "uuid", "firstName": "...", "lastName": "...", "email": "...", "role": "instructor" },
    "token": "<access jwt>",
    "refreshToken": "<refresh jwt>"
  }
}
```

- `role` must be exactly `student` | `instructor` | `admin` (lowercase).
- Access token field name must be **`token`**, not only `accessToken`.
- Do not put `user` / `token` only at the envelope root.

---

## Frontend is not the bug

- `POST /auth/login` body: `{ email, password, rememberMe: true }`
- Saves `data.token` + `data.refreshToken` + `data.user`
- Sends `Authorization: Bearer <token>` on `/auth/me` and `/dashboard/*`
- Instructor → `/dashboard`, admin → `/admin`, student → `/account/enrolled-courses`
- 401 → refresh → fail → `/login` (spec: FRONTEND_API.md / FRONTEND_CHANGES.md)

Do not change those URLs.

---

## Please verify after deploy

```bash
# 1. Login
curl -s -X POST https://nexura-hub-backend.onrender.com/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"tapas@nexurahub.com","password":"password123","rememberMe":true}'

# 2. Use data.token immediately (must be 200, not 401/500)
curl -s -o /dev/null -w "%{http_code}" \
  -H "Authorization: Bearer <token>" \
  https://nexura-hub-backend.onrender.com/api/v1/auth/me

curl -s -o /dev/null -w "%{http_code}" \
  -H "Authorization: Bearer <token>" \
  https://nexura-hub-backend.onrender.com/api/v1/dashboard/courses

curl -s -o /dev/null -w "%{http_code}" \
  -H "Authorization: Bearer <token>" \
  https://nexura-hub-backend.onrender.com/api/v1/dashboard/stats

# 3. Refresh (must be 200)
curl -s -X POST https://nexura-hub-backend.onrender.com/api/v1/auth/refresh-token \
  -H "Content-Type: application/json" \
  -d '{"refreshToken":"<refresh from login>"}'
```

All of the above must be **200** in the same minute as login. Then the frontend will stay on `/dashboard`.
