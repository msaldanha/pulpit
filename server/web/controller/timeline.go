package controller

import (
	"context"

	"github.com/kataras/iris/v12/mvc"
)

const timeLineTemplate = "timeline.html"
const postDetailTemplate = "post.html"

type TimelineController struct {
	AuthController
}

func (c *TimelineController) BeforeActivation(b mvc.BeforeActivation) {
	b.Handle("GET", "/{address:string}/{postKey:string}", "GetPost")
}

func (c *TimelineController) Get() mvc.Result {
	ctx := context.Background()
	items, err := c.Service.GetSubscriptionsPublications(ctx, c.Address, "", 40)
	if err != nil {
		return c.fireError(err)
	}
	return mvc.View{
		Name: timeLineTemplate,
		Data: items,
	}
}

func (c *TimelineController) GetBy(address string) mvc.Result {
	ctx := context.Background()
	items, err := c.Service.GetItems(ctx, address, "", "", "", "", 40)
	if err != nil {
		return c.fireError(err)
	}
	return mvc.View{
		Name: timeLineTemplate,
		Data: items,
	}
}

func (c *TimelineController) GetPost(address, postKey string) mvc.Result {
	ctx := context.Background()
	item, err := c.Service.GetItemByKey(ctx, address, postKey)
	if err != nil {
		return c.fireError(err)
	}
	return mvc.View{
		Name: postDetailTemplate,
		Data: item,
	}
}
