package web

import (
	"os"

	"github.com/iris-contrib/middleware/cors"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/middleware/logger"
	"github.com/kataras/iris/v12/middleware/recover"
	"github.com/kataras/iris/v12/mvc"
	"github.com/kataras/iris/v12/sessions"

	"github.com/msaldanha/pulpit/server/web/controller"
	"github.com/msaldanha/pulpit/service"
)

const basePath = "/mvc"

func NewServer(service *service.PulpitService) *iris.Application {

	app := iris.New()
	app.Logger().SetLevel("debug")
	app.Use(recover.New())
	app.Use(logger.New())

	sess := sessions.New(sessions.Config{Cookie: "pulpit"})
	app.Use(sess.Handler())

	crs := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})
	app.Use(crs)
	app.AllowMethods(iris.MethodOptions)

	app.RegisterView(iris.HTML("./pulpit/server/web/views", ".html").Layout("shared/layout.html"))

	app.HandleDir("./pulpit/server/web/public", iris.Dir("./public"))

	mvc.Configure(app.Party(basePath+"/login"),
		commonControllerSetupFunc(service, new(controller.LoginController)))

	mvc.Configure(app.Party(basePath+"/subscriptions"),
		commonControllerSetupFunc(service, new(controller.SubscriptionsController)))

	mvc.Configure(app.Party(basePath+"/"),
		commonControllerSetupFunc(service, new(controller.TimelineController)))

	return app
}

func commonControllerSetupFunc(service *service.PulpitService, ctrl interface{}) func(mvcApp *mvc.Application) {
	return func(mvcApp *mvc.Application) {
		// Register Dependencies.
		var secret controller.Secret
		secret = controller.Secret(os.Getenv("SERVER_SECRET"))
		mvcApp.Register(
			service,
			secret,
		)

		// Register Controllers.
		mvcApp.Handle(ctrl)
	}
}
