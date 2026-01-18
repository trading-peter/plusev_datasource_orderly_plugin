package orderly

import (
	"crypto/ed25519"
	"strings"

	"github.com/plusev-terminal/go-plugin-common/logging"
	"github.com/plusev-terminal/go-plugin-common/plugin"
	rt "github.com/plusev-terminal/go-plugin-common/requester/types"

	"github.com/mr-tron/base58"
)

// Client represents a client for the Orderly Network API
type Client struct {
	name       string
	baseURL    string
	wsBaseURL  string
	requester  rt.RequestDoer
	log        *logging.Logger
	accountID  string             // Orderly Account ID (decrypted)
	apiKey     string             // API Key (decrypted)
	privateKey ed25519.PrivateKey // ED25519 Private Key (decrypted)
	isTestnet  bool
}

// NewClient creates a new Orderly API client
func NewClient(req rt.RequestDoer, baseURL string, isTestnet bool) *Client {
	wsBaseURL := "wss://ws-evm.orderly.org"
	if isTestnet {
		wsBaseURL = "wss://testnet-ws-evm.orderly.org"
	}

	return &Client{
		name:      "Orderly Network",
		baseURL:   baseURL,
		wsBaseURL: wsBaseURL,
		requester: req,
		log:       logging.NewLogger("orderly-datasource"),
		isTestnet: isTestnet,
	}
}

func (c *Client) SetCredentials(creds map[string]string) {
	// Store decrypted credentials in the client struct
	if accountID, ok := creds["accountID"]; ok {
		c.accountID = accountID
	}

	if apiKey, ok := creds["apiKey"]; ok {
		c.apiKey = strings.TrimPrefix(apiKey, "ed25519:")
	}

	if privateKeyStr, ok := creds["privateKey"]; ok {
		// Decode the private key from base58
		privateKeySeed, err := base58.Decode(strings.TrimPrefix(privateKeyStr, "ed25519:"))
		if err != nil {
			c.log.ErrorWithData("Failed to decode base58 private key", map[string]any{"error": err})
			return
		}

		// ED25519 seed is 32 bytes, private key is 64 bytes
		if len(privateKeySeed) != ed25519.SeedSize {
			c.log.ErrorWithData("Invalid private key seed size", map[string]any{
				"expected": ed25519.SeedSize,
				"got":      len(privateKeySeed),
			})
			return
		}

		// Generate the full private key from the seed
		c.privateKey = ed25519.NewKeyFromSeed(privateKeySeed)
	}
}

// GetName returns the name of the data source
func (c *Client) GetName() string {
	return c.name
}

func (c *Client) GetConfigFields() []plugin.ConfigField {
	return []plugin.ConfigField{
		{
			Label:       "Account ID",
			Name:        "accountID",
			Description: "Your Orderly Network Account ID",
			Required:    true,
			Encrypt:     true,
			Mask:        false,
		},
		{
			Label:       "API Key",
			Name:        "apiKey",
			Description: "Your Orderly Network API Key",
			Required:    false, // Optional for public data only
			Encrypt:     true,
			Mask:        true,
		},
		{
			Label:       "Private Key",
			Name:        "privateKey",
			Description: "Your ED25519 private key (base58 encoded)",
			Required:    false, // Optional for public data only
			Encrypt:     true,
			Mask:        true,
		},
		{
			Label:       "Use Testnet",
			Name:        "testnet",
			Description: "Connect to Orderly testnet instead of mainnet",
			Required:    false,
			Type:        "boolean",
			Default:     false,
		},
	}
}
