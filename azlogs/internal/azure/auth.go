package azure

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

// AuthMethod represents the authentication method to use
type AuthMethod int

const (
	// AuthDefault uses DefaultAzureCredential (tries multiple methods)
	AuthDefault AuthMethod = iota
	// AuthCLI uses Azure CLI credentials
	AuthCLI
	// AuthBrowser uses interactive browser authentication
	AuthBrowser
	// AuthManagedIdentity uses Azure Managed Identity
	AuthManagedIdentity
)

// String returns the string representation of the auth method
func (a AuthMethod) String() string {
	switch a {
	case AuthDefault:
		return "Default (auto-detect)"
	case AuthCLI:
		return "Azure CLI"
	case AuthBrowser:
		return "Interactive Browser"
	case AuthManagedIdentity:
		return "Managed Identity"
	default:
		return "Unknown"
	}
}

// Authenticator handles Azure authentication
type Authenticator struct {
	credential azcore.TokenCredential
	method     AuthMethod
}

// NewAuthenticator creates a new authenticator with the specified method
func NewAuthenticator(method AuthMethod) (*Authenticator, error) {
	var cred azcore.TokenCredential
	var err error

	switch method {
	case AuthDefault:
		cred, err = azidentity.NewDefaultAzureCredential(nil)
	case AuthCLI:
		cred, err = azidentity.NewAzureCLICredential(nil)
	case AuthBrowser:
		cred, err = azidentity.NewInteractiveBrowserCredential(nil)
	case AuthManagedIdentity:
		cred, err = azidentity.NewManagedIdentityCredential(nil)
	default:
		return nil, fmt.Errorf("unknown auth method: %d", method)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create credential: %w", err)
	}

	return &Authenticator{
		credential: cred,
		method:     method,
	}, nil
}

// GetCredential returns the Azure credential
func (a *Authenticator) GetCredential() azcore.TokenCredential {
	return a.credential
}

// Method returns the authentication method being used
func (a *Authenticator) Method() AuthMethod {
	return a.method
}

// Validate checks if the credential is valid by attempting to get a token
func (a *Authenticator) Validate(ctx context.Context) error {
	_, err := a.credential.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{"https://api.loganalytics.io/.default"},
	})
	if err != nil {
		return fmt.Errorf("failed to validate credentials: %w", err)
	}
	return nil
}
