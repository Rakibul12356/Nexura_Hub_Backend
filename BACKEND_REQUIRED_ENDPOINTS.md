# 🚀 Nexura Hub - Required Backend Endpoints Specification

This document provides a complete checklist and specification of all API endpoints required by the Frontend (`Nexura-Hub`) to seamlessly integrate with the Backend (`https://nexura-hub-backend.onrender.com/api/v1`).

---

## 📌 Status Legend
- 🟢 **Supported / Implemented**
- 🟡 **Partial / Needs Verification**
- 🔴 **Missing / Urgent Action Required**

---

## 1. 🔑 Authentication & User Management (`/auth`)

| Method | Endpoint | Description | Status |
| :--- | :--- | :--- | :---: |
| `POST` | `/api/v1/auth/login` | User login (returns JWT token & user info) | 🟢 |
| `POST` | `/api/v1/auth/register` | User registration (Student / Instructor) | 🟢 |
| `POST` | `/api/v1/auth/logout` | Invalidate token / session logout | 🟢 |
| `GET` | `/api/v1/auth/me` | Fetch logged-in user profile | 🟢 |
| `PUT` | `/api/v1/auth/profile` | Update profile information | 🟢 |
| `POST` | `/api/v1/auth/change-password` | Change user password | 🟢 |
| `POST` | `/api/v1/auth/refresh-token` | Refresh JWT access token | 🟢 |
| `POST` | `/api/v1/auth/forgot-password` | Request password reset email | 🟢 |
| `POST` | `/api/v1/auth/reset-password` | Reset password using token | 🟢 |

---

## 2. 📊 Instructor Dashboard Endpoints (`/dashboard`)

These endpoints are used by Instructors to view their courses, live classes, quizzes, and revenue statistics.

| Method | Endpoint | Description | Status |
| :--- | :--- | :--- | :---: |
| `GET` | `/api/v1/dashboard/stats` | Overview stats (Total Revenue, Total Students, Courses, Enrollments) | 🟢 |
| `GET` | `/api/v1/dashboard/courses` | List of courses created by the logged-in instructor | 🟢 |
| `GET` | `/api/v1/dashboard/lives` | List of live streams/classes created by the instructor | 🟢 |
| `GET` | `/api/v1/dashboard/quiz-sets` | List of quiz sets created by the instructor | 🟢 |
| `GET` | `/api/v1/dashboard/enrollments` | Log of student enrollments in instructor's courses | 🟢 |

---

## 3. 🛡️ Admin Dashboard & Governance (`/admin` / `/users`)

Endpoints required for super-admin analytics, platform GMV calculations, and user governance.

| Method | Endpoint | Description | Status |
| :--- | :--- | :--- | :---: |
| `GET` | `/api/v1/analytics/overview` | Platform-wide GMV, Admin net earnings (5% + 100%), Student counts | 🟢 |
| `GET` | `/api/v1/analytics/revenue` | Detailed transaction ledger (Course sales, 5% cut, 95% payout) | 🟢 |
| `GET` | `/api/v1/users` | List all platform users with role filters (Students, Instructors) | 🟢 |
| `GET` | `/api/v1/users/:id` | Fetch specific user details | 🟢 |
| `PATCH` | `/api/v1/users/:id/role` | Update user role (e.g., student -> instructor -> admin) | 🟢 |
| `DELETE` | `/api/v1/users/:id` | Suspend or delete user account | 🟢 |

---

## 4. 📚 Courses, Modules & Lessons (`/courses`, `/modules`, `/lessons`)

| Method | Endpoint | Description | Status |
| :--- | :--- | :--- | :---: |
| `GET` | `/api/v1/courses` | Public list of published courses (with category/search filters) | 🟢 |
| `GET` | `/api/v1/courses/:id` | Course details with modules, lessons & instructor details | 🟢 |
| `POST` | `/api/v1/courses` | Create a new course | 🟢 |
| `PUT` | `/api/v1/courses/:id` | Edit course metadata & settings | 🟢 |
| `DELETE` | `/api/v1/courses/:id` | Delete course | 🟢 |
| `PATCH` | `/api/v1/courses/:id/publish` | Toggle publish/unpublish status | 🟢 |
| `POST` | `/api/v1/courses/:id/modules` | Add module to course | 🟢 |
| `PUT` | `/api/v1/modules/:id` | Edit module title/order | 🟢 |
| `DELETE` | `/api/v1/modules/:id` | Delete module | 🟢 |
| `POST` | `/api/v1/modules/:id/lessons` | Add lesson to module | 🟢 |
| `PUT` | `/api/v1/lessons/:id` | Update lesson video URL / content | 🟢 |
| `DELETE` | `/api/v1/lessons/:id` | Delete lesson | 🟢 |

---

## 5. 🎓 Enrollments & Learning Progress (`/enrollments`, `/lessons`)

| Method | Endpoint | Description | Status |
| :--- | :--- | :--- | :---: |
| `POST` | `/api/v1/courses/:id/enroll` | Enroll current user into a course | 🟢 |
| `GET` | `/api/v1/user/enrolled-courses` | List of courses currently enrolled by student | 🟢 |
| `POST` | `/api/v1/lessons/:id/complete` | Mark lesson as completed / update progress % | 🟢 |
| `GET` | `/api/v1/enrollments/:courseId/status` | Check if user is enrolled & get progress | 🟢 |

---

## 6. 📁 Media Upload Endpoints (`/upload`)

| Method | Endpoint | Description | Status |
| :--- | :--- | :--- | :---: |
| `POST` | `/api/v1/upload/image` | Upload course thumbnail / user avatar (returns image URL) | 🟢 |
| `POST` | `/api/v1/upload/video` | Upload lesson video file (or HLS stream link) | 🟢 |
| `POST` | `/api/v1/upload/pdf` | Upload downloadable lesson attachment PDF/resource | 🟢 |
| `DELETE` | `/api/v1/upload/delete` | Delete uploaded file by file ID / key | 🟢 |

---

## 📤 Deliverables
1. All required endpoints are 100% implemented, tested, and routed under `/api/v1`.
2. Full resolution report delivered in `BACKEND_ENDPOINTS_RESOLUTION_REPORT.md`.
3. CORS is enabled for all frontend origins.
