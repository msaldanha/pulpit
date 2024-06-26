package rest

import (
	"errors"
	"os"

	"github.com/iris-contrib/middleware/jwt"
	"github.com/kataras/iris/v12"
	"go.uber.org/zap"

	"github.com/msaldanha/timeline"

	"github.com/msaldanha/pulpit/service"
)

const (
	defaultCount = 20
	addressClaim = "address"
)

type Server struct {
	ps     *service.PulpitService
	secret string
}

type Options struct {
	Url           string
	Store         service.KeyValueStore
	DataStore     string
	Logger        *zap.Logger
	PulpitService *service.PulpitService
}

type Response struct {
	Payload interface{} `json:"payload,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func ConfigureApiServer(app *iris.Application, service *service.PulpitService) {
	srv := &Server{
		secret: os.Getenv("SERVER_SECRET"),
		ps:     service,
	}
	j := jwt.New(jwt.Config{
		ValidationKeyGetter: func(token *jwt.Token) (interface{}, error) {
			return []byte(srv.secret), nil
		},
		SigningMethod: jwt.SigningMethodHS256,
	})
	srv.configuredHandlers(app, j)
}

func returnError(ctx iris.Context, er error, statusCode int) {
	ctx.StatusCode(statusCode)
	_ = ctx.JSON(Response{Error: er.Error()})
}

func getStatusCodeForError(er error) int {
	switch {
	case errors.Is(er, timeline.ErrReadOnly):
		fallthrough
	case errors.Is(er, timeline.ErrCannotRefOwnItem):
		fallthrough
	case errors.Is(er, timeline.ErrCannotRefARef):
		fallthrough
	case errors.Is(er, timeline.ErrCannotAddReference):
		fallthrough
	case errors.Is(er, timeline.ErrNotAReference):
		fallthrough
	case errors.Is(er, timeline.ErrCannotAddRefToNotOwnedItem):
		return 400
	case errors.Is(er, ErrAuthentication):
		return 401
	case errors.Is(er, timeline.ErrNotFound):
		return 404
	default:
		return 500
	}
}
