package models

// APIResponse represents a generic API response
type APIResponse struct {
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

// LoginRequest represents login credentials
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// PasswordUpdateRequest represents password update data
type PasswordUpdateRequest struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword    string `json:"new_password" validate:"required,min=6"`
}

// PostUpdateRequest represents post update data
type PostUpdateRequest struct {
	Title string `json:"title" validate:"omitempty,min=3,max=100"`
	Body  string `json:"body" validate:"omitempty,min=10"`
} 