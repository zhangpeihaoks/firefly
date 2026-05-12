package response

import (
	"encoding/json"
	"testing"
)

func TestSuccess(t *testing.T) {
	t.Run("with nil data", func(t *testing.T) {
		resp := Success(nil)
		if resp.Code != 200 {
			t.Errorf("expected code 200, got %d", resp.Code)
		}
		if resp.Message != "success" {
			t.Errorf("expected message 'success', got %q", resp.Message)
		}
		if resp.Data != nil {
			t.Errorf("expected nil data, got %v", resp.Data)
		}
	})

	t.Run("with string data", func(t *testing.T) {
		resp := Success("hello")
		if resp.Data != "hello" {
			t.Errorf("expected data 'hello', got %v", resp.Data)
		}
	})

	t.Run("with struct data", func(t *testing.T) {
		type User struct {
			Name string `json:"name"`
			Age  int    `json:"age"`
		}
		user := User{Name: "Alice", Age: 30}
		resp := Success(user)
		data, ok := resp.Data.(User)
		if !ok {
			t.Fatalf("expected User type, got %T", resp.Data)
		}
		if data.Name != "Alice" || data.Age != 30 {
			t.Errorf("expected User{Alice, 30}, got %+v", data)
		}
	})

	t.Run("JSON serialization", func(t *testing.T) {
		resp := Success(map[string]int{"count": 42})
		data, err := json.Marshal(resp)
		if err != nil {
			t.Fatalf("failed to marshal response: %v", err)
		}
		var decoded Response
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if decoded.Code != 200 {
			t.Errorf("expected code 200, got %d", decoded.Code)
		}
		if decoded.Message != "success" {
			t.Errorf("expected message 'success', got %q", decoded.Message)
		}
	})
}

func TestSuccessWithMessage(t *testing.T) {
	t.Run("with custom message", func(t *testing.T) {
		resp := SuccessWithMessage("操作成功", "data")
		if resp.Code != 200 {
			t.Errorf("expected code 200, got %d", resp.Code)
		}
		if resp.Message != "操作成功" {
			t.Errorf("expected message '操作成功', got %q", resp.Message)
		}
		if resp.Data != "data" {
			t.Errorf("expected data 'data', got %v", resp.Data)
		}
	})

	t.Run("with nil data", func(t *testing.T) {
		resp := SuccessWithMessage("created", nil)
		if resp.Data != nil {
			t.Errorf("expected nil data, got %v", resp.Data)
		}
	})
}

func TestSuccessWithPage(t *testing.T) {
	t.Run("basic pagination", func(t *testing.T) {
		items := []string{"a", "b", "c"}
		resp := SuccessWithPage(items, 1, 10, 3)
		if resp.Code != 200 {
			t.Errorf("expected code 200, got %d", resp.Code)
		}
		if resp.Message != "success" {
			t.Errorf("expected message 'success', got %q", resp.Message)
		}
		if resp.Page == nil {
			t.Fatal("expected page info, got nil")
		}
		if resp.Page.Page != 1 {
			t.Errorf("expected page 1, got %d", resp.Page.Page)
		}
		if resp.Page.PageSize != 10 {
			t.Errorf("expected page size 10, got %d", resp.Page.PageSize)
		}
		if resp.Page.Total != 3 {
			t.Errorf("expected total 3, got %d", resp.Page.Total)
		}
		if resp.Page.TotalPage != 1 {
			t.Errorf("expected total page 1, got %d", resp.Page.TotalPage)
		}
	})

	t.Run("multiple pages", func(t *testing.T) {
		items := make([]int, 25)
		resp := SuccessWithPage(items, 1, 10, 25)
		if resp.Page.TotalPage != 3 {
			t.Errorf("expected total page 3, got %d", resp.Page.TotalPage)
		}
	})

	t.Run("exact division", func(t *testing.T) {
		resp := SuccessWithPage(nil, 2, 5, 10)
		if resp.Page.TotalPage != 2 {
			t.Errorf("expected total page 2, got %d", resp.Page.TotalPage)
		}
	})

	t.Run("page info JSON", func(t *testing.T) {
		resp := SuccessWithPage([]int{1, 2}, 1, 2, 2)
		data, err := json.Marshal(resp)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}
		var decoded PageResponse
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if decoded.Page.Page != 1 || decoded.Page.PageSize != 2 {
			t.Errorf("page info mismatch: %+v", decoded.Page)
		}
	})
}

func TestError(t *testing.T) {
	t.Run("basic error", func(t *testing.T) {
		resp := Error(400, "bad request")
		if resp.Code != 400 {
			t.Errorf("expected code 400, got %d", resp.Code)
		}
		if resp.Message != "bad request" {
			t.Errorf("expected message 'bad request', got %q", resp.Message)
		}
		if resp.Data != nil {
			t.Errorf("expected nil data, got %v", resp.Data)
		}
	})

	t.Run("different status codes", func(t *testing.T) {
		tests := []struct {
			code    int
			message string
		}{
			{400, "Bad Request"},
			{401, "Unauthorized"},
			{403, "Forbidden"},
			{404, "Not Found"},
			{500, "Internal Server Error"},
			{503, "Service Unavailable"},
		}
		for _, tt := range tests {
			resp := Error(tt.code, tt.message)
			if resp.Code != tt.code {
				t.Errorf("expected code %d, got %d", tt.code, resp.Code)
			}
		}
	})

	t.Run("JSON serialization", func(t *testing.T) {
		resp := Error(500, "internal error")
		data, err := json.Marshal(resp)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}
		var decoded Response
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if decoded.Code != 500 {
			t.Errorf("expected code 500, got %d", decoded.Code)
		}
		if decoded.Message != "internal error" {
			t.Errorf("expected 'internal error', got %q", decoded.Message)
		}
	})
}

func TestErrorWithData(t *testing.T) {
	t.Run("with data", func(t *testing.T) {
		resp := ErrorWithData(422, "validation failed", map[string]string{
			"field": "name",
			"error": "required",
		})
		if resp.Code != 422 {
			t.Errorf("expected code 422, got %d", resp.Code)
		}
		if resp.Message != "validation failed" {
			t.Errorf("expected 'validation failed', got %q", resp.Message)
		}
		data, ok := resp.Data.(map[string]string)
		if !ok {
			t.Fatalf("expected map[string]string, got %T", resp.Data)
		}
		if data["field"] != "name" {
			t.Errorf("expected field 'name', got %q", data["field"])
		}
	})

	t.Run("nil data", func(t *testing.T) {
		resp := ErrorWithData(400, "error", nil)
		if resp.Data != nil {
			t.Errorf("expected nil data, got %v", resp.Data)
		}
	})
}

func TestResponseImmutability(t *testing.T) {
	// Verify that Response, PageResponse, and PageInfo are value types
	// and can be safely used in multiple contexts
	r1 := Success("data")
	r2 := *r1
	r2.Data = "modified"
	if r1.Data != "data" {
		t.Error("Response should be a value type (copied by dereference)")
	}
}
