package service

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	files "github.com/ipfs/boxo/files"
	"github.com/ipfs/boxo/path"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/kubo/core"
	icore "github.com/ipfs/kubo/core/coreiface"
	bolt "go.etcd.io/bbolt"
	"go.uber.org/zap"

	"github.com/msaldanha/setinstone/address"
	"github.com/msaldanha/setinstone/crypto"
	"github.com/msaldanha/setinstone/event"
	"github.com/msaldanha/setinstone/graph"
	"github.com/msaldanha/timeline"

	"github.com/msaldanha/pulpit/models"
)

const addressValue = "address"

type PulpitService struct {
	store              KeyValueStore
	timelines          map[string]*timeline.Timeline
	ipfs               icore.CoreAPI
	node               *core.IpfsNode
	logins             map[string]string
	evmFactory         event.ManagerFactory
	logger             *zap.Logger
	subsStore          SubscriptionsStore
	compositeTimelines map[string]*timeline.CompositeTimeline
	nameSpace          string
	db                 *bolt.DB
}

func NewPulpitService(nameSpace string, store KeyValueStore, ipfs icore.CoreAPI, node *core.IpfsNode, evmFactory event.ManagerFactory,
	logger *zap.Logger, subsStore SubscriptionsStore, db *bolt.DB) *PulpitService {
	return &PulpitService{
		store:              store,
		ipfs:               ipfs,
		node:               node,
		timelines:          map[string]*timeline.Timeline{},
		logins:             map[string]string{},
		evmFactory:         evmFactory,
		logger:             logger.Named("Pulpit"),
		subsStore:          subsStore,
		compositeTimelines: map[string]*timeline.CompositeTimeline{},
		nameSpace:          nameSpace,
		db:                 db,
	}
}

func (s *PulpitService) CreateAddress(ctx context.Context, pass string) (string, error) {
	if pass == "" {
		return "", fmt.Errorf("password cannot be empty")
	}

	a, er := address.NewAddressWithKeys()
	if er != nil {
		return "", er
	}

	dbAddress := a.Clone()
	dbAddress.Keys.PrivateKey = hex.EncodeToString(crypto.Encrypt([]byte(dbAddress.Keys.PrivateKey), pass))
	ar := AddressRecord{
		Address:  *dbAddress,
		Bookmark: crypto.Encrypt([]byte(bookmarkFlag), pass),
	}

	er = s.store.Put(dbAddress.Address, ar.ToBytes())
	if er != nil {
		return "", er
	}

	s.logins[a.Address] = pass

	return a.Address, nil
}

func (s *PulpitService) DeleteAddress(ctx context.Context, addr string) error {
	_, found, er := s.store.Get(addr)
	if er != nil {
		return er
	}
	if !found {
		return errors.New("addr not found in local storage")
	}
	er = s.store.Delete(addr)
	if er != nil {
		return er
	}
	return nil
}

func (s *PulpitService) Login(ctx context.Context, addr, password string) error {
	if addr == "" {
		return fmt.Errorf("address cannot be empty")
	}

	if password == "" {
		return fmt.Errorf("password cannot be empty")
	}

	a, er := s.getAddress(addr, password)
	if er != nil || !a.HasKeys() {
		return errors.New("invalid addr or password")
	}

	s.logins[addr] = password

	tl, err := s.createTimeLine(a)
	if err != nil {
		return err
	}
	compositeTimeline, err := s.createCompositeTimeLine(a)
	if err != nil {
		return err
	}
	err = compositeTimeline.AddTimeline(tl)
	if err != nil {
		return err
	}

	return nil
}

func (s *PulpitService) IsLoggedIn(addr string) bool {
	_, found := s.logins[addr]
	return found
}

func (s *PulpitService) GetRandomAddress(ctx context.Context) (*address.Address, error) {
	a, er := address.NewAddressWithKeys()
	if er != nil {
		return nil, er
	}
	return a, nil
}

func (s *PulpitService) GetMedia(ctx context.Context, id string) (io.Reader, error) {
	c, er := cid.Parse(id)
	if er != nil {
		return nil, er
	}
	p := path.FromCid(c)

	node, er := s.ipfs.Unixfs().Get(ctx, p)
	if er == context.DeadlineExceeded {
		return nil, fmt.Errorf("not found: %s", id)
	}
	if er != nil {
		return nil, er
	}
	f, ok := node.(files.File)
	if !ok {
		return nil, fmt.Errorf("not a file: %s", id)
	}
	return f, nil
}

func (s *PulpitService) PostMedia(ctx context.Context, f []string) []models.AddMediaResult {
	results := []models.AddMediaResult{}
	c := context.Background()
	for _, v := range f {
		id, er := s.addFile(c, v)
		if er != nil {
			results = append(results, models.AddMediaResult{
				File:  v,
				Id:    id,
				Error: er.Error(),
			})
		} else {
			results = append(results, models.AddMediaResult{
				File:  v,
				Id:    id,
				Error: "",
			})
		}

	}
	return results
}

func (s *PulpitService) GetAddresses(ctx context.Context) ([]*address.Address, error) {
	all, er := s.store.GetAll()
	if er != nil {
		return nil, er
	}
	addresses := []*address.Address{}
	for _, v := range all {
		ar := AddressRecord{}
		_ = ar.FromBytes(v)
		addresses = append(addresses, &ar.Address)
	}
	return addresses, nil
}

func (s *PulpitService) GetItems(ctx context.Context, addr, keyRoot, connector, from, to string, count int) ([]timeline.Item, error) {
	if connector == "" {
		connector = "main"
	}

	tl, er := s.getTimeline(addr)
	if er != nil {
		return nil, er
	}

	items, er := tl.GetFrom(ctx, keyRoot, connector, from, to, count)
	if er != nil && !errors.Is(er, timeline.ErrNotFound) {
		return nil, er
	}

	return items, nil
}

func (s *PulpitService) GetItemByKey(ctx context.Context, addr, key string) (*timeline.Item, error) {
	ctl, found := s.getCompositeTimeline(ctx)
	if found {
		item, found, _ := ctl.Get(ctx, key)
		if found {
			return &item, nil
		}
	}

	tl, er := s.getTimeline(addr)
	if er != nil {
		return nil, er
	}

	item, ok, er := tl.Get(ctx, key)
	if er != nil {
		return nil, er
	}

	if ok {
		return &item, nil
	}

	return nil, nil
}

func (s *PulpitService) CreateItem(ctx context.Context, addr, keyRoot, connector string, body models.AddItemRequest) (string, error) {
	if connector == "" {
		connector = "main"
	}

	tl, er := s.getTimeline(addr)
	if er != nil {
		return "", er
	}

	key := ""
	switch body.Type {
	case timeline.TypePost:
		key, er = s.createPost(ctx, tl, body.PostItem, keyRoot, connector)
	case timeline.TypeReference:
		key, er = s.createReference(ctx, tl, body.ReferenceItem, keyRoot, connector)
	default:
		er = fmt.Errorf("unknown type %s", body.Type)
		return "", er
	}

	return key, er
}

func (s *PulpitService) AddSubscription(ctx context.Context, sub models.Subscription) error {
	compositeTimeline, found := s.compositeTimelines[sub.Owner]
	if !found {
		var err error
		dao := timeline.NewCompositeDao(s.db, sub.Owner)
		compositeTimeline, err = timeline.NewCompositeTimeline(s.nameSpace, s.node, s.evmFactory, s.logger, sub.Owner, dao)
		if err != nil {
			return fmt.Errorf("unable to create composite timeline for owner %s %w", sub.Owner, err)
		}
		err = compositeTimeline.Init()
		if err != nil {
			return fmt.Errorf("unable to init composite timeline for owner %s %w", sub.Owner, err)
		}
		s.compositeTimelines[sub.Owner] = compositeTimeline
	}
	addr := &address.Address{Address: sub.Address}
	err := compositeTimeline.LoadTimeline(addr)
	if err != nil {
		return err
	}
	return s.subsStore.AddSubscription(sub)
}

func (s *PulpitService) RemoveSubscription(ctx context.Context, sub models.Subscription) error {
	compositeTimeline, found := s.compositeTimelines[sub.Owner]
	if !found {
		return fmt.Errorf("no composite timeline for owner %s", sub.Owner)
	}
	err := compositeTimeline.RemoveTimeline(sub.Address)
	if err != nil {
		return err
	}
	return s.subsStore.RemoveSubscription(sub)
}

func (s *PulpitService) GetSubscriptions(ctx context.Context, owner string) ([]models.Subscription, error) {
	return s.subsStore.GetAllSubscriptionsForOwner(owner)
}

func (s *PulpitService) GetSubscriptionsPublications(ctx context.Context, owner, from string, count int) ([]timeline.Item, error) {
	compositeTimeline, found := s.compositeTimelines[owner]
	if !found {
		return nil, fmt.Errorf("no composite timeline for owner %s", owner)
	}
	items, err := compositeTimeline.GetFrom(ctx, from, count)
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (s *PulpitService) ClearSubscriptionsPublications(ctx context.Context, owner string) error {
	compositeTimeline, found := s.compositeTimelines[owner]
	if !found {
		return fmt.Errorf("no composite timeline for owner %s", owner)
	}
	err := compositeTimeline.Clear()
	if err != nil {
		return err
	}
	return nil
}

func (s *PulpitService) createPost(ctx context.Context, tl *timeline.Timeline, postItem models.PostItem, keyRoot, connector string) (string, error) {
	if len(postItem.Connectors) == 0 {
		er := fmt.Errorf("reference types cannot be empty")
		return "", er
	}
	for _, v := range postItem.Connectors {
		if v == "" {
			er := fmt.Errorf("reference types cannot contain empty value")
			return "", er
		}
	}

	post, er := s.toTimelinePost(postItem)
	if er != nil {
		return "", er
	}
	key, er := tl.AppendPost(ctx, post, keyRoot, connector)
	if er != nil {
		return "", er
	}
	return key, nil
}

func (s *PulpitService) createReference(ctx context.Context, tl *timeline.Timeline, postItem models.ReferenceItem, keyRoot, connector string) (string, error) {
	if postItem.Target == "" {
		er := fmt.Errorf("target cannot be empty")
		return "", er
	}

	post := s.toTimelineReference(postItem)

	key, er := tl.AppendReference(ctx, post, keyRoot, connector)
	if er != nil {
		return "", er
	}
	return key, nil
}

func (s *PulpitService) getTimeline(addr string) (*timeline.Timeline, error) {
	tl, found := s.timelines[addr]
	if found {
		return tl, nil
	}

	pass := s.logins[addr]

	a, er := s.getAddress(addr, pass)
	if er != nil {
		return nil, er
	}

	return s.createTimeLine(a)
}

func (s *PulpitService) getCompositeTimeline(ctx context.Context) (*timeline.CompositeTimeline, bool) {
	addr := s.extractAddress(ctx)
	if addr == "" {
		return nil, false
	}
	tl, found := s.compositeTimelines[addr]
	return tl, found
}

func (s *PulpitService) getAddress(addr, pass string) (*address.Address, error) {
	var a address.Address
	a = address.Address{Address: addr}
	if pass != "" {
		buf, found, _ := s.store.Get(addr)
		if found {
			ar := AddressRecord{}
			er := ar.FromBytes(buf)
			if er != nil {
				return nil, er
			}
			a = ar.Address
			privKey, er := hex.DecodeString(ar.Address.Keys.PrivateKey)
			if er != nil {
				return nil, er
			}
			pk, er := crypto.Decrypt(privKey, pass)
			if er != nil {
				return nil, er
			}
			a.Keys.PrivateKey = string(pk)
		}
	}
	return &a, nil
}

func (s *PulpitService) createTimeLine(a *address.Address) (*timeline.Timeline, error) {
	gr := graph.New(s.nameSpace, a, s.node, s.logger)
	tl, er := timeline.NewTimeline(s.nameSpace, a, gr, s.evmFactory, s.logger)
	if er != nil {
		return nil, er
	}
	s.timelines[a.Address] = tl
	return tl, nil
}

func (s *PulpitService) createCompositeTimeLine(a *address.Address) (*timeline.CompositeTimeline, error) {
	dao := timeline.NewCompositeDao(s.db, a.Address)
	compositeTimeline, er := timeline.NewCompositeTimeline(s.nameSpace, s.node, s.evmFactory, s.logger, a.Address, dao)
	if er != nil {
		return nil, fmt.Errorf("failed to create composite timeline: %s", er.Error())
	}
	er = compositeTimeline.Init()
	if er != nil {
		return nil, fmt.Errorf("failed to init composite timeline: %s", er.Error())
	}
	subs, er := s.subsStore.GetAllSubscriptionsForOwner(a.Address)
	if er != nil {
		return nil, fmt.Errorf("failed to read subscriptions: %s", er.Error())
	}
	for _, sub := range subs {
		addr := &address.Address{Address: sub.Address}
		err := compositeTimeline.LoadTimeline(addr)
		if err != nil {
			return nil, fmt.Errorf("failed to load subscription: %s", err.Error())
		}
	}
	er = compositeTimeline.Run()
	if er != nil {
		return nil, fmt.Errorf("failed to run composite timeline: %s", er.Error())
	}
	s.compositeTimelines[a.Address] = compositeTimeline
	return compositeTimeline, nil
}

func (s *PulpitService) toTimelineReference(referenceItem models.ReferenceItem) timeline.Reference {
	return timeline.Reference{
		Target:    referenceItem.Target,
		Connector: referenceItem.Connector,
		Base: timeline.Base{
			Type: timeline.TypeReference,
		},
	}
}
func (s *PulpitService) toTimelinePost(postItem models.PostItem) (timeline.Post, error) {
	post := timeline.Post{
		Part:  postItem.Part,
		Links: postItem.Links,
		Base: timeline.Base{
			Type:       timeline.TypePost,
			Connectors: postItem.Connectors,
		},
	}
	c := context.Background()
	for i, v := range postItem.Attachments {
		mimeType, er := getFileContentType(v)
		if er != nil {
			return timeline.Post{}, er
		}
		cid, er := s.addFile(c, v)
		if er != nil {
			return timeline.Post{}, er
		}
		post.Attachments = append(post.Attachments, timeline.PostPart{
			Seq:  i + 1,
			Name: filepath.Base(v),
			Part: timeline.Part{
				MimeType: mimeType,
				Encoding: "",
				Body:     "ipfs://" + cid,
			},
		})
	}

	return post, nil
}

func (s *PulpitService) addFile(ctx context.Context, name string) (string, error) {
	someFile, er := getUnixfsNode(name)
	if er != nil {
		return "", er
	}

	cidFile, er := s.ipfs.Unixfs().Add(ctx, someFile)
	if er != nil {
		return "", er
	}

	fmt.Printf("Added file to IPFS with CID %s\n", cidFile.String())
	return cidFile.String(), nil
}

func (s *PulpitService) extractAddress(ctx context.Context) string {
	v := ctx.Value(addressValue)
	if v == nil {
		return ""
	}
	addr, ok := v.(string)
	if !ok {
		return ""
	}
	if _, exists := s.logins[addr]; !exists {
		return ""
	}
	return addr
}

func getFileContentType(path string) (string, error) {
	f, er := os.Open(path)
	if er != nil {
		return "", er
	}
	defer f.Close()

	// Only the first 512 bytes are used to sniff the content type.
	buffer := make([]byte, 512)

	_, er = f.Read(buffer)
	if er != nil {
		return "", er
	}

	// Use the net/http package's handy DetectContentType function. Always returns a valid
	// content-type by returning "application/octet-stream" if no others seemed to match.
	contentType := http.DetectContentType(buffer)

	return contentType, nil
}

func getUnixfsNode(path string) (files.Node, error) {
	st, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	f, err := files.NewSerialFile(path, false, st)
	if err != nil {
		return nil, err
	}

	return f, nil
}
