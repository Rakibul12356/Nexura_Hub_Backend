# Nexura Hub - Frontend Integration Guide & API Specification

> **Document Type:** Production Frontend Integration Contract  
> **Target Engineering Team:** Frontend Developers (React 19 + TypeScript + Redux Toolkit)  
> **Base API URL:** `http://localhost:8080`  
> **WebSocket URL:** `ws://localhost:8080/ws?token=<JWT_TOKEN>`  
> **Authentication Header:** `Authorization: Bearer <JWT_TOKEN>`

---

## 📋 Table of Contents
1. [General API Principles & Standard Error Response](#1-general-api-principles--standard-error-response)
2. [Authentication Endpoints (`/api/v1/auth`)](#2-authentication-endpoints)
3. [Public Course Catalog Endpoints (`/api/v1/courses`)](#3-public-course-catalog-endpoints)
4. [Student Actions (`/api/v1/courses`, `/api/v1/lessons`)](#4-student-actions)
5. [Instructor Studio (`/api/v1/instructor`)](#5-instructor-studio)
6. [Admin Control Hub (`/api/v1/admin`)](#6-admin-control-hub)
7. [Real-time Chat REST API (`/api/v1/conversations`)](#7-real-time-chat-rest-api)
8. [Real-time WebSocket Hub Protocol (`/ws`)](#8-real-time-websocket-hub-protocol)
9. [TypeScript Interfaces Cheat Sheet](#9-typescript-interfaces-cheat-sheet)

---

## 1. General API Principles & Standard Error Response

### Headers Requirement
For authenticated requests, send the JWT token in the `Authorization` HTTP header:
```http
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
Content-Type: application/json
```

### Standard Success Response Envelope
```json
{
  "status": "success",
  "data": { ... },
  "meta": { ... } // Optional for paginated feeds
}
```

### Standard Error Response Envelope
```json
{
  "status": "error",
  "message": "Human-readable error explanation here"
}
```

---

## 2. Authentication Endpoints

### 2.1 Register New User
- **HTTP Method:** `POST`
- **Path:** `/api/v1/auth/register`
- **Auth Required:** No

#### Request Body:
```json
{
  "firstName": "Karim",
  "lastName": "Rahman",
  "email": "karim@example.com",
  "password": "password123",
  "role": "student" // "student" | "instructor"
}
```

#### Success Response (`201 Created`):
```json
{
  "status": "success",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "user": {
      "id": "c7a8b9f1-3d2e-4b5a-9f8e-1a2b3c4d5e6f",
      "firstName": "Karim",
      "lastName": "Rahman",
      "email": "karim@example.com",
      "role": "student",
      "status": "active",
      "avatar": null,
      "bio": null,
      "occupation": null,
      "phone": null,
      "website": null,
      "createdAt": "2026-09-14T12:00:00Z",
      "updatedAt": "2026-09-14T12:00:00Z"
    }
  }
}
```

---

### 2.2 Login User
- **HTTP Method:** `POST`
- **Path:** `/api/v1/auth/login`
- **Auth Required:** No

#### Request Body:
```json
{
  "email": "NexuraHubAdmin@gmail.com", // Or student@nexurahub.com / instructor@nexurahub.com
  "password": "password123"
}
```

#### Success Response (`200 OK`):
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
      "role": "admin", // "student" | "instructor" | "admin"
      "status": "active"
    }
  }
}
```

---

### 2.3 Get Currently Authenticated Profile
- **HTTP Method:** `GET`
- **Path:** `/api/v1/auth/me`
- **Auth Required:** Yes (`Bearer <TOKEN>`)

#### Success Response (`200 OK`):
```json
{
  "status": "success",
  "data": {
    "id": "f659784d-1f7f-44f5-9e32-c1f7bb5aafc5",
    "firstName": "Nexura",
    "lastName": "Admin",
    "email": "NexuraHubAdmin@gmail.com",
    "role": "admin",
    "status": "active",
    "avatar": null,
    "bio": null,
    "createdAt": "2026-09-14T11:41:01Z",
    "updatedAt": "2026-09-14T11:41:01Z"
  }
}
```

---

### 2.4 Logout User
- **HTTP Method:** `POST`
- **Path:** `/api/v1/auth/logout`
- **Auth Required:** Optional (`Bearer <TOKEN>`)

#### Success Response (`200 OK`):
```json
{
  "success": true,
  "status": "success",
  "message": "Logged out successfully"
}
```

---

## 3. Public Course Catalog Endpoints

### 3.1 Fetch Published Course Feed (Paginated + Filters)
- **HTTP Method:** `GET`
- **Path:** `/api/v1/courses`
- **Query Parameters:**
  - `page` (default: `1`)
  - `limit` (default: `10`)
  - `search` (optional string search in title/subtitle)
  - `category` (optional category slug e.g. `web-dev`)
- **Auth Required:** No

#### Example Request:
`GET http://localhost:8080/api/v1/courses?page=1&limit=10&search=react&category=web-dev`

#### Success Response (`200 OK`):
```json
{
  "status": "success",
  "meta": {
    "total": 24,
    "page": 1,
    "totalPages": 3
  },
  "data": [
    {
      "id": "512bc9cd-54a6-457b-b4d6-8833cf93dc6c",
      "title": "Reactive Accelerator",
      "slug": "reactive-accelerator",
      "subtitle": "Master React 19 & Redux Toolkit with Go",
      "description": "Comprehensive fullstack engineering course.",
      "categoryId": 1,
      "category": "Web Development",
      "instructorId": "17dae99d-222c-4b9e-8edf-040109f6b8da",
      "instructor": {
        "id": "17dae99d-222c-4b9e-8edf-040109f6b8da",
        "name": "Tapas Adhikary",
        "avatar": null,
        "role": "instructor"
      },
      "thumbnail": "https://images.unsplash.com/photo-1633356122544-f134324a6cee",
      "price": 4500.00,
      "discountPrice": 3500.00,
      "isPublished": true,
      "learningPoints": [
        "Master React 19 concurrent features & server actions",
        "Build scalable Redux Toolkit global state architecture"
      ],
      "createdAt": "2026-09-14T11:38:05Z",
      "updatedAt": "2026-09-14T11:38:05Z"
    }
  ]
}
```

---

### 3.2 Fetch All Course Categories
- **HTTP Method:** `GET`
- **Path:** `/api/v1/courses/categories`
- **Auth Required:** No

#### Success Response (`200 OK`):
```json
{
  "status": "success",
  "data": [
    {
      "id": 1,
      "title": "Web Development",
      "slug": "web-dev",
      "thumbnail": "https://images.unsplash.com/photo-1633356122544-f134324a6cee",
      "createdAt": "2026-09-14T11:38:05Z"
    }
  ]
}
```

---

### 3.3 Fetch Single Course Details by Slug (Includes Modules & Lessons)
- **HTTP Method:** `GET`
- **Path:** `/api/v1/courses/:slug`
- **Example Path:** `/api/v1/courses/reactive-accelerator`
- **Auth Required:** No

#### Success Response (`200 OK`):
```json
{
  "status": "success",
  "data": {
    "id": "512bc9cd-54a6-457b-b4d6-8833cf93dc6c",
    "title": "Reactive Accelerator",
    "slug": "reactive-accelerator",
    "subtitle": "Master React 19 & Redux Toolkit with Go",
    "description": "Comprehensive fullstack engineering course.",
    "categoryId": 1,
    "category": "Web Development",
    "instructorId": "17dae99d-222c-4b9e-8edf-040109f6b8da",
    "instructor": {
      "id": "17dae99d-222c-4b9e-8edf-040109f6b8da",
      "name": "Tapas Adhikary",
      "avatar": null,
      "role": "instructor"
    },
    "thumbnail": "https://images.unsplash.com/photo-1633356122544-f134324a6cee",
    "price": 4500.00,
    "discountPrice": 3500.00,
    "isPublished": true,
    "learningPoints": [
      "Master React 19 concurrent features & server actions"
    ],
    "modules": [
      {
        "id": "m101-uuid",
        "courseId": "512bc9cd-54a6-457b-b4d6-8833cf93dc6c",
        "title": "Module 1: Modern Frontend Architecture",
        "description": "Getting started with React 19 and state management.",
        "position": 1,
        "isPublished": true,
        "lessons": [
          {
            "id": "l101-uuid",
            "moduleId": "m101-uuid",
            "title": "Lesson 1.1: Introduction to React 19",
            "description": "Overview of new React 19 APIs.",
            "videoUrl": "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
            "duration": "12:45",
            "isFree": true,
            "isPublished": true,
            "position": 1,
            "resources": [
              {
                "id": "res-1",
                "title": "React 19 CheatSheet.pdf",
                "url": "https://example.com/sheet.pdf",
                "type": "pdf",
                "size": "2.4 MB"
              }
            ]
          }
        ]
      }
    ]
  }
}
```

---

## 4. Student Actions

### 4.1 Purchase / Enroll in Course (Triggers 5% Admin Cut & Auto Group Chat Join)
- **HTTP Method:** `POST`
- **Path:** `/api/v1/courses/:courseId/enroll`
- **Auth Required:** Yes (`Bearer <STUDENT_TOKEN>`)

#### Request Body:
```json
{
  "paymentMethod": "bkash" // "bkash" | "nagad" | "card"
}
```

#### Success Response (`200 OK`):
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

### 4.2 Fetch My Enrolled Courses
- **HTTP Method:** `GET`
- **Path:** `/api/v1/courses/enrolled`
- **Auth Required:** Yes (`Bearer <STUDENT_TOKEN>`)

#### Success Response (`200 OK`):
```json
{
  "status": "success",
  "data": [
    {
      "id": "b1827439-019d-47fe-a112-ef1092837461",
      "studentId": "ed3a8903-ec52-44d2-94f6-0823053bf268",
      "courseId": "512bc9cd-54a6-457b-b4d6-8833cf93dc6c",
      "enrolledAt": "2026-09-14T11:41:01Z",
      "completedAt": null,
      "progressPercentage": 0.00,
      "course": {
        "title": "Reactive Accelerator",
        "slug": "reactive-accelerator",
        "subtitle": "Master React 19 & Redux Toolkit with Go",
        "thumbnail": "https://images.unsplash.com/photo-1633356122544-f134324a6cee",
        "price": 4500.00
      }
    }
  ]
}
```

---

### 4.3 Track Lesson Completion Status
- **HTTP Method:** `POST`
- **Path:** `/api/v1/lessons/:lessonId/complete`
- **Auth Required:** Yes (`Bearer <STUDENT_TOKEN>`)

#### Success Response (`200 OK`):
```json
{
  "status": "success",
  "message": "Lesson marked as completed"
}
```

---

## 5. Instructor Studio

### 5.1 Fetch Instructor Dashboard Analytics (95% Net Cut Aggregator)
- **HTTP Method:** `GET`
- **Path:** `/api/v1/instructor/dashboard/stats`
- **Auth Required:** Yes (`Bearer <INSTRUCTOR_TOKEN>`)

#### Success Response (`200 OK`):
```json
{
  "status": "success",
  "data": {
    "totalEarnings": 142500.00, // 95% net earnings total
    "activeStudents": 350,
    "totalCourses": 4,
    "recentEnrollments": [
      {
        "studentName": "John Student",
        "courseTitle": "Reactive Accelerator",
        "price": 4500.00,
        "instructorNet": 4275.00, // 95%
        "date": "2026-09-14T11:41:01Z"
      }
    ]
  }
}
```

---

### 5.2 Create New Course Draft
- **HTTP Method:** `POST`
- **Path:** `/api/v1/instructor/courses`
- **Auth Required:** Yes (`Bearer <INSTRUCTOR_TOKEN>`)

#### Request Body:
```json
{
  "title": "Fullstack Go & React Enterprise Masterclass",
  "subtitle": "Build high performance clean architecture backends",
  "description": "Learn clean architecture in Go with React frontend.",
  "category": "Software Engineering",
  "price": 5000.00,
  "discountPrice": 4200.00,
  "thumbnail": "https://images.unsplash.com/photo-1633356122544-f134324a6cee",
  "learningPoints": [
    "Master Go Clean Architecture",
    "PostgreSQL transaction cut logic",
    "WebSocket Room Management"
  ]
}
```

#### Success Response (`201 Created`):
```json
{
  "status": "success",
  "message": "Course created successfully",
  "data": {
    "id": "c2918471-2918-471a-bc01-927361827419",
    "title": "Fullstack Go & React Enterprise Masterclass",
    "slug": "fullstack-go-react-enterprise-masterclass-9821",
    "isPublished": true
  }
}
```

---

## 6. Admin Control Hub

### 6.1 Fetch Admin Overview Financial Analytics (GMV & 5% Net Commission)
- **HTTP Method:** `GET`
- **Path:** `/api/v1/admin/overview`
- **Auth Required:** Yes (`Bearer <ADMIN_TOKEN>`)

#### Success Response (`200 OK`):
```json
{
  "status": "success",
  "data": {
    "totalRevenue": 580000.00,       // Gross Sales (GMV)
    "adminNetCommission": 58000.00,  // 5% of instructor sales + 100% self courses
    "instructorPayouts": 522000.00,  // 95% instructor payouts
    "totalStudents": 1200,
    "totalInstructors": 45,
    "totalCourses": 80
  }
}
```

---

### 6.2 Approve & Publish Course
- **HTTP Method:** `PATCH`
- **Path:** `/api/v1/admin/courses/:id/approve`
- **Auth Required:** Yes (`Bearer <ADMIN_TOKEN>`)

#### Success Response (`200 OK`):
```json
{
  "status": "success",
  "message": "Course approved and published successfully",
  "data": {
    "courseId": "512bc9cd-54a6-457b-b4d6-8833cf93dc6c",
    "isPublished": true
  }
}
```

---

### 6.3 Update User Account Status (Suspend / Activate)
- **HTTP Method:** `PATCH`
- **Path:** `/api/v1/admin/users/:id/status`
- **Auth Required:** Yes (`Bearer <ADMIN_TOKEN>`)

#### Request Body:
```json
{
  "status": "suspended" // "active" | "suspended" | "pending"
}
```

#### Success Response (`200 OK`):
```json
{
  "status": "success",
  "message": "User status updated successfully",
  "data": {
    "userId": "ed3a8903-ec52-44d2-94f6-0823053bf268",
    "status": "suspended"
  }
}
```

---

## 7. Real-time Chat REST API

> 🛑 **Security Note:** Admin accounts are restricted from accessing chat features by middleware (`403 Forbidden`).

### 7.1 Fetch My Chat Conversations
- **HTTP Method:** `GET`
- **Path:** `/api/v1/conversations`
- **Auth Required:** Yes (`Bearer <STUDENT_OR_INSTRUCTOR_TOKEN>`)

#### Success Response (`200 OK`):
```json
{
  "status": "success",
  "data": [
    {
      "id": "91a82741-2918-471a-bc01-927361827419",
      "type": "group",
      "name": "Reactive Accelerator Discussion Group",
      "avatar": "https://images.unsplash.com/photo-1633356122544-f134324a6cee",
      "courseId": "512bc9cd-54a6-457b-b4d6-8833cf93dc6c",
      "lastMessage": {
        "id": "msg-101",
        "senderName": "Tapas Adhikary",
        "content": "Welcome everyone to the discussion group!",
        "timestamp": "10:45 AM",
        "reactions": []
      },
      "createdAt": "2026-09-14T11:41:01Z",
      "updatedAt": "2026-09-14T11:41:01Z"
    }
  ]
}
```

---

### 7.2 Fetch Room Messages (High Performance Cursor-Based Pagination)
- **HTTP Method:** `GET`
- **Path:** `/api/v1/conversations/:id/messages`
- **Query Parameters:**
  - `limit` (default: `20`)
  - `cursor` (optional message ID string to fetch messages older than cursor)
- **Auth Required:** Yes (`Bearer <TOKEN>`)

#### Example Request:
`GET http://localhost:8080/api/v1/conversations/91a82741-2918-471a-bc01-927361827419/messages?limit=20`

#### Success Response (`200 OK`):
```json
{
  "status": "success",
  "meta": {
    "limit": 20,
    "nextCursor": "msg-100-uuid"
  },
  "data": [
    {
      "id": "msg-101-uuid",
      "conversationId": "91a82741-2918-471a-bc01-927361827419",
      "senderId": "17dae99d-222c-4b9e-8edf-040109f6b8da",
      "senderName": "Tapas Adhikary",
      "senderAvatar": null,
      "senderRole": "instructor",
      "content": "Welcome everyone!",
      "imageUrl": null,
      "replyToId": null,
      "reactions": [
        {
          "emoji": "❤️",
          "count": 2,
          "users": ["ed3a8903-ec52-44d2-94f6-0823053bf268"]
        }
      ],
      "timestamp": "10:45 AM",
      "createdAt": "2026-09-14T11:45:00Z"
    }
  ]
}
```

---

### 7.3 Send REST Chat Message
- **HTTP Method:** `POST`
- **Path:** `/api/v1/conversations/messages`
- **Auth Required:** Yes (`Bearer <TOKEN>`)

#### Request Body:
```json
{
  "conversationId": "91a82741-2918-471a-bc01-927361827419",
  "content": "Hello instructor! Excited to learn.",
  "imageUrl": null
}
```

---

### 7.4 Toggle Emoji Reaction
- **HTTP Method:** `POST`
- **Path:** `/api/v1/conversations/reactions`
- **Auth Required:** Yes (`Bearer <TOKEN>`)

#### Request Body:
```json
{
  "conversationId": "91a82741-2918-471a-bc01-927361827419",
  "messageId": "msg-101-uuid",
  "emoji": "❤️"
}
```

---

## 8. Real-time WebSocket Hub Protocol

### Connection URL
Connect via standard browser WebSocket client:
```javascript
const ws = new WebSocket("ws://localhost:8080/ws?token=" + userJwtToken);
```

---

### Client Event Emitters (Sending to Server)

#### 1. Join Chat Room (`join_room`)
```json
{
  "event": "join_room",
  "data": {
    "conversationId": "91a82741-2918-471a-bc01-927361827419"
  }
}
```

#### 2. Send Real-time Message (`send_message`)
```json
{
  "event": "send_message",
  "data": {
    "conversationId": "91a82741-2918-471a-bc01-927361827419",
    "message": {
      "content": "Hello everyone!",
      "imageUrl": null
    }
  }
}
```

#### 3. Send Reaction (`send_reaction`)
```json
{
  "event": "send_reaction",
  "data": {
    "conversationId": "91a82741-2918-471a-bc01-927361827419",
    "messageId": "msg-101-uuid",
    "emoji": "❤️"
  }
}
```

#### 4. Broadcast Typing Indicator (`typing_status`)
```json
{
  "event": "typing_status",
  "data": {
    "conversationId": "91a82741-2918-471a-bc01-927361827419",
    "userId": "student-uuid",
    "userName": "John Student",
    "isTyping": true
  }
}
```

---

### Server Event Broadcasters (Receiving from Server)

#### 1. Receive Incoming Message (`receive_message`)
```json
{
  "event": "receive_message",
  "data": {
    "conversationId": "91a82741-2918-471a-bc01-927361827419",
    "message": {
      "id": "msg-102-uuid",
      "senderId": "student-uuid",
      "senderName": "John Student",
      "senderAvatar": null,
      "senderRole": "student",
      "content": "Hello everyone!",
      "timestamp": "10:46 AM",
      "reactions": []
    }
  }
}
```

#### 2. Receive Reaction Broadcast (`receive_reaction`)
```json
{
  "event": "receive_reaction",
  "data": {
    "conversationId": "91a82741-2918-471a-bc01-927361827419",
    "messageId": "msg-101-uuid",
    "emoji": "❤️",
    "userId": "student-uuid"
  }
}
```

---

## 9. TypeScript Interfaces Cheat Sheet

Copy and paste these exact types into your Frontend project (`src/types/api.ts`):

```typescript
export type UserRole = 'student' | 'instructor' | 'admin';
export type UserStatus = 'active' | 'suspended' | 'pending';

export interface User {
  id: string;
  firstName: string;
  lastName: string;
  email: string;
  role: UserRole;
  status: UserStatus;
  avatar?: string | null;
  bio?: string | null;
  createdAt: string;
  updatedAt: string;
}

export interface Course {
  id: string;
  title: string;
  slug: string;
  subtitle?: string | null;
  description?: string | null;
  categoryId?: number | null;
  category?: string;
  instructorId: string;
  instructor?: {
    id: string;
    name: string;
    avatar?: string | null;
    role: UserRole;
  };
  thumbnail: string;
  price: number;
  discountPrice?: number | null;
  isPublished: boolean;
  learningPoints: string[];
  modules?: Module[];
  createdAt: string;
  updatedAt: string;
}

export interface Module {
  id: string;
  courseId: string;
  title: string;
  description?: string | null;
  position: number;
  isPublished: boolean;
  lessons: Lesson[];
  createdAt: string;
}

export interface Lesson {
  id: string;
  moduleId: string;
  title: string;
  description?: string | null;
  videoUrl?: string | null;
  duration?: string | null;
  isFree: boolean;
  isPublished: boolean;
  position: number;
  resources: LessonResource[];
  completed?: boolean;
  createdAt: string;
}

export interface LessonResource {
  id: string;
  title: string;
  url: string;
  type: string;
  size: string;
}

export interface EmojiReaction {
  emoji: string;
  count: number;
  users: string[];
}

export interface ChatMessage {
  id: string;
  conversationId: string;
  senderId: string;
  senderName: string;
  senderAvatar?: string | null;
  senderRole: UserRole;
  content?: string | null;
  imageUrl?: string | null;
  replyToId?: string | null;
  reactions: EmojiReaction[];
  timestamp: string;
  createdAt: string;
}

export interface Conversation {
  id: string;
  type: 'direct' | 'group';
  name: string;
  avatar?: string | null;
  courseId?: string | null;
  lastMessage?: ChatMessage | null;
  createdAt: string;
  updatedAt: string;
}

export interface AdminOverviewStats {
  totalRevenue: number;
  adminNetCommission: number;
  instructorPayouts: number;
  totalStudents: number;
  totalInstructors: number;
  totalCourses: number;
}

export interface InstructorStats {
  totalEarnings: number;
  activeStudents: number;
  totalCourses: number;
  recentEnrollments: {
    studentName: string;
    courseTitle: string;
    price: number;
    instructorNet: number;
    date: string;
  }[];
}
```
