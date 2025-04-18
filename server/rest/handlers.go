package rest

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/iris-contrib/middleware/jwt"
	"github.com/kataras/iris/v12"

	"github.com/msaldanha/pulpit/models"
)

func (s *Server) configuredHandlers(app *iris.Application, j *jwt.Middleware) {
	topLevel := app.Party(basePath)

	topLevel.Get("/media", j.Serve, s.getMedia)
	topLevel.Post("/media", j.Serve, s.postMedia)
	topLevel.Post("/login", s.login)

	addresses := topLevel.Party("/addresses")
	addresses.Get("randomaddress", j.Serve, s.getRandomAddress)
	addresses.Get("/", j.Serve, s.getAddresses)
	addresses.Post("/", j.Serve, s.createAddress)
	addresses.Delete("/{addr:string}", s.deleteAddress, j.Serve)

	topLevel.Get("/{addr:string}/publications", s.getItems)
	topLevel.Get("/{addr:string}/publications/{key:string}", s.getItemByKey)
	topLevel.Get("/{addr:string}/publications/{key:string}/{connector:string}", s.getItems)
	topLevel.Post("/{addr:string}/publications", j.Serve, s.createItem)
	topLevel.Post("/{addr:string}/publications/{key:string}/{connector:string}", j.Serve, s.createItem)

	topLevel.Get("/{addr:string}/subscriptions", j.Serve, s.getSubscriptions)
	topLevel.Post("/{addr:string}/subscriptions", j.Serve, s.addSubscription)
	topLevel.Delete("/{addr:string}/subscriptions", j.Serve, s.removeSubscription)
	topLevel.Get("/{addr:string}/subscriptions/publications", s.getSubscriptionsPublications)
	topLevel.Delete("/{addr:string}/subscriptions/publications", j.Serve, s.clearSubscriptionPublications)
}

func (s *Server) createAddress(ctx iris.Context) {
	body := models.LoginRequest{}
	er := ctx.ReadJSON(&body)
	if er != nil {
		returnError(ctx, er, getStatusCodeForError(er))
		return
	}

	pass := body.Password
	if pass == "" {
		returnError(ctx, fmt.Errorf("password cannot be empty"), 400)
		return
	}

	c := context.Background()
	key, er := s.ps.CreateAddress(c, pass)

	if er != nil {
		returnError(ctx, er, getStatusCodeForError(er))
		return
	}

	_ = ctx.JSON(Response{Payload: key})
}

func (s *Server) deleteAddress(ctx iris.Context) {
	addr := ctx.Params().Get("addr")
	c := context.Background()
	er := s.ps.DeleteAddress(c, addr)
	if er != nil {
		returnError(ctx, er, getStatusCodeForError(er))
		return
	}
}

func (s *Server) login(ctx iris.Context) {
	body := models.LoginRequest{}
	er := ctx.ReadJSON(&body)
	if er != nil {
		returnError(ctx, er, getStatusCodeForError(er))
		return
	}

	if body.Address == "" {
		returnError(ctx, fmt.Errorf("address cannot be empty"), 400)
		return
	}

	if body.Password == "" {
		returnError(ctx, fmt.Errorf("password cannot be empty"), 400)
		return
	}

	c := context.Background()
	er = s.ps.Login(c, body.Address, body.Password)
	if er != nil {
		returnError(ctx, er, getStatusCodeForError(er))
		return
	}

	token := jwt.NewTokenWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		addressClaim: body.Address,
	})

	// Sign and get the complete encoded token as a string using the secret
	tokenString, _ := token.SignedString([]byte(s.secret))

	_ = ctx.JSON(Response{Payload: tokenString})
}

func (s *Server) getRandomAddress(ctx iris.Context) {
	c := context.Background()
	a, er := s.ps.GetRandomAddress(c)
	if er != nil {
		returnError(ctx, er, getStatusCodeForError(er))
		return
	}
	_ = ctx.JSON(Response{Payload: a})
}

func (s *Server) getMedia(ctx iris.Context) {
	id := ctx.URLParam("id")
	c, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	f, er := s.ps.GetMedia(c, id)
	if er != nil {
		returnError(ctx, er, 500)
		return
	}
	ctx.Header("Transfer-Encoding", "chunked")
	ctx.StreamWriter(func(w io.Writer) error {
		io.Copy(w, f)
		return nil
	})
}

func (s *Server) postMedia(ctx iris.Context) {
	body := models.AddMediaRequest{}
	er := ctx.ReadJSON(&body)
	if er != nil {
		returnError(ctx, er, getStatusCodeForError(er))
		return
	}

	c := context.Background()
	results := s.ps.PostMedia(c, body.Files)

	_ = ctx.JSON(Response{Payload: results})
}

func (s *Server) getAddresses(ctx iris.Context) {
	c := context.Background()
	addresses, er := s.ps.GetAddresses(c)
	if er != nil {
		returnError(ctx, er, getStatusCodeForError(er))
		return
	}
	_ = ctx.JSON(Response{Payload: addresses})
}

func (s *Server) getItems(ctx iris.Context) {
	addr := ctx.Params().Get("addr")
	keyRoot := ctx.Params().Get("key")
	connector := ctx.Params().Get("connector")
	from := ctx.URLParam("from")
	to := ctx.URLParam("to")
	count := ctx.URLParamIntDefault("count", defaultCount)

	c := context.Background()
	payload, er := s.ps.GetItems(c, addr, keyRoot, connector, from, to, count)
	if er != nil {
		returnError(ctx, er, getStatusCodeForError(er))
		return
	}

	er = ctx.JSON(Response{Payload: payload})
	if er != nil {
		returnError(ctx, er, getStatusCodeForError(er))
		return
	}
}

func (s *Server) getItemByKey(ctx iris.Context) {
	addr := ctx.Params().Get("addr")
	key := ctx.Params().Get("key")

	c := context.Background()
	item, er := s.ps.GetItemByKey(c, addr, key)
	if er != nil {
		returnError(ctx, er, getStatusCodeForError(er))
		return
	}

	resp := Response{}
	if item != nil {
		resp.Payload = item
	}

	er = ctx.JSON(resp)
	if er != nil {
		returnError(ctx, er, 500)
		return
	}
}

func (s *Server) createItem(ctx iris.Context) {
	addr := ctx.Params().Get("addr")
	keyRoot := ctx.Params().Get("key")
	connector := ctx.Params().Get("connector")
	if connector == "" {
		connector = "main"
	}

	tkValue := ctx.Values().Get("jwt")
	if tkValue == nil {
		ctx.StatusCode(401)
		return
	}
	user, ok := tkValue.(*jwt.Token)
	if !ok {
		ctx.StatusCode(401)
		return
	}
	claims := user.Claims.(jwt.MapClaims)
	if claims[addressClaim] != addr {
		ctx.StatusCode(401)
		return
	}

	body := models.AddItemRequest{}
	er := ctx.ReadJSON(&body)
	if er != nil {
		returnError(ctx, er, getStatusCodeForError(er))
		return
	}

	c := context.Background()
	key, er := s.ps.CreateItem(c, addr, keyRoot, connector, body)
	if er != nil {
		returnError(ctx, er, getStatusCodeForError(er))
		return
	}

	_ = ctx.JSON(Response{Payload: key})
}

func (s *Server) getSubscriptions(ctx iris.Context) {
	owner := ctx.Params().Get("addr")
	c := context.Background()
	subscriptions, er := s.ps.GetSubscriptions(c, owner)
	if er != nil {
		returnError(ctx, er, getStatusCodeForError(er))
		return
	}

	resp := Response{}
	if subscriptions != nil {
		resp.Payload = subscriptions
	}

	er = ctx.JSON(resp)
	if er != nil {
		returnError(ctx, er, 500)
		return
	}
}

func (s *Server) getSubscriptionsPublications(ctx iris.Context) {
	owner := ctx.Params().Get("addr")
	from := ctx.URLParam("from")
	count := ctx.URLParamIntDefault("count", defaultCount)
	c := context.Background()
	publications, er := s.ps.GetSubscriptionsPublications(c, owner, from, count)
	if er != nil {
		returnError(ctx, er, getStatusCodeForError(er))
		return
	}

	resp := Response{}
	if publications != nil {
		resp.Payload = publications
	}

	er = ctx.JSON(resp)
	if er != nil {
		returnError(ctx, er, 500)
		return
	}
}

func (s *Server) clearSubscriptionPublications(ctx iris.Context) {
	owner := ctx.Params().Get("addr")
	c := context.Background()
	er := s.ps.ClearSubscriptionsPublications(c, owner)
	if er != nil {
		returnError(ctx, er, getStatusCodeForError(er))
		return
	}

	resp := Response{}

	er = ctx.JSON(resp)
	if er != nil {
		returnError(ctx, er, 500)
		return
	}
}

func (s *Server) addSubscription(ctx iris.Context) {
	addr := ctx.Params().Get("addr")

	user := ctx.Values().Get("jwt").(*jwt.Token)
	claims := user.Claims.(jwt.MapClaims)

	if claims[addressClaim] != addr {
		ctx.StatusCode(401)
		return
	}

	body := models.AddSubscriptionRequest{}
	er := ctx.ReadJSON(&body)
	if er != nil {
		returnError(ctx, er, getStatusCodeForError(er))
		return
	}

	c := context.Background()
	er = s.ps.AddSubscription(c, models.Subscription{
		Owner:   addr,
		Address: body.Address,
	})
	if er != nil {
		returnError(ctx, er, getStatusCodeForError(er))
		return
	}

	_ = ctx.JSON(Response{})
}

func (s *Server) removeSubscription(ctx iris.Context) {
	addr := ctx.Params().Get("addr")

	user := ctx.Values().Get("jwt").(*jwt.Token)
	claims := user.Claims.(jwt.MapClaims)

	if claims[addressClaim] != addr {
		ctx.StatusCode(401)
		return
	}

	body := models.AddSubscriptionRequest{}
	er := ctx.ReadJSON(&body)
	if er != nil {
		returnError(ctx, er, getStatusCodeForError(er))
		return
	}

	c := context.Background()
	er = s.ps.RemoveSubscription(c, models.Subscription{
		Owner:   addr,
		Address: body.Address,
	})
	if er != nil {
		returnError(ctx, er, getStatusCodeForError(er))
		return
	}

	_ = ctx.JSON(Response{})
}
