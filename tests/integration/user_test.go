package integration

import (
	"testing"

	"go-auth-boilerplate/internal/models"
	"go-auth-boilerplate/internal/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserFlow(t *testing.T) {
	ts := testutil.NewTestServer(t)
	defer ts.Close(t)

	err := ts.DB.AutoMigrate(&models.User{})
	require.NoError(t, err)

	var token string
	var userID uint

	t.Run("create user", func(t *testing.T) {
		createUserReq := map[string]interface{}{
			"first_name": "John",
			"last_name":  "Doe",
			"age":        30,
			"email":      "john@example.com",
			"password":   "Pass123",
		}

		resp := ts.SendRequest(t, "POST", "/api/v1/user/signup", createUserReq, nil)
		assert.Equal(t, 201, resp.StatusCode)

		var result map[string]interface{}
		err := resp.DecodeBody(&result)
		require.NoError(t, err)

		assert.Contains(t, result, "token")
		token = result["token"].(string)
		assert.NotEmpty(t, token)

		var user models.User
		err = ts.DB.First(&user, "email = ?", createUserReq["email"]).Error
		require.NoError(t, err)
		assert.Equal(t, createUserReq["email"], user.Email)
		userID = user.ID
	})

	t.Run("create user with duplicate email", func(t *testing.T) {
		createUserReq := map[string]interface{}{
			"first_name": "John",
			"last_name":  "Doe",
			"age":        30,
			"email":      "john@example.com",
			"password":   "Pass123",
		}

		resp := ts.SendRequest(t, "POST", "/api/v1/user/signup", createUserReq, nil)
		assert.Equal(t, 400, resp.StatusCode)

		var result map[string]interface{}
		err := resp.DecodeBody(&result)
		require.NoError(t, err)
		assert.Contains(t, result["error"], "Email already registered")
	})

	t.Run("create user with invalid data", func(t *testing.T) {
		createUserReq := map[string]interface{}{
			"first_name": "John",
			"last_name":  "Doe",
			"age":        -1,
			"email":      "invalid-email",
			"password":   "123",
		}

		resp := ts.SendRequest(t, "POST", "/api/v1/user/signup", createUserReq, nil)
		assert.Equal(t, 400, resp.StatusCode)
	})

	t.Run("login with correct credentials", func(t *testing.T) {
		loginReq := map[string]interface{}{
			"email":    "john@example.com",
			"password": "Pass123",
		}

		resp := ts.SendRequest(t, "POST", "/api/v1/user/login", loginReq, nil)
		assert.Equal(t, 200, resp.StatusCode)

		var result map[string]interface{}
		err := resp.DecodeBody(&result)
		require.NoError(t, err)
		assert.Contains(t, result, "token")
		token = result["token"].(string)
	})

	t.Run("login with incorrect password", func(t *testing.T) {
		loginReq := map[string]interface{}{
			"email":    "john@example.com",
			"password": "WrongPass123",
		}

		resp := ts.SendRequest(t, "POST", "/api/v1/user/login", loginReq, nil)
		assert.Equal(t, 401, resp.StatusCode)

		var result map[string]interface{}
		err := resp.DecodeBody(&result)
		require.NoError(t, err)
		assert.Contains(t, result["error"], "Invalid credentials")
	})

	t.Run("login with non-existent email", func(t *testing.T) {
		loginReq := map[string]interface{}{
			"email":    "nonexistent@example.com",
			"password": "Pass123",
		}

		resp := ts.SendRequest(t, "POST", "/api/v1/user/login", loginReq, nil)
		assert.Equal(t, 401, resp.StatusCode)
	})

	t.Run("login with invalid request", func(t *testing.T) {
		loginReq := map[string]interface{}{
			"email": "invalid-email",
		}

		resp := ts.SendRequest(t, "POST", "/api/v1/user/login", loginReq, nil)
		assert.Equal(t, 400, resp.StatusCode)
	})

	t.Run("get session", func(t *testing.T) {
		headers := map[string]string{
			"Authorization": "Bearer " + token,
		}

		resp := ts.SendRequest(t, "GET", "/api/v1/session", nil, headers)
		assert.Equal(t, 200, resp.StatusCode)

		var result map[string]interface{}
		err := resp.DecodeBody(&result)
		require.NoError(t, err)

		assert.Equal(t, float64(userID), result["id"])
		assert.Equal(t, "John", result["first_name"])
		assert.Equal(t, "Doe", result["last_name"])
		assert.Equal(t, float64(30), result["age"])
		assert.Equal(t, "john@example.com", result["email"])
		assert.Contains(t, result, "created_at")
		assert.Contains(t, result, "updated_at")
	})

	t.Run("get session without token", func(t *testing.T) {
		resp := ts.SendRequest(t, "GET", "/api/v1/session", nil, nil)
		assert.Equal(t, 401, resp.StatusCode)
	})

	t.Run("get session with invalid token", func(t *testing.T) {
		headers := map[string]string{
			"Authorization": "Bearer invalid-token",
		}

		resp := ts.SendRequest(t, "GET", "/api/v1/session", nil, headers)
		assert.Equal(t, 401, resp.StatusCode)
	})

	t.Run("update user", func(t *testing.T) {
		updateUserReq := map[string]interface{}{
			"first_name": "Johnny",
			"last_name":  "Doe",
			"age":        31,
		}

		headers := map[string]string{
			"Authorization": "Bearer " + token,
		}

		resp := ts.SendRequest(t, "PATCH", "/api/v1/user", updateUserReq, headers)
		assert.Equal(t, 200, resp.StatusCode)

		var result map[string]interface{}
		err := resp.DecodeBody(&result)
		require.NoError(t, err)

		var user models.User
		err = ts.DB.First(&user, userID).Error
		require.NoError(t, err)
		assert.Equal(t, updateUserReq["first_name"], user.FirstName)
		assert.Equal(t, updateUserReq["age"], user.Age)
	})

	t.Run("update user with invalid data", func(t *testing.T) {
		updateUserReq := map[string]interface{}{
			"age": -1,
		}

		headers := map[string]string{
			"Authorization": "Bearer " + token,
		}

		resp := ts.SendRequest(t, "PATCH", "/api/v1/user", updateUserReq, headers)
		assert.Equal(t, 400, resp.StatusCode)
	})

	t.Run("update password", func(t *testing.T) {
		updatePasswordReq := map[string]interface{}{
			"current_password": "Pass123",
			"new_password":     "NewPass123",
		}

		headers := map[string]string{
			"Authorization": "Bearer " + token,
		}

		resp := ts.SendRequest(t, "PATCH", "/api/v1/user/update_password", updatePasswordReq, headers)
		assert.Equal(t, 200, resp.StatusCode)

		loginReq := map[string]interface{}{
			"email":    "john@example.com",
			"password": "NewPass123",
		}

		resp = ts.SendRequest(t, "POST", "/api/v1/user/login", loginReq, nil)
		assert.Equal(t, 200, resp.StatusCode)

		var result map[string]interface{}
		err := resp.DecodeBody(&result)
		require.NoError(t, err)
		assert.Contains(t, result, "token")
		token = result["token"].(string)
	})

	t.Run("update password with invalid current password", func(t *testing.T) {
		updatePasswordReq := map[string]interface{}{
			"current_password": "WrongPass123",
			"new_password":     "NewPass123",
		}

		headers := map[string]string{
			"Authorization": "Bearer " + token,
		}

		resp := ts.SendRequest(t, "PATCH", "/api/v1/user/update_password", updatePasswordReq, headers)
		assert.Equal(t, 401, resp.StatusCode)
	})

	t.Run("update password with invalid new password", func(t *testing.T) {
		updatePasswordReq := map[string]interface{}{
			"current_password": "NewPass123",
			"new_password":     "123",
		}

		headers := map[string]string{
			"Authorization": "Bearer " + token,
		}

		resp := ts.SendRequest(t, "PATCH", "/api/v1/user/update_password", updatePasswordReq, headers)
		assert.Equal(t, 400, resp.StatusCode)
	})

	t.Run("logout", func(t *testing.T) {
		headers := map[string]string{
			"Authorization": "Bearer " + token,
		}

		resp := ts.SendRequest(t, "POST", "/api/v1/user/logout", nil, headers)
		assert.Equal(t, 200, resp.StatusCode)

		resp = ts.SendRequest(t, "GET", "/api/v1/session", nil, headers)
		assert.Equal(t, 401, resp.StatusCode)
	})

	t.Run("logout without token", func(t *testing.T) {
		resp := ts.SendRequest(t, "POST", "/api/v1/user/logout", nil, nil)
		assert.Equal(t, 200, resp.StatusCode)
	})

	t.Run("delete user", func(t *testing.T) {
		loginReq := map[string]interface{}{
			"email":    "john@example.com",
			"password": "NewPass123",
		}

		resp := ts.SendRequest(t, "POST", "/api/v1/user/login", loginReq, nil)
		assert.Equal(t, 200, resp.StatusCode)

		var result map[string]interface{}
		err := resp.DecodeBody(&result)
		require.NoError(t, err)
		token = result["token"].(string)

		headers := map[string]string{
			"Authorization": "Bearer " + token,
		}

		resp = ts.SendRequest(t, "DELETE", "/api/v1/user", nil, headers)
		assert.Equal(t, 200, resp.StatusCode)

		var user models.User
		err = ts.DB.First(&user, userID).Error
		assert.Error(t, err)
	})

	t.Run("delete non-existent user", func(t *testing.T) {
		headers := map[string]string{
			"Authorization": "Bearer " + token,
		}

		resp := ts.SendRequest(t, "DELETE", "/api/v1/user", nil, headers)
		assert.Equal(t, 401, resp.StatusCode)
	})
}
