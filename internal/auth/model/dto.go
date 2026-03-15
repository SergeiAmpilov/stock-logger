package model

// AuthRequest represents the authentication request payload
type AuthRequest struct {
	Password string `json:"password"`
}

// AuthResponse represents the authentication response payload
type AuthResponse struct {
	Token string `json:"token"`
}