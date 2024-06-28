package controller

import (
	"strings"

	"github.com/kataras/iris/v12/mvc"
	"github.com/msaldanha/timeline"

	"github.com/msaldanha/pulpit/models"
	"github.com/msaldanha/pulpit/server/web/model"
)

const (
	timeLineTemplate     = "timeline.html"
	timeLineItemTemplate = "timeline_item.html"
	postDetailTemplate   = "post.html"
)

type TimelineController struct {
	AuthController
}

func (c *TimelineController) BeforeActivation(b mvc.BeforeActivation) {
	b.Handle("GET", "/{address:string}/{postKey:string}", "GetPost")
}

func (c *TimelineController) Get() mvc.Result {
	items, err := c.Service.GetSubscriptionsPublications(c.ctx, c.Address, "", 40)
	if err != nil {
		return c.fireError(err)
	}
	return view(timeLineTemplate, items, false)
}

func (c *TimelineController) Post(req model.AddPostRequest) mvc.Result {
	connectors := strings.Split(req.Connectors, ",")
	key, err := c.Service.CreateItem(c.ctx, c.Address, "", "", models.AddItemRequest{
		Type: "Post",
		PostItem: models.PostItem{
			Part: timeline.Part{
				MimeType: "text/text",
				Encoding: "",
				Title:    "",
				Body:     req.Body,
			},
			Links:       nil,
			Attachments: nil,
			Connectors:  connectors,
		},
	})
	if err != nil {
		return c.fireError(err)
	}
	item, err := c.Service.GetItemByKey(c.ctx, c.Address, key)
	if err != nil {
		return c.fireError(err)
	}
	return view(timeLineItemTemplate, item, true)
}

func (c *TimelineController) GetBy(address string) mvc.Result {
	items, err := c.Service.GetItems(c.ctx, address, "", "", "", "", 40)
	if err != nil {
		return c.fireError(err)
	}
	return view(timeLineTemplate, items, false)
}

func (c *TimelineController) GetPost(address, postKey string) mvc.Result {
	item, err := c.Service.GetItemByKey(c.ctx, address, postKey)
	if err != nil {
		return c.fireError(err)
	}
	return view(postDetailTemplate, item, false)
}
