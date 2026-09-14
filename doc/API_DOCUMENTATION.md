# Nexura Hub - REST API & WebSocket Specifications

Production API documentation for **Nexura Hub LMS Backend** built with **Golang** & **Supabase PostgreSQL**.

---

## 🌐 Base URL
```
http://localhost:8080
```

---

## 🔑 1. Authentication Endpoints

### 1.1 Login User / Admin / Instructor
- **Endpoint:** `POST /api/v1/auth/login`
- **Headers:** `Content-Type: application/json`

#### Default Test Accounts:
- **Admin:** `NexuraHubAdmin@gmail.com` / `password123`
- **Student:** `student@nexurahub.com` / `password123`
- **Instructor:** `instructor@nexurahub.com` / `password123`

#### Request Body:
```json
{
  "email": "NexuraHubAdmin@gmail.com",
  "password": "password123"
}
```

#### Response `200 OK`:
```json
{
  "status": "success",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "user": {
      "id": "f659784d-1f7f-44f5-9e32-c1f7bb5aafc5",
      "firstName": "Nexura",
      "lastName": "Admin",
      "email": "NexuraHubAdmin@gmail.com",
      "role": "admin",
      "status": "active"
    }
  }
}
```

---

### 1.2 Register User
- **Endpoint:** `POST /api/v1/auth/register`
- **Headers:** `Content-Type: application/json`

#### Request Body:
```json
{
  "firstName": "Karim",
  "lastName": "Rahman",
  "email": "karim@example.com",
  "password": "password123",
  "role": "student"
}
```

---

### 1.3 Get Current User Profile
- **Endpoint:** `GET /api/v1/auth/me`
- **Headers:** `Authorization: Bearer <JWT_TOKEN>`

---

## 📚 2. Course Public Endpoints

### 2.1 List Courses
- **Endpoint:** `GET /api/v1/courses`
- **Query Params:** `page=1&limit=10&search=react&category=web-dev`

#### Response `200 OK`:
```json
{
  "status": "success",
  "meta": { "total": 1, "page": 1, "totalPages": 1 },
  "data": [
    {
      "id": "512bc9cd-54a6-457b-b4d6-8833cf93dc6c",
      "title": "Reactive Accelerator",
      "slug": "reactive-accelerator",
      "subtitle": "Master React 19 & Redux Toolkit with Go",
      "category": "Web Development",
      "thumbnail": "https://images.unsplash.com/photo-1633356122544-f134324a6cee",
      "price": 4500,
      "discountPrice": 3500,
      "isPublished": true,
      "learningPoints": [
        "Master React 19 concurrent features & server actions",
        "Build scalable Redux Toolkit global state architecture"
      ]
    }
  ]
}
```

---

### 2.2 Get Course Categories
- **Endpoint:** `GET /api/v1/courses/categories`

---

### 2.3 Get Course Details by Slug
- **Endpoint:** `GET /api/v1/courses/:slug`
- **Example:** `GET /api/v1/courses/reactive-accelerator`

---

## 🎓 3. Student Endpoints

### 3.1 Enroll in Course
- **Endpoint:** `POST /api/v1/courses/:courseId/enroll`
- **Headers:** `Authorization: Bearer <STUDENT_JWT_TOKEN>`
- **Request Body:**
```json
{
  "paymentMethod": "bkash"
}
```

#### Response `200 OK`:
```json
{
  "status": "success",
  "message": "Enrolled successfully",
  "data": {
    "enrollmentId": "b1827439-019d-47fe-a112-ef1092837461",
    "courseId": "512bc9cd-54a6-457b-b4d6-8833cf93dc6c",
    "groupConversationId": "91a82741-2918-471a-bc01-927361827419"
  }
}
```

---

### 3.2 Get Enrolled Courses
- **Endpoint:** `GET /api/v1/courses/enrolled`
- **Headers:** `Authorization: Bearer <STUDENT_JWT_TOKEN>`

---

### 3.3 Mark Lesson Complete
- **Endpoint:** `POST /api/v1/lessons/:lessonId/complete`
- **Headers:** `Authorization: Bearer <STUDENT_JWT_TOKEN>`

---

## 👨‍🏫 4. Instructor Studio Endpoints

### 4.1 Get Instructor Dashboard Stats
- **Endpoint:** `GET /api/v1/instructor/dashboard/stats`
- **Headers:** `Authorization: Bearer <INSTRUCTOR_JWT_TOKEN>`

#### Response `200 OK`:
```json
{
  "status": "success",
  "data": {
    "totalEarnings": 142500.00,
    "activeStudents": 350,
    "totalCourses": 4,
    "recentEnrollments": []
  }
}
```

---

### 4.2 Create New Course
- **Endpoint:** `POST /api/v1/instructor/courses`
- **Headers:** `Authorization: Bearer <INSTRUCTOR_JWT_TOKEN>`
- **Request Body:**
```json
{
  "title": "Fullstack Go & React Enterprise Masterclass",
  "subtitle": "Learn Clean Architecture in Go with React 19",
  "description": "Build high-performance web applications.",
  "category": "Software Engineering",
  "price": 5000.00,
  "discountPrice": 4200.00,
  "thumbnail": "https://images.unsplash.com/photo-1633356122544-f134324a6cee",
  "learningPoints": ["Master Go Clean Architecture", "PostgreSQL Commission System"]
}
```

---

## 🛡️ 5. Admin Control Hub Endpoints

### 5.1 Get Admin Overview Stats
- **Endpoint:** `GET /api/v1/admin/overview`
- **Headers:** `Authorization: Bearer <ADMIN_JWT_TOKEN>`

#### Response `200 OK`:
```json
{
  "status": "success",
  "data": {
    "totalRevenue": 580000.00,
    "adminNetCommission": 58000.00,
    "instructorPayouts": 522000.00,
    "totalStudents": 1200,
    "totalInstructors": 45,
    "totalCourses": 80
  }
}
```

---

### 5.2 Approve Course
- **Endpoint:** `PATCH /api/v1/admin/courses/:id/approve`
- **Headers:** `Authorization: Bearer <ADMIN_JWT_TOKEN>`

---

### 5.3 Update User Status
- **Endpoint:** `PATCH /api/v1/admin/users/:id/status`
- **Headers:** `Authorization: Bearer <ADMIN_JWT_TOKEN>`
- **Request Body:**
```json
{
  "status": "suspended"
}
```

---

## 💬 6. Real-time Chat & WebSocket Specs

### 6.1 Get User Conversations
- **Endpoint:** `GET /api/v1/conversations`
- **Headers:** `Authorization: Bearer <JWT_TOKEN>`

---

### 6.2 Get Historical Messages (Cursor Pagination)
- **Endpoint:** `GET /api/v1/conversations/:id/messages?limit=20&cursor=<MESSAGE_ID>`
- **Headers:** `Authorization: Bearer <JWT_TOKEN>`

---

### 6.3 Send Chat Message
- **Endpoint:** `POST /api/v1/conversations/messages`
- **Headers:** `Authorization: Bearer <JWT_TOKEN>`
- **Request Body:**
```json
{
  "conversationId": "<CONVERSATION_ID>",
  "content": "Hello everyone! Welcome to Nexura Hub.",
  "imageUrl": null
}
```

---

### 6.4 Send Emoji Reaction
- **Endpoint:** `POST /api/v1/conversations/reactions`
- **Headers:** `Authorization: Bearer <JWT_TOKEN>`
- **Request Body:**
```json
{
  "conversationId": "<CONVERSATION_ID>",
  "messageId": "<MESSAGE_ID>",
  "emoji": "❤️"
}
```

---

### ⚡ 6.5 WebSocket Server Connection
- **URL:** `ws://localhost:8080/ws?token=<JWT_TOKEN>`
- **Restriction:** Admin accounts are blocked from accessing chat endpoints & WebSockets.
