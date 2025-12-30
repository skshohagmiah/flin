# TaskFlow - Distributed Task Management System

A production-ready example application demonstrating all Flin features.

## Features Demonstrated

- **🗄️ Database**: User profiles and tasks (Cassandra-like)
- **🔑 KV Store**: Fast session management
- **📬 Queue**: Background job processing for notifications
- **🌊 Stream**: Real-time activity feed with pub/sub

## Architecture

```
TaskFlow
TaskFlow
├── schema.fql (FQL Schema Definition)
├── Services
│   ├── UserService (Document DB)
│   ├── SessionService (KV Store)
│   ├── TaskService (Document DB + Queue + Stream)
│   ├── ActivityService (Stream)
│   └── JobProcessor (Queue)
└── Main Application
```

## Quick Start

### 1. Setup Flin Server

```bash
cd /Users/shohag/Desktop/Flin/examples/taskflow
chmod +x setup.sh
./setup.sh
```

### 2. Run TaskFlow

```bash
go run main.go
```

## What the Demo Does

1. **Creates Users**: Alice (admin), Bob (developer), Charlie (developer)
2. **Session Management**: Creates and validates a session for Alice
3. **Creates Tasks**: Three tasks with different priorities
4. **Assigns Tasks**: Assigns tasks to Bob and Charlie
   - Triggers background notification jobs (Queue)
   - Publishes activity events (Stream)
5. **Lists Data**: Shows all users and tasks

## Features in Action

### Wide-Column Database (FQL)
- **FQL (Flin Query Language)**: Schemas defined in `schema.fql` (CQL-compatible)
- Server-side parsing and registration
- Partition keys ensure efficient data distribution
- Clustering keys enable sorted queries

### KV Store
- Sessions stored with O(1) lookup time
- Automatic expiration handling
- Perfect for caching and temporary data

### Queue
- Background jobs processed asynchronously
- Notification jobs enqueued when tasks are assigned
- Worker pool processes jobs concurrently

### Stream (Pub/Sub)
- Activity events published in real-time
- Consumer groups for scalable processing
- Offset tracking for reliable delivery

## Code Structure

```
taskflow/
├── main.go              # Application entry point
├── schemas/             # Schema definitions
│   ├── users.go
│   ├── projects.go
│   └── tasks.go
├── services/            # Business logic
│   ├── user_service.go
│   ├── session_service.go
│   ├── task_service.go
│   ├── activity_service.go
│   └── job_processor.go
└── setup.sh             # Setup script
```

## Stopping

```bash
pkill -f flin-server
```

## Notes

- All Flin features are demonstrated in a realistic production scenario
- Background jobs and activity stream run concurrently
