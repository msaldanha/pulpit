package controller

import (
	"context"

	"github.com/kataras/iris/v12/mvc"

	"github.com/msaldanha/pulpit/models"
	"github.com/msaldanha/pulpit/server/web/model"
)

const subscriptionsTemplate = "subscriptions.html"

type SubscriptionsController struct {
	AuthController
}

func (c *SubscriptionsController) Get() mvc.Result {
	ctx := context.Background()
	items, err := c.Service.GetSubscriptions(ctx, c.Address)
	if err != nil {
		return c.fireError(err)
	}
	return mvc.View{
		Name: subscriptionsTemplate,
		Data: items,
	}
}

func (c *SubscriptionsController) Post(req model.AddSubscriptionRequest) mvc.Result {
	ctx := context.Background()
	err := c.Service.AddSubscription(ctx, models.Subscription{Owner: c.Address, Address: req.Address})
	if err != nil {
		return c.fireError(err)
	}
	items, err := c.Service.GetSubscriptions(ctx, c.Address)
	if err != nil {
		return c.fireError(err)
	}
	return mvc.View{
		Name: subscriptionsTemplate,
		Data: items,
	}
}

func (c *SubscriptionsController) DeleteBy(address string) mvc.Result {
	ctx := context.Background()
	err := c.Service.RemoveSubscription(ctx, models.Subscription{Owner: c.Address, Address: address})
	if err != nil {
		return c.fireError(err)
	}
	items, err := c.Service.GetSubscriptions(ctx, c.Address)
	if err != nil {
		return c.fireError(err)
	}
	return mvc.View{
		Name: subscriptionsTemplate,
		Data: items,
	}
}
