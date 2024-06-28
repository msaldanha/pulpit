package controller

import (
	"github.com/kataras/iris/v12/mvc"

	"github.com/msaldanha/pulpit/models"
	"github.com/msaldanha/pulpit/server/web/model"
)

const subscriptionsTemplate = "subscriptions.html"

type SubscriptionsController struct {
	AuthController
}

func (c *SubscriptionsController) Get() mvc.Result {
	items, err := c.Service.GetSubscriptions(c.ctx, c.Address)
	if err != nil {
		return c.fireError(err)
	}
	return view(subscriptionsTemplate, items, false)
}

func (c *SubscriptionsController) Post(req model.AddSubscriptionRequest) mvc.Result {
	err := c.Service.AddSubscription(c.ctx, models.Subscription{Owner: c.Address, Address: req.Address})
	if err != nil {
		return c.fireError(err)
	}
	items, err := c.Service.GetSubscriptions(c.ctx, c.Address)
	if err != nil {
		return c.fireError(err)
	}
	return view(subscriptionsTemplate, items, false)
}

func (c *SubscriptionsController) DeleteBy(address string) mvc.Result {
	err := c.Service.RemoveSubscription(c.ctx, models.Subscription{Owner: c.Address, Address: address})
	if err != nil {
		return c.fireError(err)
	}
	items, err := c.Service.GetSubscriptions(c.ctx, c.Address)
	if err != nil {
		return c.fireError(err)
	}
	return view(subscriptionsTemplate, items, false)
}
