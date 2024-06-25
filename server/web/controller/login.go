package controller

import (
	"context"

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
		// Data: page{"User Registration"},
	}
}

func (c *LoginController) Post(req model.LoginRequest) mvc.Result {
	ctx := context.Background()
	err := c.Service.Login(ctx, req.Address, req.Password)
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
