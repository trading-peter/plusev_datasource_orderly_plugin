package orderly

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	rt "github.com/plusev-terminal/go-plugin-common/requester/types"
	utils "github.com/plusev-terminal/go-plugin-common/wasmutils"

	"github.com/mr-tron/base58"
)

// isAuthenticated returns true if the client has authentication credentials
func (c *Client) isAuthenticated() bool {
	return c.accountID != "" && c.apiKey != "" && c.privateKey != nil
}

// addAuthHeaders adds Orderly authentication headers to a request
func (c *Client) addAuthHeaders(req *rt.Request, body string) {
	if !c.isAuthenticated() {
		return // No authentication configured
	}

	now, _ := utils.Now()
	timestamp := now.UnixMilli()

	// Extract path from URL for signature
	path := req.URL
	if strings.Contains(path, c.baseURL) {
		path = strings.TrimPrefix(path, c.baseURL)
	}

	// Create message to sign: timestamp + method + path + body
	messageToSign := fmt.Sprintf("%d%s%s%s", timestamp, req.Method, path, body)

	// Sign with ED25519
	signature := ed25519.Sign(c.privateKey, []byte(messageToSign))
	signatureBase64 := base64.URLEncoding.EncodeToString(signature)

	// Get the public key from the private key for the orderly-key header
	publicKey := c.privateKey.Public().(ed25519.PublicKey)
	publicKeyBase58 := base58.Encode(publicKey)
	orderlyKey := fmt.Sprintf("ed25519:%s", publicKeyBase58)

	// Set headers
	if req.Headers == nil {
		req.Headers = make(map[string]string)
	}

	req.Headers["orderly-timestamp"] = strconv.FormatInt(timestamp, 10)
	req.Headers["orderly-account-id"] = c.accountID
	req.Headers["orderly-key"] = orderlyKey
	req.Headers["orderly-signature"] = signatureBase64
	req.Headers["Content-Type"] = "application/json"
}
