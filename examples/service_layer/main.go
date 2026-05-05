// Package main demonstrates the service layer pattern with Firefly framework.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/zhangpeihaoks/firefly/internal/errors"
	"github.com/zhangpeihaoks/firefly/internal/log"
	"github.com/zhangpeihaoks/firefly/internal/middleware"
	httpserver "github.com/zhangpeihaoks/firefly/internal/transport/http"
)

// User represents a user entity in the domain.
type User struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// InMemoryUserRepository is a simple in-memory implementation.
type InMemoryUserRepository struct {
	users  map[int64]*User
	nextID int64
}

func NewInMemoryUserRepository() *InMemoryUserRepository {
	return &InMemoryUserRepository{
		users:  make(map[int64]*User),
		nextID: 1,
	}
}

func (r *InMemoryUserRepository) FindByID(ctx context.Context, id int64) (*User, error) {
	user, ok := r.users[id]
	if !ok {
		return nil, errors.New(errors.CodeNotFound, "USER_NOT_FOUND", "User not found")
	}
	return user, nil
}

func (r *InMemoryUserRepository) Create(ctx context.Context, user *User) (*User, error) {
	user.ID = r.nextID
	user.CreatedAt = time.Now()
	r.users[user.ID] = user
	r.nextID++
	return user, nil
}

// UserService contains business logic for user operations.
type UserService struct {
	repo *InMemoryUserRepository
}

func NewUserService(repo *InMemoryUserRepository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) GetUser(ctx context.Context, id int64) (*User, error) {
	if id <= 0 {
		return nil, errors.New(errors.CodeBadRequest, "INVALID_ID", "User ID must be positive")
	}
	return s.repo.FindByID(ctx, id)
}

func (s *UserService) CreateUser(ctx context.Context, name, email string) (*User, error) {
	if name == "" {
		return nil, errors.New(errors.CodeBadRequest, "INVALID_NAME", "Name is required")
	}
	if email == "" {
		return nil, errors.New(errors.CodeBadRequest, "INVALID_EMAIL", "Email is required")
	}

	user := &User{Name: name, Email: email}
	return s.repo.Create(ctx, user)
}

// UserHandler handles HTTP requests for user operations.
type UserHandler struct {
	service *UserService
}

func NewUserHandler(service *UserService) *UserHandler {
	return &UserHandler{service: service}
}

func (h *UserHandler) GetUser(ctx context.Context, req any) (any, error) {
	userID, ok := httpserver.GetPathParamInt(ctx, "id")
	if !ok {
		return nil, errors.New(errors.CodeBadRequest, "INVALID_ID", "Invalid user ID")
	}

	user, err := h.service.GetUser(ctx, int64(userID))
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (h *UserHandler) CreateUser(ctx context.Context, req any) (any, error) {
	// Get request body from the http request
	r, ok := req.(*http.Request)
	if !ok {
		return nil, errors.New(errors.CodeBadRequest, "INVALID_REQUEST", "Invalid request")
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, errors.New(errors.CodeBadRequest, "INVALID_BODY", "Failed to read request body")
	}
	defer r.Body.Close()

	// Parse JSON body
	var bodyMap map[string]interface{}
	if err := json.Unmarshal(body, &bodyMap); err != nil {
		return nil, errors.New(errors.CodeBadRequest, "INVALID_JSON", "Invalid JSON body")
	}

	name, _ := bodyMap["name"].(string)
	email, _ := bodyMap["email"].(string)

	user, err := h.service.CreateUser(ctx, name, email)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func main() {
	// Initialize logger
	cleanup := log.New(&log.Config{
		FileName:   "service_example.log",
		MaxSize:    50,
		MaxBackups: 3,
		Level:      "debug",
	})
	defer cleanup()

	// Initialize repository and service
	repo := NewInMemoryUserRepository()
	userService := NewUserService(repo)
	userHandler := NewUserHandler(userService)

	// Pre-populate some test data
	userService.CreateUser(context.Background(), "Alice", "alice@example.com")
	userService.CreateUser(context.Background(), "Bob", "bob@example.com")

	// Create HTTP server
	server := httpserver.NewServer(
		httpserver.Address(":8080"),
		httpserver.Middleware(
			middleware.Recovery(),
			middleware.Logging(),
		),
	)

	// Register routes
	server.Route(http.MethodGet, "/users/:id", userHandler.GetUser)
	server.Route(http.MethodPost, "/users", userHandler.CreateUser)

	fmt.Println("Service layer example running on http://localhost:8080")
	fmt.Println("Try these endpoints:")
	fmt.Println("  GET  /users/1            - Get user by ID")
	fmt.Println("  POST /users              - Create user ({\"name\": \"Charlie\", \"email\": \"charlie@example.com\"})")

	// Start server (in real app, this would be in app.Run())
	_ = server.Start(context.Background())

	// Keep running
	select {}
}
