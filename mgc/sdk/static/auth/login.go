package auth

import (
	"context"
	_ "embed"
	"fmt"
	"net/http"
	"net/url"

	"github.com/pkg/browser"
	"magalu.cloud/core"
)

type authResult struct {
	value string
	err   error
}

type LoginParameters struct {
	Show bool `json:"show,omitempty" jsonschema_description:"Show the access token after the login completes"`
}

type LoginResult struct {
	AccessToken string `mapstructure:"accessToken,omitempty" json:"accessToken,omitempty"`
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
		func(ctx context.Context, parameters LoginParameters, _ struct{}) (output *LoginResult, err error) {
			auth := core.AuthFromContext(ctx)
			if auth == nil {
				return nil, fmt.Errorf("unable to retrieve authentication configuration")
			}

			srv, c, err := startCallbackServer(auth)
			if err != nil {
				return nil, err
			}
			defer func() {
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

			if parameters.Show {
				output = &LoginResult{AccessToken: result.value}
			}

			return output, nil
		},
	)
}

func startCallbackServer(auth *core.Auth) (srv *http.Server, c chan *authResult, err error) {
	c = make(chan *authResult, 1)
	callbackUrl, err := url.Parse(auth.RedirectUri())
	if err != nil {
		return nil, nil, fmt.Errorf("invalid redirect_uri configuration")
	}

	srvPort := ":" + callbackUrl.Port()
	srv = &http.Server{Addr: srvPort}

	http.HandleFunc(callbackUrl.Path, newCallback(auth, c))
	go func() {
		err = srv.ListenAndServe()
	}()

	return srv, c, nil
}

func newCallback(auth *core.Auth, c chan *authResult) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		authCode := req.URL.Query().Get("code")
		err := auth.RequestAuthTokeWithAuthorizationCode(authCode)
		if err != nil {
			fmt.Println(err)
			c <- &authResult{value: "", err: err}
			return
		}

		fmt.Println("You are now authenticated.")
		showSuccessPage(w)

		token, err := auth.AccessToken()
		if err != nil {
			c <- &authResult{value: "", err: err}
			return
		}

		c <- &authResult{value: token, err: nil}
	}
}

func showSuccessPage(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, successPage)
}
