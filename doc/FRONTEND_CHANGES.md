# Frontend-এ কী change করতে হবে

Backend already live: `https://nexura-hub-backend.onrender.com`  
CORS, categories, courses ঠিক আছে। **নতুন page বানাতে হবে না।** নিচের checklist অনুসারে frontend repo-তে শুধু mismatch গুলো ঠিক করো।

Full endpoint list: [`FRONTEND_API.md`](../FRONTEND_API.md)

---

## 0. আগে করো (১ মিনিট)

1. Browser-এ site খুলে **Logout** করো, অথবা cookies মুছো: `nexurahub_token`, `nexurahub_refresh_token`
2. Hard refresh (`Ctrl+Shift+R`)
3. নিচের নতুন email দিয়ে login

পুরনো token থাকলে `GET /auth/me` → 404 আসবে।

---

## 1. Env (must)

Vercel + local `.env`:

```
VITE_API_URL=https://nexura-hub-backend.onrender.com/api/v1
VITE_SOCKET_URL=https://nexura-hub-backend.onrender.com
```

| ভুল | ঠিক |
|---|---|
| `https://nexura-hub-backend.onrender.com` (API base হিসেবে) | শেষে `/api/v1` লাগবে |
| Socket URL-এ `/api/v1` | Socket-এ `/api/v1` দিও না |
| `localhost:8080` | Local backend হলে `http://localhost:5000/api/v1` |

Axios:

```ts
const api = axios.create({
  baseURL: import.meta.env.VITE_API_URL || "https://nexura-hub-backend.onrender.com/api/v1",
});
```

---

## 2. QA login (must)

পুরনো `NexuraHubAdmin@gmail.com` / `instructor@nexurahub.com` **মুছে দেওয়া হয়েছে**। Hardcoded হলে বদলাও।

| Role | Email | Password | Redirect |
|---|---|---|---|
| admin | `admin@nexurahub.com` | `password123` | `/admin` |
| instructor | `tapas@nexurahub.com` | `password123` | `/dashboard` |
| student | `student@nexurahub.com` | `password123` | `/account/enrolled-courses` |
| student | `sadia@nexurahub.com` | `password123` | `/account/enrolled-courses` |

Login body:

```json
{ "email": "admin@nexurahub.com", "password": "password123", "rememberMe": true }
```

Response `data` থেকে **দুটোই** save করো:

- `token` → cookie `nexurahub_token`
- `refreshToken` → cookie `nexurahub_refresh_token`

প্রতিটি protected request:

```
Authorization: Bearer <token>
```

---

## 3. ID type: number → UUID string (must)

DB-তে `categories.id` আগে integer ছিল, এখন **UUID**।

TypeScript:

```ts
// আগে
categoryId: number
id: number

// এখন
categoryId: string
id: string
```

কোনো জায়গায় `Number(category.id)` / `parseInt(id)` থাকলে **সরাতে হবে**।  
Filter/query-তে slug চলবে: `GET /courses?category=web-development`

---

## 4. API path params (must যদি পুরনো নাম ব্যবহার করো)

Backend route এখন শুধু `:id`। Frontend **page** route (`/courses/:courseId`) বদলানো লাগে না — শুধু **API URL** বদলাও।

| ভুল API path | ঠিক API path |
|---|---|
| `/courses/:courseId/modules` | `/courses/:id/modules` |
| `/modules/:moduleId` | `/modules/:id` |
| `/lessons/:lessonId` | `/lessons/:id` |
| `/lessons/:lessonId/complete` | `/lessons/:id/complete` |
| `/lessons/:lessonId/notes` | `/lessons/:id/notes` |
| `/quizzes/:quizId` | `/quizzes/:id` |
| `/enrollments/:courseId/status` | `/enrollments/:id/status` |

JSON body-তে `courseId` / `lessonId` আগের মতোই থাকবে। শুধু URL wildcard বদলেছে।

---

## 5. Response unwrap (must যদি raw `res.data` ব্যবহার করো)

সব REST reply:

```json
{ "success": true, "statusCode": 200, "message": "OK", "data": {}, "meta": null }
```

```ts
const unwrap = <T>(res: AxiosResponse): T => (res.data?.data ?? res.data) as T;
```

List-এ pagination: `res.data.meta` → `{ page, limit, total, totalPages }`

Error: `res.data.message` দেখাও। `401` হলে cookies clear করে `/login`।

---

## 6. Payment (must যদি Stripe/SSL real gateway call করো)

Real gateway **নেই**। Checkout:

```
POST /payments/dummy
{ "courseId": "<uuid-or-slug>", "couponCode": "NEXURA20", "gateway": "stripe" }
```

`gateway` শুধু invoice label। Free course (`price === 0`):

```
POST /enrollments
{ "courseId": "<uuid>" }
```

Coupon check: `POST /coupons/validate` `{ "code": "NEXURA20", "courseId": "..." }`

---

## 7. Chat / Socket (only if chat page আছে)

```ts
io(import.meta.env.VITE_SOCKET_URL, {
  transports: ["websocket", "polling"],
  extraHeaders: { Authorization: `Bearer ${token}` },
  query: { token, userId, userRole },
});
```

Admin chat **403** — UI-তে messages hide করো।

---

## 8. Frontend-এ যা করতে হবে না

- CORS header সেট করা (backend handle করে)
- নতুন admin/instructor dashboard page
- Category CRUD (student site-এ শুধু `GET /categories`)
- `PORT` env

---

## 9. Quick test

| Check | Expected |
|---|---|
| `GET https://nexura-hub-backend.onrender.com/health` | `{ "status": "ok" }` |
| Home: categories + featured courses | list আসে, id UUID |
| Login `tapas@nexurahub.com` | `/dashboard`, wallet/courses আছে |
| Login `student@nexurahub.com` | enrolled: Reactive Accelerator |
| Login `admin@nexurahub.com` | `/admin` overview |
| Checkout dummy pay | enroll হয়, 409 যদি আগে enrolled |

কোনো endpoint-এর exact body লাগলে `FRONTEND_API.md` দেখো।
