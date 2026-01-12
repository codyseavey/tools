package azure

import (
	"context"
	"testing"
	"time"
)

func TestNewAuthenticator_CLI(t *testing.T) {
	auth, err := NewAuthenticator(AuthCLI)
	if err != nil {
		t.Fatalf("Failed to create CLI authenticator: %v", err)
	}

	if auth.Method() != AuthCLI {
		t.Errorf("Expected method %v, got %v", AuthCLI, auth.Method())
	}

	if auth.GetCredential() == nil {
		t.Error("Expected non-nil credential")
	}
}

func TestNewAuthenticator_Default(t *testing.T) {
	auth, err := NewAuthenticator(AuthDefault)
	if err != nil {
		t.Fatalf("Failed to create default authenticator: %v", err)
	}

	if auth.Method() != AuthDefault {
		t.Errorf("Expected method %v, got %v", AuthDefault, auth.Method())
	}

	if auth.GetCredential() == nil {
		t.Error("Expected non-nil credential")
	}
}

func TestAuthenticator_Validate(t *testing.T) {
	auth, err := NewAuthenticator(AuthCLI)
	if err != nil {
		t.Fatalf("Failed to create authenticator: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = auth.Validate(ctx)
	if err != nil {
		t.Errorf("Token validation failed: %v", err)
		t.Log("Make sure you are logged in with 'az login'")
	}
}

func TestAuthMethodString(t *testing.T) {
	tests := []struct {
		method   AuthMethod
		expected string
	}{
		{AuthDefault, "Default (auto-detect)"},
		{AuthCLI, "Azure CLI"},
		{AuthBrowser, "Interactive Browser"},
		{AuthManagedIdentity, "Managed Identity"},
		{AuthMethod(99), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.method.String(); got != tt.expected {
				t.Errorf("AuthMethod.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}
