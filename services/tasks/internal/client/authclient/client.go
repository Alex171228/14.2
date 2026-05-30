package authclient

import "context"

type AuthVerifier interface {
	Verify(ctx context.Context, token string) (*VerifyResponse, error)
}
