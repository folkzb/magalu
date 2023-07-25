package core

import "context"

type Auth struct {
	AccessToken  string
	RefreshToken string
}
type authKey string

var key = authKey("auth")

func NewAuthContext(parentCtx context.Context, auth *Auth) context.Context {
	return context.WithValue(parentCtx, key, auth)
}
func AuthFromContext(ctx context.Context) *Auth {
	a, _ := ctx.Value(key).(*Auth)
	return a
}
