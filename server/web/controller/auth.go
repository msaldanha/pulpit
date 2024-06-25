package controller

import (
	"strings"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/mvc"
	"github.com/kataras/iris/v12/sessions"

	"github.com/msaldanha/pulpit/service"
)

const (
	sessionIDKey = "Address"
	BasePath     = "/mvc"
	addressClaim = "address"
)

// paths
var (
	PathLogin    = mvc.Response{Path: BasePath + "/login"}
	PathLogout   = mvc.Response{Path: BasePath + "/logout"}
	PathTimeline = mvc.Response{Path: BasePath}
)

type Secret string

func (s Secret) String() string {
	return string(s)
}

// AuthController is the user authentication controller, a custom shared controller.
type AuthController struct {
	// context is auto-binded if struct depends on this,
	// in this controller we don't we do everything with mvc-style,
	// and that's neither the 30% of its features.
	// Ctx iris.Context

	Service *service.PulpitService
	Session *sessions.Session

	// the whole controller is request-scoped because we already depend on Session, so
	// this will be new for each new incoming request, BeginRequest sets that based on the session.
	Address string
	secret  Secret
}

// BeginRequest saves login state to the context, the user id.
func (c *AuthController) BeginRequest(ctx iris.Context) {
	c.Address = c.Session.GetString(sessionIDKey)
	if !strings.Contains(ctx.Path(), PathLogin.Path) && !c.Service.IsLoggedIn(c.Address) {
		ctx.Redirect(PathLogin.Path, iris.StatusFound)
		return
	}
}

// EndRequest is here just to complete the BaseController
// in order to be tell iris to call the `BeginRequest` before the main method.
func (c *AuthController) EndRequest(ctx iris.Context) {}

func (c *AuthController) fireError(err error) mvc.View {
	return mvc.View{
		Code: iris.StatusBadRequest,
		Name: "shared/error.html",
		Data: iris.Map{"Title": "User Error", "Message": strings.ToUpper(err.Error())},
	}
}

func (c *AuthController) redirectTo(address string) mvc.Response {
	return mvc.Response{Path: BasePath + "/addresses/" + address}
}

func (c *AuthController) isLoggedIn() bool {
	// we don't search by session, we have the user id
	// already by the `BeginRequest` middleware.
	return c.Address != ""
}

// if logged in then destroy the session
// and redirect to the login page
// otherwise redirect to the registration page.
func (c *AuthController) logout() mvc.Response {
	if c.isLoggedIn() {
		c.Session.Destroy()
	}
	return PathLogin
}
