package auth

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/pkg/browser"
	"magalu.cloud/core"
)

type AuthResult struct {
	value string
	err   error
}

var (
	//go:embed success.html
	successPage string
)

func newLogin() *core.StaticExecute {
	return core.NewStaticExecute(
		"login",
		"",
		"authenticate with magalu cloud",
		&core.Schema{},
		&core.Schema{},
		func(ctx context.Context, parameters, configs map[string]core.Value) (output core.Value, err error) {
			auth := core.AuthFromContext(ctx)
			if auth == nil {
				return nil, errors.New("unable to retrieve authentication configuration")
			}

			srv, c, err := startCallbackServer(auth)
			if err != nil {
				return nil, err
			}
			defer func() {
				output = nil
				err = srv.Shutdown(context.Background())
			}()

			codeUrl, err := auth.CodeChallengeToURL()
			if err != nil {
				return nil, err
			}

			fmt.Println("Waiting authentication result. Press Ctrl+C if you want to cancel...")
			if err := browser.OpenURL(codeUrl.String()); err != nil {
				return nil, err
			}

			result := <-c
			if result.err != nil {
				return nil, result.err
			}

			return result.value, nil
		},
	)
}

func startCallbackServer(auth *core.Auth) (srv *http.Server, c chan *AuthResult, err error) {
	c = make(chan *AuthResult, 1)
	callbackUrl, err := url.Parse(auth.RedirectUri())
	if err != nil {
		return nil, nil, errors.New("invalid redirect_uri configuration")
	}

	srvPort := ":" + callbackUrl.Port()
	srv = &http.Server{Addr: srvPort}

	http.HandleFunc(callbackUrl.Path, newCallback(auth, c))
	go func() {
		err = srv.ListenAndServe()
	}()

	return srv, c, nil
}

func newCallback(auth *core.Auth, c chan *AuthResult) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		authCode := req.URL.Query().Get("code")
		err := auth.RequestAuthTokeWithAuthorizationCode(authCode)
		if err != nil {
			c <- &AuthResult{value: "", err: err}
		}

		fmt.Println("You are now authenticated.")
		showSuccessPage(w)
		c <- &AuthResult{value: auth.AccessToken(), err: nil}
	}
}

func showSuccessPage(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, successPage)
}
