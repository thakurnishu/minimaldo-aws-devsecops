# 📝 Todo App - Microservices Architecture

A modern, full-stack todo application built with Go backend, vanilla JavaScript frontend, and PostgreSQL database. Designed as microservices that can be deployed independently.

## 🏗️ Architecture

```
Frontend (React)  ←→  Backend (Go)  ←→  Database (PostgreSQL)
     Port 3000           Port 8080           Port 5432
```

## 🔄 CI/CD Pipeline

The project includes GitHub Actions workflows in the `.github/workflows` directory for automated testing and deployment. The workflows are triggered on push and pull requests to the main branch.

### CI Workflow
![CI Workflow](.github/images/Minimal-CI-Workflow.excalidraw.png)

### CD Workflow With GitOps
![CD Workflow](.github/images/Minimal-CD-Workflow.excalidraw.png)

## 🛠️ Quick Start

### Using Makefile (Recommended)

The project includes a comprehensive Makefile to simplify development and deployment tasks:

```bash
# Build all Docker images
make build

# Start all services in detached mode
make up

# Stop all services
make down

# View logs from all services
make logs

# Clean up containers, volumes, and images
make clean
```

### Access the Application
- Frontend: http://localhost:3000
- Backend API: http://localhost:8080
- Database: localhost:5432

## 📡 API Endpoints

### Todo Operations

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET`    | `/api/todos` | Get all todos (sorted by creation date, newest first) |
| `POST`   | `/api/todos` | Create a new todo |
| `PUT`    | `/api/todos/:id` | Update an existing todo |
| `DELETE` | `/api/todos/:id` | Delete a todo |
| `GET`    | `/api/todos/by-date` | Get todos filtered by date range |
| `GET`    | `/api/health` | Health check endpoint |

### Example API Usage

**Create Todo:**
```bash
curl -X POST http://localhost:8080/api/todos \
  -H "Content-Type: application/json" \
  -d '{"title":"Learn Go","description":"Build a todo app","completed":false}'
```

**Get All Todos:**
```bash
curl http://localhost:8080/api/todos
```

**Update Todo:**
```bash
curl -X PUT http://localhost:8080/api/todos/1 \
  -H "Content-Type: application/json" \
  -d '{"title":"Updated Title","description":"Updated description","completed":true}'
```

**Delete Todo:**
```bash
curl -X DELETE http://localhost:8080/api/todos/1
```

**Get Todos by Date Range:**
```bash
# Get todos for a specific day
curl "http://localhost:8080/api/todos/by-date?range=day&date=2023-01-01"

# Get todos for a specific week (starting from Sunday)
curl "http://localhost:8080/api/todos/by-date?range=week&date=2023-01-01"

# Get todos for a specific month
curl "http://localhost:8080/api/todos/by-date?range=month&date=2023-01-01"
```

**Health Check:**
```bash
curl http://localhost:8080/api/health
```

### Response Formats

**Todo Object:**
```json
{
  "id": 1,
  "title": "Example Todo",
  "description": "This is an example todo",
  "completed": false,
  "created_at": "2023-01-01T00:00:00Z",
  "updated_at": "2023-01-01T00:00:00Z"
}
```

**Grouped Todos Response (for date ranges):**
```json
[
  {
    "date": "2023-01-01",
    "todos": [
      {
        "id": 1,
        "title": "Morning Task",
        "description": "Complete morning routine",
        "completed": true,
        "created_at": "2023-01-01T09:00:00Z",
        "updated_at": "2023-01-01T09:00:00Z"
      },
      {
        "id": 2,
        "title": "Afternoon Task",
        "description": "Work on project",
        "completed": false,
        "created_at": "2023-01-01T14:00:00Z",
        "updated_at": "2023-01-01T14:00:00Z"
      }
    ]
  }
]
```

