// Example demonstrating dynamic route parameter parsing in Firefly HTTP server
package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/zhangpeihaoks/firefly/internal/log"
	"github.com/zhangpeihaoks/firefly/internal/middleware"
	httpserver "github.com/zhangpeihaoks/firefly/internal/transport/http"
)

func main() {
	// Initialize logger
	cleanup := log.New(&log.Config{
		FileName:   "dynamic_routing.log",
		MaxSize:    100,
		MaxBackups: 3,
		MaxAge:     7,
		Level:      "info",
		JSONFormat: true,
		Location:   true,
	})
	defer cleanup()

	// Create HTTP server
	srv := httpserver.NewServer(
		httpserver.Address(":8080"),
		httpserver.Timeout(30*time.Second),
		httpserver.Middleware(
			middleware.Recovery(),
			middleware.Logging(),
		),
	)

	// Register routes with dynamic parameters
	registerRoutes(srv)

	fmt.Println("Starting HTTP server on :8080...")
	fmt.Println("Try these endpoints:")
	fmt.Println("  GET http://localhost:8080/users/123")
	fmt.Println("  GET http://localhost:8080/products/laptop-2025")
	fmt.Println("  GET http://localhost:8080/search?q=golang&page=2&limit=20")
	fmt.Println("  GET http://localhost:8080/orgs/456/users/789")
	fmt.Println()

	// Start server
	ctx := context.Background()
	if err := srv.Start(ctx); err != nil {
		log.Error("Failed to start server", "error", err)
		return
	}

	// Keep server running for demonstration
	fmt.Println("Server started. Press Ctrl+C to stop.")
	time.Sleep(60 * time.Second)

	// Stop server
	fmt.Println("Stopping server...")
	if err := srv.Stop(ctx); err != nil {
		log.Error("Failed to stop server", "error", err)
		return
	}

	fmt.Println("Server stopped.")
}

func registerRoutes(srv *httpserver.Server) {
	// Example 1: Simple path parameter
	srv.Route(http.MethodGet, "/users/:id", getUserHandler)

	// Example 2: Path parameter with custom pattern
	srv.Route(http.MethodGet, "/products/:slug", getProductHandler)

	// Example 3: Query parameters
	srv.Route(http.MethodGet, "/search", searchHandler)

	// Example 4: Multiple path parameters
	srv.Route(http.MethodGet, "/orgs/:org_id/users/:user_id", getOrgUserHandler)

	// Example 5: Route group with dynamic parameters
	api := srv.Group("/api/v1")
	{
		api.Route(http.MethodGet, "/posts/:post_id", getPostHandler)
		api.Route(http.MethodGet, "/posts/:post_id/comments/:comment_id", getCommentHandler)
	}
}

// Handler functions demonstrating parameter extraction

func getUserHandler(ctx context.Context, req any) (any, error) {
	// Extract path parameter
	userID, ok := httpserver.GetPathParamInt(ctx, "id")
	if !ok {
		return map[string]interface{}{
			"error": "Invalid user ID",
		}, nil
	}

	return map[string]interface{}{
		"user_id": userID,
		"name":    fmt.Sprintf("User %d", userID),
		"email":   fmt.Sprintf("user%d@example.com", userID),
		"active":  true,
	}, nil
}

func getProductHandler(ctx context.Context, req any) (any, error) {
	// Extract string path parameter
	slug, ok := httpserver.GetPathParam(ctx, "slug")
	if !ok {
		return map[string]interface{}{
			"error": "Product slug is required",
		}, nil
	}

	return map[string]interface{}{
		"slug":        slug,
		"name":        fmt.Sprintf("Product: %s", slug),
		"price":       99.99,
		"in_stock":    true,
		"description": "A great product with slug: " + slug,
	}, nil
}

func searchHandler(ctx context.Context, req any) (any, error) {
	// Extract query parameters
	query, _ := httpserver.GetQueryParam(ctx, "q")
	page, _ := httpserver.GetQueryParamInt(ctx, "page")
	limit, _ := httpserver.GetQueryParamInt(ctx, "limit")

	// Default values
	if page == 0 {
		page = 1
	}
	if limit == 0 {
		limit = 10
	}

	// Simulate search results
	results := []map[string]interface{}{
		{"id": 1, "title": fmt.Sprintf("Result 1 for '%s'", query), "score": 0.95},
		{"id": 2, "title": fmt.Sprintf("Result 2 for '%s'", query), "score": 0.87},
		{"id": 3, "title": fmt.Sprintf("Result 3 for '%s'", query), "score": 0.76},
	}

	return map[string]interface{}{
		"query":   query,
		"page":    page,
		"limit":   limit,
		"total":   100,
		"results": results,
	}, nil
}

func getOrgUserHandler(ctx context.Context, req any) (any, error) {
	// Extract multiple path parameters
	orgID, _ := httpserver.GetPathParamInt(ctx, "org_id")
	userID, _ := httpserver.GetPathParamInt(ctx, "user_id")

	return map[string]interface{}{
		"org_id":      orgID,
		"user_id":     userID,
		"org_name":    fmt.Sprintf("Organization %d", orgID),
		"user_name":   fmt.Sprintf("User %d in Org %d", userID, orgID),
		"permissions": []string{"read", "write"},
		"last_active": time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
	}, nil
}

func getPostHandler(ctx context.Context, req any) (any, error) {
	// Using MustGetPathParam for required parameters
	postID := httpserver.MustGetPathParamInt(ctx, "post_id")

	return map[string]interface{}{
		"post_id":  postID,
		"title":    fmt.Sprintf("Post %d Title", postID),
		"content":  fmt.Sprintf("This is the content of post %d", postID),
		"author":   "John Doe",
		"created":  time.Now().Add(-time.Duration(postID) * time.Hour).Format(time.RFC3339),
		"likes":    postID * 10,
		"comments": postID * 3,
	}, nil
}

func getCommentHandler(ctx context.Context, req any) (any, error) {
	// Extract parameters from nested route
	postID := httpserver.MustGetPathParamInt(ctx, "post_id")
	commentID := httpserver.MustGetPathParamInt(ctx, "comment_id")

	return map[string]interface{}{
		"post_id":    postID,
		"comment_id": commentID,
		"text":       fmt.Sprintf("Comment %d on post %d", commentID, postID),
		"author":     "Jane Smith",
		"created":    time.Now().Add(-time.Duration(commentID) * time.Minute).Format(time.RFC3339),
		"upvotes":    commentID * 5,
	}, nil
}
