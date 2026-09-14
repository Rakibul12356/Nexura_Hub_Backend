# Nexura Hub LMS - Backend Architecture & Technical Specification

> **Document Version:** 2.0 (Modular Package-by-Feature Architecture)  
> **Target Audience:** Tech Leads, Senior Backend Engineers, DBAs  
> **Tech Stack:** Golang 1.26+, PostgreSQL (Supabase Cloud Pooler), Gin Gonic Framework, JWT Auth, Gorilla WebSocket  
> **Architecture Pattern:** Package-by-Feature / Modular Architecture (Domain-Driven Design)

---

## 📁 1. Project Directory Structure & Responsibility Matrix

```
Nexura_Hub_Backend/
├── cmd/
│   ├── api/
│   │   └── main.go                  // Application entry point: Server startup & dependency injection
│   └── seed/
│       └── main.go                  // DB DDL Migration & Data Seeder (Admin, Instructor, Student, Courses, Chat)
├── config/
│   └── config.go                    // Environment configuration loader (.env loader)
├── db/
│   └── migrations/
│       ├── 000001_init_schema.up.sql// DDL SQL: Enums, 10 Tables, Foreign Keys, Soft Deletes, Partial Indexes
│       └── 000001_init_schema.down.sql
├── doc/
│   ├── API_DOCUMENTATION.md         // Complete REST API & WebSocket specifications
│   ├── NEXURA_BACKEND_ARCHITECTURE.md // Architecture design document
│   └── Nexura_Hub_Postman_Collection.json // Importable Postman v2.1 API collection
├── internal/
│   ├── core/                        // Shared Infrastructure & Application Core
│   │   ├── config/                  // Core Config loader
│   │   ├── errors/                  // Domain sentinel errors (ErrUserNotFound, ErrAdminChatBlocked, etc.)
│   │   ├── middleware/              // Auth JWT, RBAC, Admin Chat Block, CORS middlewares
│   │   └── server/                  // Gin engine setup & server routing pipeline
│   └── modules/                     // Domain Modules (Package-by-Feature)
│       ├── admin/                   // Admin analytics, course approval, user status (models, repo, usecase, handler)
│       ├── auth/                    // User authentication & profile management (models, repo, usecase, handler)
│       ├── chat/                    // Real-time chat & WebSocket hub (models, repo, usecase, handler, hub, client, ws)
│       ├── course/                  // Public course catalog & category management (models, repo, usecase, handler)
│       ├── enrollment/              // Student enrollment & 5% commission TX (models, repo, usecase, handler)
│       └── instructor/              // Instructor 95% net earnings studio dashboard (models, repo, usecase, handler)
├── pkg/                             // Wrappers for Third-Party Libraries
│   ├── database/                    // PostgreSQL pool initializer (`database/sql` + `lib/pq`)
│   ├── hash/                        // Bcrypt password hasher & checker
│   └── jwt/                         // HMAC-SHA256 JWT generator & claim validator
├── .env                             // Production environment variables (Supabase DB URL, JWT Secret, Port)
└── go.mod                           // Go dependencies manifest
```

---

## 🏛️ 2. Package-by-Feature Architecture Rules

The codebase is organized into self-contained feature modules under `internal/modules/{feature_name}`:

Each feature module contains:
- `models.go` (Domain entities, enums & request/response DTOs)
- `repository.go` (Data Access Layer via raw SQL and `database/sql`)
- `usecase.go` (Application business logic & interface boundaries)
- `handler.go` (HTTP controllers using Gin)

Dependencies flow strictly from **Handler** -> **Usecase Interface** -> **Repository Interface**.

---

## 💰 3. Financial Business Logic: 5% Commission Cut Execution

When a student enrolls in a course, the backend executes an **atomic database transaction** (`EnrollmentRepository.ExecuteEnrollmentTx`):

- **If Course Creator is an Instructor:**
  - Admin Commission Rate: `5%` (`0.05`)
  - Instructor Net Earnings: `95%` (`0.95`)
- **If Course Creator is an Admin:**
  - Admin Commission Rate: `100%` (`1.00`)
  - Instructor Net Earnings: `0%` (`0.00`)

```go
// Atomic DB Transaction Steps:
// 1. Query course price & instructor role.
// 2. Compute Admin Amount = coursePrice * commissionRate.
// 3. Compute Instructor Amount = coursePrice - adminAmount.
// 4. Insert row into `transactions` table.
// 5. Insert row into `enrollments` table.
// 6. Auto-join student to `conversation_members` for the course group chat.
```

---

## 🔒 4. Role-Based Access Control (RBAC) & Security Guard

| User Role | Public Catalog | Course Player | Group Chat Room | Instructor Studio | Admin Control Hub |
| :--- | :---: | :---: | :---: | :---: | :---: |
| **Student** (`student`) | ✅ | ✅ | ✅ | ❌ | ❌ |
| **Instructor** (`instructor`) | ✅ | ✅ | ✅ | ✅ | ❌ |
| **Admin** (`admin`) | ✅ | ✅ | 🛑 **BLOCKED by Middleware** | ✅ | ✅ |

> **Security Guard:** `BlockAdminChatMiddleware` prevents `admin` accounts from joining chat rooms or opening WebSocket connections to protect student-instructor chat privacy.

---

## ⚡ 5. High-Performance Cursor-Based Chat Pagination

To eliminate PostgreSQL performance degradation with `OFFSET` on large chat datasets, message fetching uses **Cursor-based Pagination ($O(1)$ index lookup)**:

```sql
SELECT m.id, m.conversation_id, m.sender_id, u.first_name || ' ' || u.last_name, 
       m.content, m.image_url, COALESCE(m.reactions, '[]'::jsonb), m.created_at
FROM chat_messages m
JOIN users u ON m.sender_id = u.id
WHERE m.conversation_id = $1 AND m.created_at < (SELECT created_at FROM chat_messages WHERE id = $2)
ORDER BY m.created_at DESC
LIMIT 20;
```

---

## 🔑 6. Default Test Accounts Credentials Matrix

| Account Role | Email Address | Password | Purpose |
| :--- | :--- | :--- | :--- |
| **Admin** | `NexuraHubAdmin@gmail.com` | `password123` | Platform oversight, course approval, user moderation |
| **Student** | `student@nexurahub.com` | `password123` | Course purchasing, watching player, chat participation |
| **Instructor** | `instructor@nexurahub.com` | `password123` | Course creation, earnings analytics, studio dashboard |

---

## 📡 7. REST API & Real-time WebSocket Protocol

| Method | Endpoint | Auth Required | Target Role | Description |
| :--- | :--- | :---: | :---: | :--- |
| `POST` | `/api/v1/auth/register` | No | Public | Register new student or instructor |
| `POST` | `/api/v1/auth/login` | No | Public | Authenticate user & issue JWT token |
| `GET` | `/api/v1/auth/me` | Bearer JWT | Any | Get currently authenticated profile |
| `GET` | `/api/v1/courses` | No | Public | List published courses with search & category filters |
| `GET` | `/api/v1/courses/categories` | No | Public | Fetch all available course categories |
| `GET` | `/api/v1/courses/:slug` | No | Public | Fetch course details, modules & lessons |
| `POST` | `/api/v1/courses/:courseId/enroll` | Bearer JWT | Student | Purchase course & trigger 5% commission TX |
| `GET` | `/api/v1/courses/enrolled` | Bearer JWT | Student | Fetch enrolled courses list |
| `POST` | `/api/v1/lessons/:lessonId/complete` | Bearer JWT | Student | Track lesson completion status |
| `GET` | `/api/v1/instructor/dashboard/stats` | Bearer JWT | Instructor / Admin | Fetch 95% net earnings & active student stats |
| `POST` | `/api/v1/instructor/courses` | Bearer JWT | Instructor / Admin | Create new course draft |
| `GET` | `/api/v1/admin/overview` | Bearer JWT | Admin | Fetch GMV, net platform 5% commission, total users |
| `PATCH` | `/api/v1/admin/courses/:id/approve` | Bearer JWT | Admin | Approve & publish course |
| `PATCH` | `/api/v1/admin/users/:id/status` | Bearer JWT | Admin | Update user status (`active` / `suspended`) |
| `GET` | `/api/v1/conversations` | Bearer JWT | Student / Instructor | Fetch user chat conversations |
| `GET` | `/api/v1/conversations/:id/messages` | Bearer JWT | Student / Instructor | Fetch chat messages using cursor pagination |
| `POST` | `/api/v1/conversations/messages` | Bearer JWT | Student / Instructor | Send message to chat room |
| `POST` | `/api/v1/conversations/reactions` | Bearer JWT | Student / Instructor | Toggle emoji reaction on message |
| `GET` | `/ws?token=<JWT>` | WebSocket Token | Student / Instructor | Real-time WebSocket connection endpoint |

---
*Created for Nexura Hub Engineering Team.*
