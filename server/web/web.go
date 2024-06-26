package web

import (
	"os"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/mvc"

	"github.com/msaldanha/pulpit/server/web/controller"
	"github.com/msaldanha/pulpit/service"
)

const basePath = "/mvc"

func ConfigureWebServer(app *iris.Application, service *service.PulpitService) {
	app.RegisterView(iris.HTML("./server/web/views", ".html").Layout("shared/layout.html"))

	app.HandleDir("./server/web/public", iris.Dir("./public"))

	mvc.Configure(app.Party(basePath+"/login"),
		commonControllerSetupFunc(service, new(controller.LoginController)))

	mvc.Configure(app.Party(basePath+"/subscriptions"),
		commonControllerSetupFunc(service, new(controller.SubscriptionsController)))

	mvc.Configure(app.Party(basePath+"/"),
		commonControllerSetupFunc(service, new(controller.TimelineController)))
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
