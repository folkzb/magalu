package auth

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"slices"
	"syscall"
	"time"

	"github.com/MagaluCloud/magalu/mgc/core"
	"github.com/MagaluCloud/magalu/mgc/core/auth"
	mgcAuthPkg "github.com/MagaluCloud/magalu/mgc/core/auth"
	"github.com/MagaluCloud/magalu/mgc/core/utils"
	mgcAuthScope "github.com/MagaluCloud/magalu/mgc/sdk/static/auth/scopes"
	"github.com/pkg/browser"
	"github.com/skip2/go-qrcode"
	"go.uber.org/zap"
)

type authResult struct {
	value string
	err   error
}

type loginParameters struct {
	Show     bool `json:"show,omitempty" jsonschema:"description=Show the access token after the login completes"`
	QRcode   bool `json:"qrcode,omitempty" jsonschema:"description=Generate a qrcode for the login URL,default=false"`
	Headless bool `json:"headless,omitempty" jsonschema:"description=Generate URL for the login at local environment,default=false"`
}

type loginResult struct {
	AccessToken    string       `json:"access_token,omitempty"`
	SelectedTenant *auth.Tenant `json:"selected_tenant,omitempty"`
}

const serverShutdownTimeout = 500 * time.Millisecond

var (
	//go:embed success.html
	successPage         string
	loginLoggerInstance *zap.SugaredLogger
)

func loginLogger() *zap.SugaredLogger {
	if loginLoggerInstance == nil {
		loginLoggerInstance = logger().Named("login")
	}
	return loginLoggerInstance
}

var getLogin = utils.NewLazyLoader[core.Executor](func() core.Executor {
	executor := core.NewStaticExecute(
		core.DescriptorSpec{
			Name:    "login",
			Summary: "Authenticate with Magalu Cloud",
			Description: `Log in to your Magalu Cloud account. When you login with this command,
the current Tenant will always be set to the default one. To see more details
about a successful login, use the '--show' flag when logging in`,
		},
		login,
	)

	return core.NewExecuteResultOutputOptions(executor, func(exec core.Executor, result core.Result) string {
		appName := os.Args[0]
		return fmt.Sprintf(`template=Successfully logged in.{{if .access_token}}

Access-token: {{.access_token}}{{end}}
{{if .selected_tenant.legal_name}}
Selected Tenant: {{.selected_tenant.legal_name}} (ID: {{.selected_tenant.uuid}})
{{else}}
Selected Tenant ID: {{.selected_tenant.uuid}}
{{end}}
Run '%s auth tenant list' to list all available Tenants for current login.
`, appName)
	})
})

func login(ctx context.Context, parameters loginParameters, _ struct{}) (*loginResult, error) {
	auth := mgcAuthPkg.FromContext(ctx)
	if auth == nil {
		return nil, fmt.Errorf("programming error: unable to retrieve authentication configuration")
	}
	isHeadless := parameters.QRcode || parameters.Headless

	resultChan, cancel, err := startCallbackServer(ctx, auth, isHeadless)
	if err != nil {
		return nil, err
	}
	defer cancel()

	// Always force built-in parameters
	scopes := core.Scopes{}
	for _, builtIn := range auth.BuiltInScopes() {
		scopes.Add(builtIn)
	}

	// Also add all available scopes by default when logging if no scope is explicitly passed in
	allScopes, err := mgcAuthScope.ListAllAvailable(ctx)
	if err != nil {
		return nil, err
	}

	for _, scope := range allScopes {
		scopes.Add(scope)
	}

	scopes.Add("evt:event-tr")
	scopes.Add("pa:sa:manage")

	codeUrl, err := auth.CodeChallengeToURL(scopes)
	if err != nil {
		return nil, err
	}
	loginLogger().Infow("running login", "codeUrl", codeUrl)

	if isHeadless {
		err = preHeadlessLogin(ctx, parameters, codeUrl, auth, resultChan)
		if err != nil {
			return nil, err
		}
	} else if err := browser.OpenURL(codeUrl.String()); err != nil {
		loginLogger().Infow("Cant't open browser. Logging in a headless environment")
		fmt.Println("Could not open browser, please open it manually: ")
		err := preHeadlessLogin(ctx, parameters, codeUrl, auth, resultChan)
		if err != nil {
			return nil, err
		}
	}

	loginLogger().Infow("waiting authentication result", "redirectUri", auth.RedirectUri())
	result := <-resultChan
	if result.err != nil {
		return nil, result.err
	}

	currentTenant, err := auth.CurrentTenant(ctx)
	if err != nil {
		return nil, err
	}

	checkScopesAfterLogin(auth, scopes)

	loginLogger().Infow("sucessfully logged in")

	output := &loginResult{AccessToken: "", SelectedTenant: currentTenant}

	if parameters.Show {
		output.AccessToken = result.value
	}

	return output, nil
}

func checkScopesAfterLogin(a *auth.Auth, desiredScopes core.Scopes) {
	currentScopes, err := a.CurrentScopes()
	if err != nil {
		loginLogger().Warnw(
			"unable to check if current scopes match desired scopes",
			"err", err,
		)
		return
	}

	missing := core.Scopes{}
	for _, desiredScope := range desiredScopes {
		if !slices.Contains(currentScopes, desiredScope) {
			missing.Add(desiredScope)
		}
	}

	if len(missing) < 1 {
		return
	}

	loginLogger().Debugw(
		"login was successful but resulting scopes were not as requested. This may lead to some operations failing.",
		"missing", missing,
	)
}

func startCallbackServer(ctx context.Context, auth *auth.Auth, isHeadless bool) (resultChan chan *authResult, cancel func(), err error) {
	callbackUrl, err := url.Parse(auth.RedirectUri())
	if err != nil {
		return nil, nil, fmt.Errorf("invalid redirect_uri configuration")
	}

	// Host includes the port, then listen to specific address + port, ex: "localhost:8095"
	addr := callbackUrl.Host

	if envListenAddr := os.Getenv("MGC_LISTEN_ADDRESS"); envListenAddr != "" {
		addr = envListenAddr + ":8095"
	}

	// Listen so we can fail early on bad address, before starting goroutine
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, nil, err
	}

	resultChan = make(chan *authResult, 1)
	callbackChan := make(chan *authResult, 1)
	cancelChan := make(chan struct{}, 1)

	handler := &callbackHandler{
		auth,
		callbackUrl.Path,
		callbackChan,
		ctx,
	}
	srv := &http.Server{Addr: addr, Handler: handler}

	// serve HTTP until shutdown happened, then report result via channel
	serveAndReportResult := func() {
		serverErrorChan := make(chan error, 1)
		signalChan := make(chan os.Signal, 1)
		serverDoneChan := make(chan *authResult, 1)

		signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

		waitChannelsAndShutdownServer := func() {
			var result *authResult

			select {
			case err := <-serverErrorChan:
				result = &authResult{err: fmt.Errorf("Could not serve HTTP: %w", err)}

			case sig := <-signalChan:
				result = &authResult{err: fmt.Errorf("Canceled by signal: %v", sig)}

			case <-cancelChan:
				result = &authResult{err: fmt.Errorf("Canceled by user")}

			case result = <-callbackChan:
			}

			signal.Stop(signalChan)

			ctx, cancelShutdown := context.WithTimeout(context.Background(), serverShutdownTimeout)
			defer cancelShutdown()

			// sync: unblocks serveAndReportResult()/srv.Serve()
			if err := srv.Shutdown(ctx); err != nil {
				srv.Close() // aggressively try to close it
			}

			// sync: notify serveAndReportResult() we're done
			serverDoneChan <- result
		}
		go waitChannelsAndShutdownServer()
		if !isHeadless {
			if err := srv.Serve(listener); !errors.Is(err, http.ErrServerClosed) {
				// sync: unblock waitChannelsAndShutdownServer()
				serverErrorChan <- err
			}
		}

		result := <-serverDoneChan // sync: wait server shutdown by waitChannelsAndShutdownServer()

		close(callbackChan)
		close(cancelChan)

		close(serverErrorChan)
		close(signalChan)
		close(serverDoneChan)

		resultChan <- result
	}

	cancel = func() {
		defer func() {
			// serveAndReportResult() will close channels when done.
			// That means there is nothing to cancel and we should do nothing else, just ignore.
			_ = recover()
		}()

		cancelChan <- struct{}{} // exit waitChannelsAndShutdownServer()
		<-resultChan             // wait serveAndReportResult(), discard as results are not meaningful
	}

	go serveAndReportResult()

	return resultChan, cancel, nil
}

type callbackHandler struct {
	auth *mgcAuthPkg.Auth
	path string
	done chan *authResult
	ctx  context.Context
}

func (h *callbackHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	url := req.URL
	if url.Path != h.path {
		err := fmt.Errorf("Unknown Path: %s", url)
		showErrorPage(w, err, http.StatusNotFound)
		return
	}

	authCode := url.Query().Get("code")
	err := h.auth.RequestAuthTokenWithAuthorizationCode(h.ctx, authCode)
	if err != nil {
		showErrorPage(w, err, http.StatusInternalServerError)
		h.done <- &authResult{err: fmt.Errorf("Could not request auth token with authorization code: %w", err)}
		return
	}

	if err := showSuccessPage(w); err != nil {
		loginLogger().Warnw("could not show whole Succes Page", "error", err)
	}

	token, _ := h.auth.AccessToken(h.ctx) // this is guaranteed if RequestAuthTokeWithAuthorizationCode succeeds
	h.done <- &authResult{value: token}
}

func showSuccessPage(w http.ResponseWriter) error {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "text/html")
	if _, err := io.WriteString(w, successPage); err != nil {
		return err
	}

	return nil
}

func showErrorPage(w http.ResponseWriter, err error, status int) {
	w.WriteHeader(status)
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, "Error: %s", err.Error())
}

func preHeadlessLogin(ctx context.Context, parameters loginParameters, codeUrl *url.URL, auth *auth.Auth, resultChan chan *authResult) error {
	if parameters.QRcode {
		qrCode, err := qrcode.New(codeUrl.String(), qrcode.Low)
		if err != nil {
			return err
		}
		qrCodeString := qrCode.ToSmallString(false)
		fmt.Println(qrCodeString)
	} else if parameters.Headless || (!parameters.Headless && !parameters.QRcode) {
		fmt.Print(codeUrl.String() + "\n\n")
	}
	err := headlessLogin(ctx, auth, resultChan)
	if err != nil {
		return err
	}
	return nil
}

func headlessLogin(ctx context.Context, auth *auth.Auth, resultChan chan *authResult) error {
	var responseUrl string
	fmt.Println("Enter the response URL:")
	_, _ = fmt.Scanln(&responseUrl)

	url, err := url.Parse(responseUrl)
	if err != nil {
		return err
	}

	authCode := url.Query().Get("code")
	if authCode == "" {
		return fmt.Errorf("Invalid response URL!")
	}

	err = auth.RequestAuthTokenWithAuthorizationCode(ctx, authCode)
	if err != nil {
		return err
	}

	token, _ := auth.AccessToken(ctx)
	temp := &authResult{value: token}
	resultChan <- temp
	return nil
}
