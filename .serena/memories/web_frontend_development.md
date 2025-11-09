# TelHawk Stack - Web Frontend Development

## Frontend Overview

The web UI is a React-based single-page application that provides:
- Search console for querying events
- Event table with OCSF field inspection
- Severity color-coding
- Time-based filtering
- Real-time search

## Technology Stack

### Frontend
- **Framework**: React (functional components with hooks)
- **Build Tool**: Create React App or Vite
- **Language**: JavaScript/TypeScript
- **Styling**: CSS modules or styled-components
- **HTTP Client**: fetch API or axios

### Backend
- **Language**: Go
- **Purpose**: Serves static frontend files + proxies API requests
- **Port**: 3000
- **Location**: `web/backend/`

## Directory Structure

```
web/
├── backend/              # Go backend server
│   ├── cmd/
│   │   └── web/
│   │       └── main.go   # Server entry point
│   ├── internal/
│   │   ├── config/
│   │   ├── handlers/
│   │   └── server/
│   ├── static/           # Built frontend files (generated)
│   ├── config.yaml
│   ├── Dockerfile
│   └── go.mod
│
└── frontend/             # React application
    ├── public/           # Public assets
    ├── src/              # React source code
    │   ├── components/   # React components
    │   ├── pages/        # Page components
    │   ├── services/     # API services
    │   ├── utils/        # Utility functions
    │   ├── App.js        # Root component
    │   └── index.js      # Entry point
    ├── package.json
    ├── package-lock.json
    └── .env.example
```

## Development Setup

### Prerequisites
```bash
# Node.js and npm (check versions)
node --version  # Should be 16+
npm --version   # Should be 8+
```

### Install Dependencies
```bash
cd web/frontend
npm install
```

### Development Server
```bash
cd web/frontend

# Start development server (hot reload)
npm start

# Opens browser at http://localhost:3000 (or different port)
```

### Development with Backend
```bash
# Terminal 1: Start backend services
docker-compose up -d auth ingest core storage query opensearch auth-db redis

# Terminal 2: Start Go web backend (for API proxy)
cd web/backend
go run ./cmd/web

# Terminal 3: Start React frontend
cd web/frontend
npm start
```

The React dev server will proxy API requests to the Go backend.

## Building for Production

### Build Frontend
```bash
cd web/frontend

# Create optimized production build
npm run build

# Output: build/ directory with static files
```

### Copy Build to Backend
```bash
# Build files need to be in web/backend/static/
cp -r web/frontend/build/* web/backend/static/
```

### Build Docker Image
```bash
# Dockerfile handles the build process
docker-compose build web
```

The Dockerfile typically:
1. Builds React app in frontend/
2. Copies build output to backend/static/
3. Builds Go backend
4. Serves static files from Go server

## Frontend Development Commands

### Development
```bash
# Start dev server
npm start

# Run with specific port
PORT=3001 npm start
```

### Building
```bash
# Production build
npm run build

# Build with custom config
REACT_APP_API_URL=https://api.example.com npm run build
```

### Testing
```bash
# Run tests
npm test

# Run tests with coverage
npm test -- --coverage

# Run tests in CI mode
CI=true npm test
```

### Linting and Formatting
```bash
# Run ESLint
npm run lint

# Fix linting issues
npm run lint -- --fix

# Format with Prettier (if configured)
npm run format
```

### Dependencies
```bash
# Install new dependency
npm install <package-name>

# Install dev dependency
npm install --save-dev <package-name>

# Update dependencies
npm update

# Audit for vulnerabilities
npm audit

# Fix vulnerabilities
npm audit fix
```

## Frontend Configuration

### Environment Variables

Create `.env` file in `web/frontend/`:

```bash
# API endpoint (development)
REACT_APP_API_URL=http://localhost:8082

# Other config
REACT_APP_VERSION=1.0.0
```

### Environment Files
- `.env`: Default values
- `.env.local`: Local overrides (gitignored)
- `.env.development`: Development-specific
- `.env.production`: Production-specific

### Accessing in Code
```javascript
// Environment variables must start with REACT_APP_
const apiUrl = process.env.REACT_APP_API_URL;
```

## API Integration

### API Service Pattern
Create service files for API calls:

```javascript
// src/services/api.js
const API_BASE_URL = process.env.REACT_APP_API_URL || 'http://localhost:8082';

export const searchEvents = async (query, timeRange) => {
  const response = await fetch(`${API_BASE_URL}/api/v1/search`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${getToken()}`,
    },
    body: JSON.stringify({ query, time_range: timeRange }),
  });
  
  if (!response.ok) {
    throw new Error(`Search failed: ${response.statusText}`);
  }
  
  return response.json();
};

export const getEvents = async (filters) => {
  const response = await fetch(`${API_BASE_URL}/api/v1/events`, {
    method: 'GET',
    headers: {
      'Authorization': `Bearer ${getToken()}`,
    },
  });
  
  return response.json();
};
```

### Authentication
```javascript
// src/services/auth.js
export const login = async (username, password) => {
  const response = await fetch('http://localhost:8080/api/v1/auth/login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username, password }),
  });
  
  const data = await response.json();
  
  // Store token
  localStorage.setItem('token', data.access_token);
  localStorage.setItem('refresh_token', data.refresh_token);
  
  return data;
};

export const getToken = () => {
  return localStorage.getItem('token');
};

export const logout = () => {
  localStorage.removeItem('token');
  localStorage.removeItem('refresh_token');
};
```

## Common React Patterns

### Component Structure
```jsx
// src/components/EventTable.js
import React, { useState, useEffect } from 'react';
import { getEvents } from '../services/api';
import './EventTable.css';

const EventTable = ({ filters }) => {
  const [events, setEvents] = useState([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  
  useEffect(() => {
    const fetchEvents = async () => {
      setLoading(true);
      try {
        const data = await getEvents(filters);
        setEvents(data.events);
      } catch (err) {
        setError(err.message);
      } finally {
        setLoading(false);
      }
    };
    
    fetchEvents();
  }, [filters]);
  
  if (loading) return <div>Loading...</div>;
  if (error) return <div>Error: {error}</div>;
  
  return (
    <table className="event-table">
      <thead>
        <tr>
          <th>Timestamp</th>
          <th>Severity</th>
          <th>Message</th>
        </tr>
      </thead>
      <tbody>
        {events.map(event => (
          <tr key={event.id}>
            <td>{new Date(event.time).toLocaleString()}</td>
            <td>{event.severity}</td>
            <td>{event.message}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
};

export default EventTable;
```

## Styling

### CSS Modules (if configured)
```javascript
import styles from './Component.module.css';

<div className={styles.container}>
  <h1 className={styles.title}>Title</h1>
</div>
```

### Inline Styles
```javascript
const styles = {
  container: {
    padding: '20px',
    backgroundColor: '#f0f0f0',
  },
};

<div style={styles.container}>Content</div>
```

## Deployment

### Production Build Process
```bash
# 1. Build frontend
cd web/frontend
npm run build

# 2. Build Docker image (includes frontend build)
cd ../..
docker-compose build web

# 3. Deploy
docker-compose up -d web
```

### Backend Serving Static Files

The Go backend serves the built React app:

```go
// web/backend/internal/server/server.go
http.Handle("/", http.FileServer(http.Dir("./static")))
```

All routes fallback to `index.html` for React Router support.

## Common Development Tasks

### Add New Component
```bash
# Create component file
cd web/frontend/src/components
touch MyComponent.js MyComponent.css

# Add tests
touch MyComponent.test.js
```

### Add New API Endpoint
```javascript
// 1. Add to service
// src/services/api.js
export const newApiCall = async (params) => {
  const response = await fetch(`${API_BASE_URL}/api/v1/new-endpoint`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(params),
  });
  return response.json();
};

// 2. Use in component
import { newApiCall } from '../services/api';
```

### Add New Page/Route
```javascript
// 1. Create page component
// src/pages/NewPage.js
const NewPage = () => {
  return <div>New Page</div>;
};
export default NewPage;

// 2. Add route (if using React Router)
// src/App.js
import { BrowserRouter, Routes, Route } from 'react-router-dom';
import NewPage from './pages/NewPage';

<Routes>
  <Route path="/new-page" element={<NewPage />} />
</Routes>
```

## Troubleshooting

### CORS Issues
If API calls fail with CORS errors:

1. Configure backend to allow CORS:
```go
// web/backend/internal/server/server.go
w.Header().Set("Access-Control-Allow-Origin", "*")
w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
```

2. Or use proxy in `package.json`:
```json
{
  "proxy": "http://localhost:8082"
}
```

### Build Errors
```bash
# Clear cache and rebuild
rm -rf node_modules package-lock.json
npm install
npm run build
```

### Port Already in Use
```bash
# Change port
PORT=3001 npm start

# Or kill process using port 3000
lsof -ti:3000 | xargs kill
```

## Testing

### Unit Tests (Jest)
```javascript
// MyComponent.test.js
import { render, screen } from '@testing-library/react';
import MyComponent from './MyComponent';

test('renders component', () => {
  render(<MyComponent />);
  const element = screen.getByText(/expected text/i);
  expect(element).toBeInTheDocument();
});
```

### Integration Tests
```javascript
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import LoginPage from './LoginPage';

test('logs in user', async () => {
  render(<LoginPage />);
  
  userEvent.type(screen.getByLabelText(/username/i), 'admin');
  userEvent.type(screen.getByLabelText(/password/i), 'admin123');
  userEvent.click(screen.getByRole('button', { name: /login/i }));
  
  await waitFor(() => {
    expect(screen.getByText(/welcome/i)).toBeInTheDocument();
  });
});
```

## Performance Optimization

### Code Splitting
```javascript
// Lazy load components
import React, { lazy, Suspense } from 'react';

const HeavyComponent = lazy(() => import('./HeavyComponent'));

<Suspense fallback={<div>Loading...</div>}>
  <HeavyComponent />
</Suspense>
```

### Memoization
```javascript
import React, { useMemo, useCallback } from 'react';

// Memoize expensive calculations
const expensiveValue = useMemo(() => computeExpensiveValue(a, b), [a, b]);

// Memoize callbacks
const handleClick = useCallback(() => {
  doSomething(a);
}, [a]);
```

## Quick Reference

| Task | Command |
|------|---------|
| Install dependencies | `npm install` |
| Start dev server | `npm start` |
| Build for production | `npm run build` |
| Run tests | `npm test` |
| Lint code | `npm run lint` |
| Update dependencies | `npm update` |
| Audit security | `npm audit` |
| Clear cache | `rm -rf node_modules && npm install` |