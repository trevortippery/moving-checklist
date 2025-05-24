# Moving Checklist API

A RESTful API project designed to help users manage their moving checklist items. Built with Go, PostgreSQL, and Docker, this project is inspired by [melkeydev](https://github.com/melkeydev) from Frontend Masters.

## Features


- RESTful API for managing checklist items
- PostgreSQL database for data storage
- Dockerized for easy development and deployment
- Database schema versioning with `goose` migrations
- Lightweight routing using `chi`
- Scalable backend architecture with clean code principles

## Project Structure
```graphql

moving-checklist/
├── docker-compose.yml # Docker services configuration
├── src/ # Main application code
│ ├── main.go # Entry point of the application
│ ├── api/ # HTTP handler logic (e.g., task and user handlers)
│ ├── db/ # Database connection logic and setup
│ ├── migrations/ # Goose migration files
│ ├── routes/ # Route registration and grouping (using chi)
│ └── utils/ # Shared utility functions (e.g., writing JSON, reading ID param)
└── README.md
```



## Tech Stack

- **Language**: Go (Golang)
- **Database**: PostgreSQL
- **Migrations**: [Goose](https://github.com/pressly/goose)
- **Routing**: [Chi](https://github.com/go-chi/chi)
- **Containerization**: Docker

## Getting Started

### Prerequisites

- [Docker](https://www.docker.com/)
- [Go](https://golang.org/)
- [Goose](https://github.com/pressly/goose)
- [Chi](https://github.com/go-chi/chi)

### 1. Clone the repository

```bash
git clone https://github.com/trevortippery/moving-checklist.git
cd moving-checklist/src
```

### 2. Run the application with Docker

```bash
docker compose up --build
```

Open a new terminal in moving-checklist/src directory:

```bash
go run main.go
```

### Sample Task JSON

```json
{
  "name": "Address Change",
  "description": "Update your address with USPS and banks.",
  "category": "Location",
  "is_complete": false,
  "due_date": "2025-06-01T00:00:00Z"
}
```


### API Endpoints

- POST /tasks - Create a new task
- PUT /tasks/id — Update a task by ID
- DELETE /tasks/id — Delete a task by ID
- GET /tasks/id — Retrieve a task by ID

## Testing

Unit and integration tests are written using [testify](https://github.com/stretchr/testify). Tests are colocated with the source files they cover and focus on handler logic, database interactions, and utility functions.

To run all tests:

```bash
go test ./...
```

## Plans (TODO)
- Add user authentication
- Implement frontend (React)
