package trademap

import "github.com/amar-jay/amartrade/internal/trademap/reference"

// ReferenceData returns the reference catalog API backed by this client.
func (client *Client) ReferenceData() *reference.Service {
	return reference.NewService(client)
}
