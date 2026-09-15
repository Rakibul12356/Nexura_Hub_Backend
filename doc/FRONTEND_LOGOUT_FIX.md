# Frontend Integration Guide - Auth Logout Endpoint (`POST /api/v1/auth/logout`)

**Date:** 15 September 2026  
**Target Team:** Frontend Developers (React 19 / Redux Toolkit / Axios)  
**Backend Environment:** Production (`https://nexura-hub-backend.onrender.com/api/v1`) | Local (`http://localhost:8080/api/v1`)

---

## 📌 Issue Resolved

The `404 Not Found` error when calling `POST /api/v1/auth/logout` has been resolved. The backend now supports the logout endpoint.

---

## 🌐 Endpoint Specification

* **URL:** `/api/v1/auth/logout`
* **HTTP Method:** `POST`
* **Authentication Header:** `Authorization: Bearer <user_access_token>` *(Optional)*
* **Content-Type:** `application/json`
* **Request Body:** None

### 🟢 Success Response (`200 OK`)

```json
{
  "success": true,
  "status": "success",
  "message": "Logged out successfully"
}
```

---

## 💻 Frontend Implementation Example

### 1. Using Axios (or custom API client)

```typescript
import axios from 'axios';

export const logoutUser = async (): Promise<void> => {
  try {
    const token = localStorage.getItem('token');
    await axios.post(
      'http://localhost:8080/api/v1/auth/logout',
      {},
      {
        headers: {
          Authorization: token ? `Bearer ${token}` : '',
        },
      }
    );
  } catch (error) {
    console.warn('Backend logout call returned error, proceeding with client-side cleanup:', error);
  } finally {
    // Clear client-side auth state regardless of outcome
    localStorage.removeItem('token');
    localStorage.removeItem('user');
    window.location.href = '/login';
  }
};
```

### 2. Using Redux Toolkit (AsyncThunk Example)

```typescript
import { createAsyncThunk, createSlice } from '@reduxjs/toolkit';
import api from '../services/api';

export const logout = createAsyncThunk('auth/logout', async (_, { rejectWithValue }) => {
  try {
    const response = await api.post('/auth/logout');
    return response.data;
  } catch (err: any) {
    return rejectWithValue(err.response?.data || 'Logout failed');
  }
});

const authSlice = createSlice({
  name: 'auth',
  initialState: { user: null, token: null, isAuthenticated: false },
  reducers: {
    clearAuth: (state) => {
      state.user = null;
      state.token = null;
      state.isAuthenticated = false;
      localStorage.removeItem('token');
    },
  },
  extraReducers: (builder) => {
    builder
      .addCase(logout.fulfilled, (state) => {
        state.user = null;
        state.token = null;
        state.isAuthenticated = false;
        localStorage.removeItem('token');
      })
      .addCase(logout.rejected, (state) => {
        // Fallback cleanup on error
        state.user = null;
        state.token = null;
        state.isAuthenticated = false;
        localStorage.removeItem('token');
      });
  },
});

export default authSlice.reducer;
```

---

## ✅ Summary Checklist for Frontend Developer

- [ ] Update logout handler to dispatch `POST /api/v1/auth/logout`.
- [ ] Confirm client-side state/localStorage cleanup happens even if network request fails.
- [ ] Test logout button flow in user interface.
