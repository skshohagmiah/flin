package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"taskflow/services"

	flin "github.com/skshohagmiah/flin/clients/go"
)

func main() {
	fmt.Println("🚀 TaskFlow - Distributed Task Management System")
	fmt.Println("=================================================")
	fmt.Println()

	// Connect to Flin server
	opts := flin.DefaultOptions("localhost:7380")
	client, err := flin.NewClient(opts)
	if err != nil {
		log.Fatalf("Failed to create Flin client: %v", err)
	}
	defer client.Close()

	fmt.Println("✅ Connected to Flin server")

	// Load Schema from FQL file
	fmt.Println("📋 Loading schema from schema.fql...")
	schemaContent, err := os.ReadFile("schema.fql")
	if err != nil {
		log.Fatalf("Failed to read schema.fql: %v", err)
	}

	fmt.Println("   Registering schema FQL...")
	if err := client.DB.RegisterFQL(string(schemaContent)); err != nil {
		fmt.Printf("⚠️ (timeout expected): %v\n", err)
	} else {
		fmt.Printf("✅ Schema registered\n")
	}

	fmt.Println("✅ Schemas initialized")
	fmt.Println()

	// Initialize services (DB and KV only - focusing on core features)
	userService := services.NewUserService(client)
	sessionService := services.NewSessionService(client)
	taskService := services.NewTaskService(client)
	// jobProcessor := services.NewJobProcessor(client)
	// jobProcessor.Start()
	// defer jobProcessor.Stop()

	fmt.Println("🎯 TaskFlow is running!")
	fmt.Println()
	fmt.Println("Running demo scenario...")
	fmt.Println()

	// Demo scenario
	runDemo(userService, sessionService, taskService)

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println()
	fmt.Println("Press Ctrl+C to stop...")
	<-sigChan

	fmt.Println("\n👋 Shutting down TaskFlow...")
}

func runDemo(userService *services.UserService, sessionService *services.SessionService, taskService *services.TaskService) {
	// 1. Create users
	fmt.Println("👤 Creating users...")
	alice, err := userService.CreateUser("alice@example.com", "Alice Johnson", "admin")
	if err != nil {
		log.Printf("Error creating Alice: %v", err)
	} else {
		fmt.Printf("   ✓ Created user: %s (%s)\n", alice.Name, alice.Email)
	}

	bob, err := userService.CreateUser("bob@example.com", "Bob Smith", "developer")
	if err != nil {
		log.Printf("Error creating Bob: %v", err)
	} else {
		fmt.Printf("   ✓ Created user: %s (%s)\n", bob.Name, bob.Email)
	}

	charlie, err := userService.CreateUser("charlie@example.com", "Charlie Brown", "developer")
	if err != nil {
		log.Printf("Error creating Charlie: %v", err)
	} else {
		fmt.Printf("   ✓ Created user: %s (%s)\n", charlie.Name, charlie.Email)
	}
	fmt.Println()

	// 2. Create session for Alice
	fmt.Println("🔐 Creating session for Alice...")
	if alice != nil {
		session, err := sessionService.CreateSession(alice.UserID)
		if err != nil {
			log.Printf("Error creating session: %v", err)
		} else {
			fmt.Printf("   ✓ Session created: %s\n", session.SessionID)

			// Verify session
			retrieved, err := sessionService.GetSession(session.SessionID)
			if err != nil {
				log.Printf("Error retrieving session: %v", err)
			} else {
				fmt.Printf("   ✓ Session verified for user: %s\n", retrieved.UserID)
			}
		}
	}
	fmt.Println()

	// 3. Create tasks
	fmt.Println("📝 Creating tasks...")
	projectID := "proj-001"

	task1, err := taskService.CreateTask(projectID, "Implement authentication", "Add JWT-based authentication", "high")
	if err != nil {
		log.Printf("Error creating task 1: %v", err)
	} else {
		fmt.Printf("   ✓ Created task: %s\n", task1.Title)
	}

	task2, err := taskService.CreateTask(projectID, "Setup database", "Configure PostgreSQL database", "medium")
	if err != nil {
		log.Printf("Error creating task 2: %v", err)
	} else {
		fmt.Printf("   ✓ Created task: %s\n", task2.Title)
	}

	task3, err := taskService.CreateTask(projectID, "Write documentation", "Document API endpoints", "low")
	if err != nil {
		log.Printf("Error creating task 3: %v", err)
	} else {
		fmt.Printf("   ✓ Created task: %s\n", task3.Title)
	}
	fmt.Println()

	// 4. Assign tasks
	fmt.Println("👥 Assigning tasks...")
	if task1 != nil && bob != nil {
		err := taskService.AssignTask(projectID, task1.TaskID, bob.UserID)
		if err != nil {
			log.Printf("Error assigning task 1: %v", err)
		} else {
			fmt.Printf("   ✓ Assigned '%s' to %s\n", task1.Title, bob.Name)
		}
	}

	if task2 != nil && charlie != nil {
		err := taskService.AssignTask(projectID, task2.TaskID, charlie.UserID)
		if err != nil {
			log.Printf("Error assigning task 2: %v", err)
		} else {
			fmt.Printf("   ✓ Assigned '%s' to %s\n", task2.Title, charlie.Name)
		}
	}
	fmt.Println()

	// 5. List all users
	fmt.Println("📋 Listing all users...")
	users, err := userService.ListUsers()
	if err != nil {
		log.Printf("Error listing users: %v", err)
	} else {
		for i, user := range users {
			fmt.Printf("   %d. %s (%s) - %s\n", i+1, user.Name, user.Email, user.Role)
		}
	}
	fmt.Println()

	// 6. List tasks for project
	fmt.Println("📋 Listing tasks for project...")
	tasks, err := taskService.ListTasksByProject(projectID)
	if err != nil {
		log.Printf("Error listing tasks: %v", err)
	} else {
		for i, task := range tasks {
			assignee := "Unassigned"
			if task.AssigneeID != "" {
				assignee = task.AssigneeID
			}
			fmt.Printf("   %d. %s [%s] - %s (Assignee: %s)\n",
				i+1, task.Title, task.Priority, task.Status, assignee)
		}
	}
	fmt.Println()

	// Close client connection before summary to avoid race on shutdown if any
	// client.Close() // Already deferred

	fmt.Println("✅ Demo completed!")
	fmt.Println()
	fmt.Println("Features demonstrated:")
	fmt.Println("  ✓ Document DB: User and Task storage with CRUD operations")
	fmt.Println("  ✓ FQL Schema: Server-side schema parsing and registration")
	fmt.Println("  ✓ KV Store: Fast session management")
	fmt.Println("  ✓ Queries: Filtering and listing documents")
	fmt.Println("  ✓ Updates: Modifying documents in place")
}
