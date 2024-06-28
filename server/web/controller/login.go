package controller

import (
	"github.com/kataras/iris/v12/mvc"

	"github.com/msaldanha/pulpit/server/web/model"
)

const loginTemplate = "login.html"

type LoginController struct {
	AuthController
}

func (c *LoginController) Get() mvc.Result {
	return mvc.View{
		Name: loginTemplate,
	}
}

func (c *LoginController) Post(req model.LoginRequest) mvc.Result {
	err := c.Service.Login(c.ctx, req.Address, req.Password)
	if err != nil {
		return c.fireError(err)
	}

	// token := jwt.NewTokenWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
	// 	addressClaim: req.Address,
	// })

	// Sign and get the complete encoded token as a string using the secret
	// tokenString, _ := token.SignedString([]byte(c.secret))
	c.Session.Set(sessionIDKey, req.Address)
	return PathTimeline
}
