# 🔑 Nexura Hub - API Seed Credentials & Response Reference

**Date:** 15 September 2026  
**Backend API URL:** `https://nexura-hub-backend.onrender.com/api/v1` (Local: `http://localhost:8080/api/v1`)  
**Postman Collection File:** [doc/Nexura_Hub_Postman_Collection.json](file:///c:/Users/rakib/Nexura_Hub_Backend/doc/Nexura_Hub_Postman_Collection.json)

---

## 🔑 1. User Credentials & Seed Profiles

| Role | Email | Password | Full Name | Phone | Occupation & Bio |
| :--- | :--- | :--- | :--- | :--- | :--- |
| 🛡️ **Admin** | `NexuraHubAdmin@gmail.com` | `password123` | Nexura Admin | `+8801700000000` | Super Admin & Platform Administrator |
| 👨‍🏫 **Instructor** | `instructor@nexurahub.com` | `password123` | Tapas Adhikary | `+8801712345678` | Principal Software Architect (Senior Fullstack Lead) |
| 🎓 **Student** | `student@nexurahub.com` | `password123` | Karim Rahman | `+8801812345678` | CS Student & Junior Fullstack Developer |

---

## 🌐 2. Endpoint Categories & Detailed JSON Responses

### 2.1 🔑 Authentication & Profile

#### `POST /api/v1/auth/login` (Instructor Login Example)
**Request Body:**
```json
{
  "email": "instructor@nexurahub.com",
  "password": "password123"
}
```
**Response (`200 OK`):**
```json
{
  "status": "success",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "user": {
      "id": "d4a8b9f1-3d2e-4b5a-9f8e-1a2b3c4d5e6f",
      "firstName": "Tapas",
      "lastName": "Adhikary",
      "email": "instructor@nexurahub.com",
      "role": "instructor",
      "status": "active",
      "avatar": "https://images.unsplash.com/photo-1534528741775-53994a69daeb?w=500",
      "bio": "Senior Fullstack Engineer & Tech Lead with 10+ years of experience in Go, React & Cloud Native Systems.",
      "occupation": "Principal Software Architect",
      "phone": "+8801712345678",
      "website": "https://tapasadhikary.com",
      "createdAt": "2026-08-26T10:00:00Z",
      "updatedAt": "2026-09-15T18:00:00Z"
    }
  }
}
```

#### `GET /api/v1/auth/me` (Student Profile Example)
**Headers:** `Authorization: Bearer <TOKEN>`  
**Response (`200 OK`):**
```json
{
  "status": "success",
  "data": {
    "id": "c7a8b9f1-3d2e-4b5a-9f8e-1a2b3c4d5e6f",
    "firstName": "Karim",
    "lastName": "Rahman",
    "email": "student@nexurahub.com",
    "role": "student",
    "status": "active",
    "avatar": "https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?w=500",
    "bio": "Aspiring Fullstack Developer learning React 19, TypeScript and Golang microservices.",
    "occupation": "CS Student & Junior Developer",
    "phone": "+8801812345678",
    "website": "https://github.com/karim-rahman",
    "createdAt": "2026-09-05T10:00:00Z",
    "updatedAt": "2026-09-15T18:00:00Z"
  }
}
```

---

### 2.2 📊 Instructor Dashboard Endpoints

#### `GET /api/v1/dashboard/stats`
**Response (`200 OK`):**
```json
{
  "status": "success",
  "data": {
    "totalRevenue": 14500.00,
    "totalStudents": 42,
    "totalCourses": 2,
    "totalEnrollments": 42,
    "monthlyEarnings": 4200.00
  }
}
```

#### `GET /api/v1/dashboard/courses`
**Response (`200 OK`):**
```json
{
  "status": "success",
  "data": [
    {
      "id": "e4a7b9f1-3d2e-4b5a-9f8e-1a2b3c4d5e6f",
      "title": "Reactive Accelerator - Fullstack Go & React 19",
      "slug": "reactive-accelerator",
      "thumbnail": "https://images.unsplash.com/photo-1633356122544-f134324a6cee?w=800",
      "price": 4500.00,
      "discountPrice": 3500.00,
      "studentsCount": 42,
      "rating": 4.9,
      "isPublished": true
    }
  ]
}
```

#### `GET /api/v1/dashboard/lives`
**Response (`200 OK`):**
```json
{
  "status": "success",
  "data": [
    {
      "id": "live-101",
      "title": "React 19 Server Components Live Q&A",
      "scheduledAt": "2026-09-20T18:00:00Z",
      "status": "upcoming",
      "streamUrl": "https://www.youtube.com/watch?v=8sXRyHI3bLw",
      "registeredStudents": 35
    }
  ]
}
```

---

### 2.3 📚 Course Details & Real Programming Video Links

#### `GET /api/v1/courses/reactive-accelerator`
**Response (`200 OK`):**
```json
{
  "status": "success",
  "data": {
    "id": "e4a7b9f1-3d2e-4b5a-9f8e-1a2b3c4d5e6f",
    "title": "Reactive Accelerator - Fullstack Go & React 19",
    "slug": "reactive-accelerator",
    "subtitle": "Master React 19 & Redux Toolkit with Go",
    "description": "Comprehensive fullstack engineering course with real-world programming projects.",
    "price": 4500.00,
    "discountPrice": 3500.00,
    "isPublished": true,
    "modules": [
      {
        "id": "m1a8b9f1-3d2e-4b5a-9f8e-1a2b3c4d5e6f",
        "title": "Module 1: Modern Frontend & Backend Architecture",
        "lessons": [
          {
            "id": "l1a8b9f1-3d2e-4b5a-9f8e-1a2b3c4d5e6f",
            "title": "Lesson 1.1: React 19 Full Course & Server Components",
            "videoUrl": "https://www.youtube.com/watch?v=8sXRyHI3bLw",
            "duration": "25:30",
            "isFree": true
          },
          {
            "id": "l2a8b9f1-3d2e-4b5a-9f8e-1a2b3c4d5e6f",
            "title": "Lesson 1.2: Golang Backend API & Clean Architecture",
            "videoUrl": "https://www.youtube.com/watch?v=YS4e4q9oBaU",
            "duration": "42:15",
            "isFree": false
          }
        ]
      }
    ]
  }
}
```

---

### 2.4 🛡️ Admin Analytics & Platform Revenue

#### `GET /api/v1/analytics/revenue`
**Response (`200 OK`):**
```json
{
  "status": "success",
  "data": {
    "totalGMV": 14500.00,
    "adminNetCut": 725.00,
    "instructorPayout": 13775.00,
    "transactions": [
      {
        "id": "tx-1001",
        "courseTitle": "Reactive Accelerator - Fullstack Go & React 19",
        "studentName": "Karim Rahman",
        "amount": 3500.00,
        "adminCut": 175.00,
        "instructorAmount": 3325.00,
        "paymentMethod": "bKash",
        "date": "2026-09-14T12:00:00Z"
      }
    ]
  }
}
```

---

### 2.5 📁 Media Upload Services

#### `POST /api/v1/upload/image`
**Response (`200 OK`):**
```json
{
  "status": "success",
  "data": {
    "fileId": "img-9f8e-1a2b3c4d5e6f",
    "url": "https://images.unsplash.com/photo-1516321318423-f06f85e504b3?w=800"
  }
}
```

#### `POST /api/v1/upload/video`
**Response (`200 OK`):**
```json
{
  "status": "success",
  "data": {
    "fileId": "vid-7a8b-9c0d1e2f3a4b",
    "url": "https://www.youtube.com/watch?v=8sXRyHI3bLw"
  }
}
```
