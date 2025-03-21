package integration

import (
	"strconv"
	"testing"

	"go-auth-boilerplate/internal/models"
	"go-auth-boilerplate/internal/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var validPost = models.Post{
	Title: "Test Post",
	Body:  "This is a test post body that is long enough to pass validation.",
}

func createTestUser(t *testing.T, ts *testutil.TestServer) string {
	createUserReq := map[string]any{
		"first_name": "John",
		"last_name":  "Doe",
		"age":        30,
		"email":      "john@example.com",
		"password":   "Pass123",
	}

	resp := ts.SendRequest(t, "POST", "/api/v1/user/signup", createUserReq, nil)
	require.Equal(t, 201, resp.StatusCode)

	var result map[string]any
	err := resp.DecodeBody(&result)
	require.NoError(t, err)

	token := result["token"].(string)
	require.NotEmpty(t, token)
	return token
}

func createTestPost(t *testing.T, ts *testutil.TestServer, token string) uint {
	headers := map[string]string{
		"Authorization": "Bearer " + token,
	}

	resp := ts.SendRequest(t, "POST", "/api/v1/posts/create", validPost, headers)
	require.Equal(t, 201, resp.StatusCode)

	var result map[string]any
	err := resp.DecodeBody(&result)
	require.NoError(t, err)

	postID := uint(result["id"].(float64))
	require.NotZero(t, postID)
	return postID
}

func getAuthHeaders(token string) map[string]string {
	return map[string]string{
		"Authorization": "Bearer " + token,
	}
}

func TestPostCreation(t *testing.T) {
	ts := testutil.NewTestServer(t)
	t.Cleanup(func() { ts.Close(t) })

	err := ts.DB.AutoMigrate(&models.User{}, &models.Post{})
	require.NoError(t, err)

	token := createTestUser(t, ts)

	tests := []struct {
		name       string
		post       models.Post
		wantStatus int
	}{
		{
			name:       "valid post",
			post:       validPost,
			wantStatus: 201,
		},
		{
			name: "empty title",
			post: models.Post{
				Title: "",
				Body:  validPost.Body,
			},
			wantStatus: 400,
		},
		{
			name: "empty body",
			post: models.Post{
				Title: validPost.Title,
				Body:  "",
			},
			wantStatus: 400,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers := getAuthHeaders(token)
			resp := ts.SendRequest(t, "POST", "/api/v1/posts/create", tt.post, headers)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

func TestPostRetrieval(t *testing.T) {
	ts := testutil.NewTestServer(t)
	t.Cleanup(func() { ts.Close(t) })

	err := ts.DB.AutoMigrate(&models.User{}, &models.Post{})
	require.NoError(t, err)

	token := createTestUser(t, ts)
	postID := createTestPost(t, ts, token)

	tests := []struct {
		name       string
		path       string
		wantStatus int
	}{
		{
			name:       "get all posts",
			path:       "/api/v1/posts",
			wantStatus: 200,
		},
		{
			name:       "get single post",
			path:       "/api/v1/posts/" + strconv.Itoa(int(postID)),
			wantStatus: 200,
		},
		{
			name:       "get non-existent post",
			path:       "/api/v1/posts/999999",
			wantStatus: 404,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers := getAuthHeaders(token)
			resp := ts.SendRequest(t, "GET", tt.path, nil, headers)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

func TestPostUpdate(t *testing.T) {
	ts := testutil.NewTestServer(t)
	t.Cleanup(func() { ts.Close(t) })

	err := ts.DB.AutoMigrate(&models.User{}, &models.Post{})
	require.NoError(t, err)

	token := createTestUser(t, ts)
	postID := createTestPost(t, ts, token)

	tests := []struct {
		name       string
		post       models.Post
		wantStatus int
	}{
		{
			name: "valid update",
			post: models.Post{
				Title: "Updated Title",
				Body:  "Updated body content",
			},
			wantStatus: 200,
		},
		{
			name: "empty title",
			post: models.Post{
				Title: "",
				Body:  validPost.Body,
			},
			wantStatus: 400,
		},
		{
			name: "empty body",
			post: models.Post{
				Title: validPost.Title,
				Body:  "",
			},
			wantStatus: 400,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers := getAuthHeaders(token)
			path := "/api/v1/posts/" + strconv.Itoa(int(postID)) + "/update"
			resp := ts.SendRequest(t, "PATCH", path, tt.post, headers)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

func TestPostDeletion(t *testing.T) {
	ts := testutil.NewTestServer(t)
	t.Cleanup(func() { ts.Close(t) })

	err := ts.DB.AutoMigrate(&models.User{}, &models.Post{})
	require.NoError(t, err)

	token := createTestUser(t, ts)
	postID := createTestPost(t, ts, token)

	tests := []struct {
		name       string
		postID     uint
		wantStatus int
	}{
		{
			name:       "delete existing post",
			postID:     postID,
			wantStatus: 200,
		},
		{
			name:       "delete non-existent post",
			postID:     999999,
			wantStatus: 404,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers := getAuthHeaders(token)
			path := "/api/v1/posts/" + strconv.Itoa(int(tt.postID)) + "/delete"
			resp := ts.SendRequest(t, "DELETE", path, nil, headers)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)

			if tt.wantStatus == 200 {
				var post models.Post
				err := ts.DB.First(&post, tt.postID).Error
				assert.Error(t, err)
			}
		})
	}
}
