package server

import (
	"context"
	"fmt"
	"time"

	"github.com/ipfs/kubo/core/coreapi"
	icore "github.com/ipfs/kubo/core/coreiface"
	"github.com/iris-contrib/middleware/cors"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/middleware/logger"
	"github.com/kataras/iris/v12/middleware/recover"
	"github.com/kataras/iris/v12/sessions"
	bolt "go.etcd.io/bbolt"
	"go.uber.org/zap"

	"github.com/msaldanha/setinstone/event"

	"github.com/msaldanha/pulpit/server/ipfs"
	"github.com/msaldanha/pulpit/server/rest"
	"github.com/msaldanha/pulpit/server/web"
	"github.com/msaldanha/pulpit/service"
)

const (
	dbFile          = ".pulpit.db"
	subsBucket      = "subscriptions"
	addressesBucket = "addresses"
	nameSpace       = "pulpit"
)

type Options struct {
	Url             string
	DataStore       string
	IpfsPort        string
	IpfsApiPort     string
	IpfsGatewayPort string
}

type Response struct {
	Payload interface{} `json:"payload,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type Server struct {
	opts       Options
	store      service.KeyValueStore
	ipfs       icore.CoreAPI
	evmf       event.ManagerFactory
	ps         *service.PulpitService
	secret     string
	logger     *zap.Logger
	ipfsServer *ipfs.IpfsServer
	db         *bolt.DB
	app        *iris.Application
}

func NewServer(opts Options) (*Server, error) {
	logger, er := zap.NewProduction()
	if er != nil {
		return nil, er
	}

	ipfsServer := ipfs.NewIpfsServer(logger, ipfs.ServerOptions{
		IpfsPort:        opts.IpfsPort,
		IpfsApiPort:     opts.IpfsApiPort,
		IpfsGatewayPort: opts.IpfsGatewayPort,
	})

	ctx := context.Background()
	node, er := ipfsServer.SpawnEphemeral(ctx)
	if er != nil {
		panic(fmt.Errorf("failed to spawn ephemeral node: %s", er))
	}
	fmt.Println("IPFS node is running")
	// Attach the Core API to the node
	ipfs, er := coreapi.NewCoreAPI(node)
	if er != nil {
		panic(fmt.Errorf("failed to get ipfs api: %s", er))
	}

	evmf, er := event.NewManagerFactory(nameSpace, ipfs.PubSub(), node.Identity)
	if er != nil {
		panic(fmt.Errorf("failed to setup event manager factory: %s", er))
	}

	db, er := bolt.Open(opts.DataStore, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if er != nil {
		panic(fmt.Errorf("failed to setup DB: %s", er))
	}

	addressStore := service.NewBoltKeyValueStore(db, addressesBucket)

	subsStore, er := service.NewSubscriptionsStore(db, subsBucket)
	if er != nil {
		panic(fmt.Errorf("failed to setup subscriptions DB: %s", er))
	}

	ps := service.NewPulpitService(nameSpace, addressStore, ipfs, node, evmf, logger, subsStore, db)

	app := NewWebApplication()
	web.ConfigureWebServer(app, ps)
	rest.ConfigureApiServer(app, ps)

	return &Server{
		opts:       opts,
		store:      addressStore,
		ipfs:       ipfs,
		evmf:       evmf,
		ps:         ps,
		secret:     "",
		logger:     logger,
		ipfsServer: ipfsServer,
		db:         db,
		app:        app,
	}, nil
}

func (s *Server) Run() error {
	wg := &ChannelWaitGroup{}
	wg.Add(1)
	errCh := make(chan error, 1)
	go func() {
		defer wg.Done()
		if err := s.app.Run(iris.Addr(s.opts.Url)); err != nil {
			errCh <- err
		}
	}()

	select {
	case <-wg.Wait():
		return nil
	case err := <-errCh:
		return err

	}

}

func NewWebApplication() *iris.Application {
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

	return app
}
