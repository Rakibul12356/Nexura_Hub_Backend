# Nexura Hub — Frontend API Guide

React 19 + TypeScript frontend-এর জন্য backend contract। Base URL, auth, প্রতিটা endpoint, request/response উদাহরণ এখানে আছে।

| | Value |
|---|---|
| **Backend host** | `https://nexura-hub-backend.onrender.com` |
| **API base (`VITE_API_URL`)** | `https://nexura-hub-backend.onrender.com/api/v1` |
| **Socket / WS (`VITE_SOCKET_URL`)** | `https://nexura-hub-backend.onrender.com` (no `/api/v1`) |
| **Health** | `GET https://nexura-hub-backend.onrender.com/health` |
| **Local API** | `http://localhost:5000/api/v1` |

Frontend `.env`:

```
VITE_API_URL=https://nexura-hub-backend.onrender.com/api/v1
VITE_SOCKET_URL=https://nexura-hub-backend.onrender.com
```

Path params এখন সব `:id` (Gin wildcard conflict fix)। `:courseId` / `:moduleId` / `:lessonId` / `:quizId` আর আলাদা route নয়।

Frontend unwrap করে: `response.data.data` আগে, তারপর `response.data`।

---

## 1. Auth header & cookies

Protected request:

```
Authorization: Bearer <access_token>
```

Cookie (frontend store করে):

| Cookie | TTL |
|---|---|
| `nexurahub_token` | 7 days |
| `nexurahub_refresh_token` | 30 days |

JSON-এও `token` + `refreshToken` আসে — দুটোই save করো।

JWT payload: `{ "sub": "<user-uuid>", "role": "student|instructor|admin", "email": "...", "userId": "..." }`

---

## 2. Standard envelope

**Success**

```json
{
  "success": true,
  "statusCode": 200,
  "message": "OK",
  "data": {}
}
```

**List + pagination**

```json
{
  "success": true,
  "statusCode": 200,
  "message": "OK",
  "data": [],
  "meta": { "page": 1, "limit": 20, "total": 120, "totalPages": 6 }
}
```

**Error**

```json
{
  "success": false,
  "statusCode": 400,
  "message": "Human readable error",
  "error": { "code": "VALIDATION_ERROR", "details": null }
}
```

| HTTP | অর্থ |
|---|---|
| 200 / 201 | Success |
| 400 | Validation |
| 401 | Token missing/expired → cookies clear করে `/login` |
| 403 | Wrong role / suspended / admin chat blocked |
| 404 | Not found |
| 409 | Duplicate (email, coupon, already enrolled) |
| 422 | Business rule (coupon invalid, quiz/certificate not ready) |
| 500 | Server error → toast |

IDs সব **UUID string**। টাকা BDT, `NUMERIC` → JSON number (e.g. `3999.2`)।

---

## 3. Seed login (QA)

| Role | Email | Password | Redirect after login |
|---|---|---|---|
| admin | `admin@nexurahub.com` | `password123` | `/admin` |
| instructor | `tapas@nexurahub.com` | `password123` | `/dashboard` |
| student | `student@nexurahub.com` | `password123` | `/account/enrolled-courses` |
| student | `sadia@nexurahub.com` | `password123` | `/account/enrolled-courses` |

Coupons: `NEXURA20` (20%), `FLAT500` (৳500)।

Admin **chat use করতে পারে না** — socket disconnect + REST 403।

---

## 4. Auth

### `POST /auth/register` — Public

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

`role` শুধু `student` | `instructor`। Admin self-register হবে না।

**201**

```json
{
  "success": true,
  "statusCode": 201,
  "message": "Registered successfully",
  "data": {
    "user": {
      "id": "uuid",
      "firstName": "John",
      "lastName": "Doe",
      "email": "m@example.com",
      "role": "student",
      "avatar": null,
      "bio": null,
      "occupation": null,
      "phone": null,
      "website": null
    },
    "token": "<jwt>",
    "refreshToken": "<jwt>"
  }
}
```

### `POST /auth/login` — Public

```json
{ "email": "m@example.com", "password": "secret1", "rememberMe": true }
```

Response shape login/register-এর মতোই (`user`, `token`, `refreshToken`)।  
Suspended user → **403**. Wrong password → **401**.

### `POST /auth/logout` — User (token optional)

Body optional: `{ "refreshToken": "..." }`  
**200** `{ "message": "Logged out successfully" }` inside `data`.

### `GET /auth/me` — User

`data` = `User` object (same fields as login `user`).

### `PUT /auth/profile` — User

Partial: `firstName, lastName, email, bio, occupation, phone, website, avatar, designation`  
**200** `data` = updated `User`.

### `POST /auth/change-password` — User

```json
{ "currentPassword": "old", "newPassword": "secret2" }
```

### `POST /auth/forgot-password` — Public

```json
{ "email": "m@example.com" }
```

Dev mode-এ `data.resetToken` ফেরত আসতে পারে।

### `POST /auth/reset-password` — Public

```json
{ "token": "...", "newPassword": "secret2" }
```

### `POST /auth/refresh-token` — Public

```json
{ "refreshToken": "<jwt>" }
```

**200** `data`: `{ "token", "refreshToken", "user" }`

### `POST /auth/verify-email` — Public

```json
{ "token": "..." }
```

### `POST /upload/avatar` — User (multipart `file`)

```json
{ "success": true, "data": { "url": "https://.../file.jpg", "size": "2.4 MB" } }
```

---

## 5. Categories (Home / Courses filters)

### `GET /categories` — Public

`data` = array:

```json
{
  "id": "uuid",
  "title": "Web Development",
  "thumbnail": "https://...",
  "value": "web-development",
  "label": "Web Development"
}
```

Alias: `GET /courses/categories`

### Admin CRUD

| Method | Path | Auth |
|---|---|---|
| GET | `/categories/:id` | Public |
| POST | `/categories` | Admin `{ title, slug?, thumbnail }` |
| PUT | `/categories/:id` | Admin |
| DELETE | `/categories/:id` | Admin |

---

## 6. Courses (storefront + studio)

### `GET /courses` — Public (published). Admin দেখে drafts-ও।

Query:

```
?page=1&limit=20&search=&sort=createdAt:desc
&category=web-development
&price=free|paid
&sort=price-asc|price-desc|popular
&isFeatured=true&isPublished=true
&instructorId=<uuid>
&creatorType=instructor|admin
&mine=true
```

**200** `data` = `Course[]` + `meta`.

List item:

```json
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
```

Admin list-এ extra: `enrollmentsCount`, `totalRevenue`, `adminEarnings`.

### `GET /courses/:id` — Public (`:id` = UUID **or slug**)

Full course + `modules[].lessons[]` + `instructor` + `reviews` + `learningPoints`.  
Paid `videoUrl` hide থাকে যদি enrolled না হও / `isFree` না হয় / owner-admin না হয়।

Lesson:

```json
{
  "id": "uuid",
  "title": "Course Overview",
  "videoUrl": "https://...",
  "duration": "12:40",
  "isFree": true,
  "isPublished": true,
  "position": 1,
  "completed": false,
  "quizSetId": null,
  "hasQuiz": false,
  "questions": [],
  "resources": []
}
```

### `POST /courses` — Instructor / Admin

```json
{ "title": "My Course", "description": "..." }
```

**201** `data` = course (`id` নিয়ে `/dashboard/courses/:id` এ যাও)।  
`creatorType` actor role থেকে set হয়। `instructor_id` = current user। Draft (`isPublished: false`)।

### `PUT /courses/:id` — Owner / Admin

Partial: `title, description, thumbnail/image, category, categoryId, price, discountPrice, isPublished, isFeatured (admin), learningPoints`

### `DELETE /courses/:id` — Owner / Admin (soft delete)

### `PATCH /courses/:id/publish` | `PATCH /courses/:id/unpublish`

Optional body: `{ "isPublished": true }`

### `PUT /courses/:id/modules/reorder`

```json
{ "moduleIds": ["uuid1", "uuid2"] }
```

---

## 7. Modules & lessons

| Method | Path | Auth | Body / notes |
|---|---|---|---|
| GET | `/courses/:id/modules` | Public titles | |
| GET | `/modules/:id` | Owner / Admin / enrolled | module + lessons |
| POST | `/courses/:id/modules` | Owner / Admin | `{ title, description? }` |
| PUT | `/modules/:id` | Owner / Admin | `{ title, description, isPublished }` |
| DELETE | `/modules/:id` | Owner / Admin | |
| PUT | `/modules/:id/lessons/reorder` | Owner / Admin | `{ lessonIds: [] }` |
| GET | `/lessons/:id` | Enrolled / isFree / owner | |
| POST | `/modules/:id/lessons` | Owner / Admin | `{ title, description, videoUrl, duration, isFree, isPublished, quizSetId }` |
| PUT | `/lessons/:id` | Owner / Admin | same fields |
| DELETE | `/lessons/:id` | Owner / Admin | |
| POST | `/lessons/:id/resources` | Owner / Admin | `{ title, type, url, size, fileName, content }` `type`: github\|link\|pdf\|richtext |
| DELETE | `/lessons/:id/resources/:resourceId` | Owner / Admin | |
| PATCH | `/lessons/:id/complete` | Enrolled | `{ completed: true }` optional. Alias: `POST /lessons/:id/complete` |

Complete response:

```json
{ "completed": true, "progress": 16.67 }
```

Progress = completed published lessons / total published lessons × 100। 100 হলে enrollment `completedAt` set।

---

## 8. Player extras (notes, Q&A, reviews, quiz)

### Notes — enrolled, own notes only

```
GET    /lessons/:id/notes
POST   /lessons/:id/notes     { "timestamp": 45, "text": "Review Virtual DOM" }
DELETE /lessons/:id/notes/:noteId
```

Note:

```json
{ "id": "uuid", "timestamp": 45, "text": "Review Virtual DOM", "createdAt": "07:12 PM" }
```

### Discussions

```
GET  /lessons/:id/discussions
POST /lessons/:id/discussions          { "content": "..." }
POST /discussions/:id/replies               { "content": "..." }
POST /discussions/:id/upvote                toggle
```

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

### Reviews

```
GET    /courses/:id/reviews          Public
POST   /courses/:id/reviews          Enrolled  { "rating": 1-5, "comment": "..." }
PUT    /reviews/:id                  Author
DELETE /reviews/:id                  Author / Admin
```

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

### Quizzes

| Method | Path | Auth |
|---|---|---|
| GET | `/quizzes` | Instructor own / Admin |
| GET | `/quizzes/:id` | Owner/Admin; student পায় **without** `isCorrect` |
| POST | `/quizzes` | `{ title, description }` → `{ id }` |
| PUT | `/quizzes/:id` | |
| DELETE | `/quizzes/:id` | |
| POST | `/quizzes/:id/questions` | see body below |
| DELETE | `/quizzes/:id/questions/:qId` | |
| POST | `/quizzes/:id/submit` | Enrolled |

Question body:

```json
{
  "title": "Which hook is used for side effects?",
  "description": "...",
  "points": 5,
  "options": [
    { "label": "useEffect", "isCorrect": true },
    { "label": "useState", "isCorrect": false }
  ]
}
```

Submit:

```json
{ "lessonId": "uuid", "answers": { "<questionId>": "<optionId>" } }
```

**200**

```json
{ "score": 80, "total": 100, "passed": true, "correctCount": 4, "totalQuestions": 5 }
```

Pass rule: **score >= 50%**.

Dashboard list: `GET /dashboard/quiz-sets`

---

## 9. Dummy payment & enrollment

**কোনো real gateway নেই।** Checkout = `POST /payments/dummy`। `gateway` শুধু invoice label।

`POST /payments/init` একই handler (alias)।

### `POST /coupons/validate` — User

```json
{ "code": "NEXURA20", "courseId": "uuid-or-slug" }
```

**200**

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

Invalid/expired/maxed → **422** `{ "message": "Invalid promo coupon code." }`

### `POST /payments/dummy` — User

```json
{ "courseId": "uuid-or-slug", "couponCode": "NEXURA20", "gateway": "stripe" }
```

**201**

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
    "enrollment": { "id": "uuid", "isEnrolled": true, "paymentStatus": "paid" },
    "split": {
      "creatorType": "instructor",
      "adminCommissionRate": 0.05,
      "adminCommissionAmount": 199.96,
      "instructorEarnings": 3799.24
    },
    "adminWallet": { "credited": 199.96, "balanceAfter": 186199.96 }
  }
}
```

Invoice PDF: `transactionId`, `amountPaid`, `courseTitle`, `paidAt`।

Errors: **400** unpublished, **409** already enrolled, **422** bad coupon.

Commission:

| Course creator | Admin | Instructor |
|---|---|---|
| instructor | 5% | 95% |
| admin | 100% | 0% |

### `GET /payments/:id` — Payer / Admin (UUID or `TXN-DUMMY-...`)

Receipt object (same money fields)।

### `POST /enrollments` — User, **free course only** (`price=0`)

```json
{ "courseId": "uuid" }
```

Paid course এখানে দিলে 422 — dummy pay use করো।

### `GET /enrollments/:id/status` — User (`:id` = course UUID or slug)

```json
{ "isEnrolled": true, "id": "uuid", "progress": 45, "paymentStatus": "paid" }
```

Not enrolled: `{ "isEnrolled": false }`

### `GET /user/enrolled-courses` — User

`data` = `EnrolledCourse[]` (Course fields +):

```json
{
  "enrolledDate": "2024-01-10",
  "completedModules": 2,
  "totalModules": 8,
  "completedQuizzes": 3,
  "totalQuizzes": 6,
  "quizScore": 80,
  "otherScore": 10,
  "totalScore": 90,
  "progress": 62,
  "paymentStatus": "paid"
}
```

---

## 10. Instructor dashboard (`/dashboard/*`)

Access: `instructor` **or** `admin`। Header: `Authorization: Bearer`.

### `GET /dashboard/stats`

```json
{
  "totalCourses": 4,
  "totalEnrollments": 128,
  "totalRevenue": 185000,
  "instructorEarnings": 175750,
  "totalStudents": 96
}
```

`instructorEarnings` = 95% dummy wallet earnings। `totalRevenue` = gross GMV of own sales.

### `GET /dashboard/courses`

Own courses (admin: all unpublished too).

### `GET /dashboard/enrollments?limit=10`

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

Course-specific: `GET /courses/:id/enrollments?search=`

### `GET /dashboard/wallet` — Instructor dummy 95% balance

```json
{
  "ownerType": "instructor",
  "balance": 3799.24,
  "currency": "BDT",
  "isDummy": true,
  "lifetimeCredits": 3799.24,
  "recent": [
    {
      "id": "uuid",
      "entryType": "credit",
      "amount": 3799.24,
      "balanceAfter": 3799.24,
      "note": "95% dummy earnings · TXN-DUMMY-18402911",
      "createdAt": "2026-09-15T13:00:00.000Z"
    }
  ]
}
```

### Lives

```
GET    /dashboard/lives
POST   /lives   { title, description, date, time, meetingLink, courseId?, duration }
GET    /lives/:id
PUT    /lives/:id
DELETE /lives/:id
```

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

`GET /analytics/instructor` — own dummy earnings summary.  
`GET /analytics/student-progress` — student progress rows.

---

## 11. Admin (`/admin/*`)

**admin only.**

### `GET /analytics/overview`  (alias `GET /admin/overview`)

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
  "growthRate": { "revenue": 24.8, "students": 12.1, "courses": 0 },
  "monthlyGrowth": [
    { "month": "Jan", "gmv": 120000, "adminRevenue": 18000, "instructorEarnings": 102000, "students": 80, "enrollments": 140 }
  ],
  "categoryStats": [
    { "name": "Web Development", "count": 12, "revenue": 450000, "color": "#0ea5e9" }
  ],
  "recentTransactions": []
}
```

### `GET /admin/wallet`

Same wallet shape as instructor, `ownerType: "admin"`। এটাই student-এর dummy টাকা।

### `GET /analytics/revenue?search=&creatorType=&page=`

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
  "transactions": [
    {
      "id": "TXN-DUMMY-18402911",
      "courseId": "uuid",
      "courseTitle": "Reactive Accelerator",
      "creatorType": "instructor",
      "instructorName": "Tapas Adhikary",
      "studentName": "Rahim Ahmed",
      "studentEmail": "rahim@example.com",
      "price": 3999.2,
      "adminCommissionRate": 0.05,
      "adminCommissionAmount": 199.96,
      "instructorEarnings": 3799.24,
      "date": "2026-09-15",
      "status": "completed",
      "paymentMethod": "Dummy"
    }
  ]
}
```

CSV frontend-এ এই list থেকে বানাও।

### Users

```
GET    /users?role=student|instructor|admin&search=&page=&limit=
GET    /users/:id
PATCH  /users/:id/role     { "role": "instructor" }
PATCH  /users/:id/status   { "status": "active"|"suspended" }
DELETE /users/:id
```

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

নিজের admin account delete যাবে না। Last admin demote/delete → 422।

### Coupons (Admin Overview)

```
GET    /coupons
POST   /coupons     { "code":"NEXURA20", "discountType":"percentage"|"flat", "discountValue":20, "expiryDate":"2026-12-31", "maxRedemptions":500 }
PATCH  /coupons/:id { "isActive": false }
DELETE /coupons/:id
```

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

---

## 12. Chat (student + instructor only)

REST prefix: `/chat/...`

| Method | Path | Purpose |
|---|---|---|
| GET | `/chat/conversations` | Inbox: lastMessage + unreadCount |
| GET | `/chat/conversations/:id` | Conversation + members |
| GET | `/chat/conversations/:id/messages?page&limit=50` | Oldest → newest |
| POST | `/chat/conversations/direct` | `{ userId }` get-or-create DM |
| POST | `/chat/groups` | Instructor `{ courseId, groupName }` |
| POST | `/chat/groups/:id/join` | Enrolled student |
| POST | `/chat/conversations/:id/read` | unreadCount = 0 |
| POST | `/chat/messages` | `{ conversationId, content, imageUrl?, replyToId? }` |
| DELETE | `/chat/messages/:id` | Unsend (sender) |
| POST | `/chat/messages/:id/reactions` | `{ emoji }` toggle |
| POST | `/chat/upload-image` | multipart image, max 5MB |

Conversation:

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
    { "id": "uuid", "name": "Tapas Adhikary", "avatar": "https://...", "role": "instructor", "isOnline": false }
  ],
  "lastMessage": {},
  "unreadCount": 2,
  "updatedAt": "07:15 PM"
}
```

Message:

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
  "replyTo": { "id": "uuid", "senderName": "Tapas", "content": "Welcome" },
  "attachment": { "type": "image", "url": "https://...", "name": "shot.png" },
  "reactions": [{ "emoji": "👍", "count": 2, "users": ["uuid1", "uuid2"] }]
}
```

Enroll হলে auto: course group join + instructor DM + welcome messages।

### Socket.IO

```js
io(VITE_SOCKET_URL, {
  transports: ["websocket", "polling"],
  auth: { userId, userRole },
  extraHeaders: { Authorization: `Bearer ${token}` },
  query: { token, userId, userRole }
})
```

Admin connect হলে immediately disconnect।

**Client → Server**

| Event | Payload |
|---|---|
| `join_room` | `{ conversationId }` |
| `leave_room` | `{ conversationId }` |
| `send_message` | `{ conversationId, message: { content, imageUrl? } }` |
| `typing_status` | `{ conversationId, userId, userName, isTyping }` |
| `send_reaction` | `{ conversationId, messageId, emoji, userId }` |

Server `message.id` ignore করে UUID generate করে persist করে।

**Server → Client**

| Event | Payload |
|---|---|
| `receive_message` | `{ conversationId, message }` |
| `user_typing` | `{ conversationId, userId, userName, isTyping }` |
| `user_presence` | `{ userId, isOnline }` |
| `receive_reaction` | `{ conversationId, messageId, emoji, userId }` |

Legacy WS: `GET /ws?token=` (gorilla). Prefer Socket.IO `/socket.io/`.

---

## 13. Uploads

Multipart field name: **`file`**

| Path | Who | Max |
|---|---|---|
| `POST /upload/image` | User | 5MB images |
| `POST /upload/avatar` | User | 5MB |
| `POST /upload/pdf` | Instructor / Admin | 20MB |
| `POST /upload/video` | Instructor / Admin | 500MB |
| `POST /upload/file` | Instructor / Admin | 50MB |
| `POST /upload/delete` | `{ fileUrl }` | |

```json
{ "success": true, "data": { "url": "https://host/uploads/<id>.jpg", "size": "2.4 MB" } }
```

Video: mp4 URL (HLS pipeline phase 2)। YouTube embed URL lesson `videoUrl`-এও চলবে।

---

## 14. Notifications

```
GET   /notifications
PATCH /notifications/:id/read
POST  /notifications/read-all
GET   /notifications/unread-count     → { "count": 3 }
```

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

`type`: `live | discussion | quiz | system | enrollment | payment`

---

## 15. Certificates

```
POST /certificates/:courseId     enrolled + progress 100
GET  /certificates/me/:courseId
GET  /certificates/verify/:publicId    Public (QR)
```

```json
{
  "publicId": "NEX-123456",
  "studentName": "Rahim Ahmed",
  "courseTitle": "Reactive Accelerator",
  "completionDate": "September 15, 2026",
  "verifyUrl": "https://nexurahub.com/verify/NEX-123456"
}
```

PDF/QR frontend generate করে। Backend `publicId` persist করে যাতে verify কাজ করে। Not ready → **422**.

---

## 16. Instructors public

```
GET /instructors/:id
GET /instructors/:id/courses     published only
```

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
  "rating": 4.9
}
```

`/inst-profile` এখনো `:id` না থাকলে `GET /instructors/:id` + query `?id=` পরে frontend এ add করো।

---

## 17. Page → endpoint map

| Frontend route | Endpoints |
|---|---|
| `/login` | `POST /auth/login` |
| `/register` | `POST /auth/register` |
| `/` Home | `GET /categories`, `GET /courses?isFeatured=true&isPublished=true&limit=8` |
| `/courses` | `GET /courses`, `GET /categories` |
| `/courses/:courseId` | `GET /courses/:id`, `GET /enrollments/:id/status`, `GET /courses/:id/reviews`, `POST /payments/dummy` |
| `/inst-profile` | `GET /instructors/:id`, `GET /courses?instructorId=` |
| `/account` | `GET /auth/me`, `PUT /auth/profile`, `POST /auth/change-password` |
| `/account/enrolled-courses` | `GET /user/enrolled-courses` |
| `/messages` `/dashboard/messages` | Chat REST + Socket.IO |
| `/player/:slug/:lessonId` | `GET /courses/:id`, `GET /lessons/:id`, `PATCH /lessons/:id/complete`, quiz submit, notes, discussions, reviews, certificates |
| `/dashboard` | `GET /dashboard/stats`, `/dashboard/enrollments?limit=10`, `/dashboard/courses` |
| `/dashboard/courses/add` | `POST /courses` |
| `/dashboard/courses/:id` | GET/PUT course, modules, publish, `POST /upload/image` |
| `/dashboard/lives*` | lives CRUD |
| `/dashboard/quiz-sets*` | quizzes CRUD |
| `/admin` | `/analytics/overview`, `/admin/wallet`, `/users`, `/courses`, `/coupons` |
| `/admin/users` | users CRUD |
| `/admin/revenue` | `/analytics/revenue` |
| `/admin/courses` | courses + publish/feature/delete |

---

## 18. Axios snippet

```ts
const api = axios.create({
  baseURL:
    import.meta.env.VITE_API_URL ||
    "https://nexura-hub-backend.onrender.com/api/v1",
});

api.interceptors.request.use((config) => {
  const token = Cookies.get("nexurahub_token");
  if (token) config.headers.Authorization = `Bearer ${token}`;
  return config;
});

api.interceptors.response.use(
  (res) => res,
  async (err) => {
    if (err.response?.status === 401) {
      Cookies.remove("nexurahub_token");
      Cookies.remove("nexurahub_refresh_token");
      window.location.href = "/login";
    }
    return Promise.reject(err);
  }
);

export const unwrap = <T>(res: AxiosResponse): T =>
  (res.data?.data ?? res.data) as T;
```

Health check (no `/api/v1`): `GET /health` → `{ "status": "ok", "service": "Nexura Hub API Server" }`।
