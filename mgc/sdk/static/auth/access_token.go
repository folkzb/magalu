package auth

import (
	"context"
	"fmt"

	"magalu.cloud/core"
)

type AccessTokenParameters struct {
	Validate bool `json:",omitempty" jsonschema_description:"Validate the token, refreshing if needed"`
}

type AccessTokenResult struct {
	AccessToken string `mapstructure:"accessToken,omitempty" json:"accessToken,omitempty"`
}

func newAccessToken() *core.StaticExecute {
	return core.NewStaticExecute(
		"access_token",
		"",
		"Retrieve the access token to use the APIs",
		func(ctx context.Context, parameters AccessTokenParameters, _ struct{}) (output *AccessTokenResult, err error) {
			auth := core.AuthFromContext(ctx)
			if auth == nil {
				return nil, fmt.Errorf("unable to retrieve authentication configuration")
			}

			if parameters.Validate {
				err := auth.ValidateAccessToken()
				if err != nil {
					return nil, fmt.Errorf("Could not validate the Access Token: %w", err)
				}
			}

			token, err := auth.AccessToken()
			if err != nil {
				return nil, err
			}

			return &AccessTokenResult{AccessToken: token}, nil
		},
	)
}
