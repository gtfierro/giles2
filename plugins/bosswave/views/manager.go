package views

import (
	"encoding/base64"
	"fmt"
	"github.com/pkg/errors"
	bw "gopkg.in/immesys/bw2bind.v5"
	"reflect"
	"strings"
	"sync"
	"time"
)

type MetadataRecord struct {
	Key       string
	Value     []byte
	Namespace string
	URI       string
	VK        string
	Time      int64
}

// returns true if the metadata record can be used
func (rec *MetadataRecord) Valid() bool {
	parts := strings.Split(rec.Key, "/")
	if len(parts) == 0 {
		return false
	}
	key := parts[len(parts)-1]
	return rec.Key == key && rec.URI != "" && rec.VK != "" && rec.Namespace != "" && rec.Time > 0
}

func GetRecord(msg *bw.SimpleMessage) *MetadataRecord {
	parts := strings.Split(msg.URI, "/")
	rec := &MetadataRecord{
		Key:       parts[len(parts)-1],
		Namespace: parts[0],
		URI:       msg.URI,
		VK:        msg.From,
		Time:      time.Now().UnixNano(),
	}

	meta, ok := msg.GetOnePODF(bw.PODFSMetadata).(bw.MetadataPayloadObject)
	if ok {
		val := meta.Value()
		if val != nil {
			rec.Value = []byte(val.Value)
		} else {
			return nil
		}
	} else {
		return nil
	}

	return rec
}

type queryContext struct {
	namespaces map[string]struct{}
	matched    []MetadataRecord
}

func newQueryContext(expr Expression) *queryContext {
	ctx := &queryContext{
		namespaces: make(map[string]struct{}),
		matched:    []MetadataRecord{},
	}

	for _, ns := range expr.NamespaceList {
		ctx.namespaces[ns] = struct{}{}
	}

	return ctx
}

//TODO: fix once we understand how to get the vk
func (ctx *queryContext) hasNamespace(ns string) (found bool) {
	_, found = ctx.namespaces[ns]
	return
}

type ViewManager struct {
	db     metadataDB
	client *bw.BW2Client
	// map of alias -> VK namespace
	namespaceAliases map[string]string
	// subscriptions to each of the namespaces
	namespaceSubscriptions map[string]chan *bw.SimpleMessage
	nsL                    sync.RWMutex

	// map of the hash for an expression to a list of view instances
	expressions map[string][]*View
	expL        sync.RWMutex

	// map of URI -> forwarder for that URI
	forwarders map[string]*forwarder
	fwdL       sync.RWMutex
}

func NewViewManager(client *bw.BW2Client) *ViewManager {
	vm := new(ViewManager)
	vm.client = client
	vm.namespaceAliases = make(map[string]string)
	vm.namespaceSubscriptions = make(map[string]chan *bw.SimpleMessage)
	vm.expressions = make(map[string][]*View)
	vm.forwarders = make(map[string]*forwarder)
	vm.db = CreateSmallMetadataDB()
	return vm
}

func (vm *ViewManager) Handle(v *View) error {
	// setup subscriptions to the namespaces if we haven't already
	vm.subscribeNamespaces(v.Expr)
	// replace all of the namespace aliases with the expanded VK
	// so we can do namespace matching on the received URIs
	newNS := make([]string, len(v.Expr.NamespaceList))
	for i, ns := range v.Expr.NamespaceList {
		newNS[i] = vm.namespaceAliases[ns]
	}
	v.Expr.NamespaceList = newNS

	vm.addView(v)

	// now, when we run vm.db.Exec(v.Expr), we will get the list of metadata records
	// for the matching URIs. We want to subscribe to those URIs
	initialMatches := vm.db.Exec(v.Expr)
	for _, rec := range initialMatches {
		vm.startForwarding(rec.URI, v)
	}
	return nil
}

func (vm *ViewManager) subscribeNamespaces(expr Expression) {
	var (
		persistedMetadata chan *bw.SimpleMessage
	)
	for _, namespace := range expr.NamespaceList {
		// canonicalize the namespace
		uri := strings.TrimSuffix(namespace, "/") + "/*/!meta/+"
		// check if we are already subscribed to the namespace
		vm.nsL.RLock()
		_, found := vm.namespaceSubscriptions[namespace]
		vm.nsL.RUnlock()
		if found {
			continue
		}

		// if we are not subscribed to the namespace, we need to resolve the namespace into
		// its actual VK, because that is what will be in the URIs we receive on subscriptions.
		// We may actually end up getting the naked VK in the namespace, but it is more likely
		// that we will receive an alias which we will have to resolve.

		// resolve namespace
		ro, _, err := vm.client.ResolveRegistry(namespace)
		if err != nil {
			log.Fatal(errors.Wrapf(err, "Could not resolve namespace %s", namespace))
		}
		// OKAY so the issue here is that bw2's objects package is vendored, and runs into
		// conflict when used with the bw2bind package. So, we cannot import the objects
		// package. We only need the objects package to get the *objects.Entity object from
		// the RoutingObject interface we get from calling ResolveRegistry. The reason why we
		// need an Entity object is so we can call its GetVK() method to get the namespace VK
		// that is mapped to by the alias we threw into ResolveRegistry.
		// Because the underlying object actually is an entity object, we can use the reflect
		// package to just call the method directly without having to import the objects
		// package to do the type conversion (e.g. ro.(*object.Entity).GetVK()).
		// The rest is just reflection crap: call the method using f.Call() using []reflect.Value
		// to indicate an empty arguments list. We use [0] to get the first (and only) result,
		// and call .Bytes() to return the underlying byte array returned by GetVK(). We
		// then interpret it using base64 urlsafe encoding to get the string value.
		f := reflect.ValueOf(ro).MethodByName("GetVK")
		nsvk := base64.URLEncoding.EncodeToString(f.Call([]reflect.Value{})[0].Bytes())
		vm.namespaceAliases[namespace] = nsvk
		log.Noticef("Resolved alias %s -> %s", namespace, nsvk)

		// subscribe to all !meta tags on that namespace
		vm.namespaceSubscriptions[namespace], err = vm.client.Subscribe(&bw.SubscribeParams{
			URI: uri,
		})
		if err != nil {
			log.Fatal(errors.Wrapf(err, "Could not subscribe to namespace %s", uri))
		}
		// handle the subscriptions
		go func(sub chan *bw.SimpleMessage) {
			// process subscription
			for msg := range sub {
				rec := GetRecord(msg)
				vm.db.Insert(rec)
			}
		}(vm.namespaceSubscriptions[namespace])
		// Subscriptions only give us metadata messages that appear AFTER the subscription begins,
		// so we execute a query to get all metadata messages that were there already
		persistedMetadata, err = vm.client.Query(&bw.QueryParams{
			URI: uri,
		})
		if err != nil {
			log.Fatal(errors.Wrapf(err, "Could not query namespace %s", uri))
		}
		go func() {
			for msg := range persistedMetadata {
				rec := GetRecord(msg)
				vm.db.Insert(rec)
			}
		}()
	}
}

func (vm *ViewManager) addView(v *View) {
	vm.expL.Lock()
	hash := v.Expr.Hash()
	if list, found := vm.expressions[hash]; found {
		vm.expressions[hash] = append(list, v)
	} else {
		vm.expressions[hash] = []*View{v}
	}
	vm.expL.Unlock()
}

func (vm *ViewManager) startForwarding(uri string, views ...*View) error {
	vm.fwdL.Lock()
	defer vm.fwdL.Unlock()
	if _, found := vm.forwarders[uri]; !found {
		subscription, err := vm.client.Subscribe(&bw.SubscribeParams{
			URI: uri,
		})
		if err != nil {
			return err
		}
		vm.forwarders[uri] = newForwarder(subscription, uri)
	}
	vm.forwarders[uri].addViews(views...)
	for _, v := range views {
		v.MatchSet[uri] = true
	}
	return nil
}

func (vm *ViewManager) stopForwarding(uri string, views ...*View) {
	vm.fwdL.RLock()
	defer vm.fwdL.RUnlock()
	if forwarder, found := vm.forwarders[uri]; found {
		forwarder.removeViews(views...)
	}
}

// iterate through all expressions and update appropriate forwarders
func (vm *ViewManager) checkViews() {
	vm.expL.RLock()
	defer vm.expL.RUnlock()
	// for each of the expressions we know about
	for _, viewList := range vm.expressions {
		// rerun the expression to get the list of matching URIs
		matching := vm.db.Exec(viewList[0].Expr)
		view := viewList[0]
		for uri := range view.MatchSet {
			view.MatchSet[uri] = false
		}
		// for all matching URIs, check if that URI previously matched
		for _, rec := range matching {
			if _, found := view.MatchSet[rec.URI]; !found {
				// if it did not match, then forward
				vm.startForwarding(rec.URI, viewList...)
				view.MatchSet[rec.URI] = true
			}
		}
		for uri, doesItStay := range view.MatchSet {
			if !doesItStay {
				vm.stopForwarding(uri, viewList...)
			}
		}
	}
}

// This handles all of the metadata storage and querying. There are two flavors
// of this:
//   - a small, in-memory database for views that don't require a ton of resources.
//     This will have no indexing, probably just using linear searches through Go's maps
//     and arrays.
//   - a fuller database (probably https://github.com/cznic/ql) with indexing, good for
//     more complex and larger views, and should be more performant.
type metadataDB interface {
	Insert(*MetadataRecord) error
	Query(string) []MetadataRecord
	Exec(Expression) []MetadataRecord
}

type memoryMetadataDB struct {
	rl sync.RWMutex
	// this is a map from the full URI of metadata to the metadata record. This kind
	// of indexing lets us update metadata records very quickly
	records map[string]MetadataRecord
}

func CreateSmallMetadataDB() metadataDB {
	return &memoryMetadataDB{
		records: make(map[string]MetadataRecord),
	}
}

func (mm *memoryMetadataDB) Insert(rec *MetadataRecord) error {
	if rec == nil {
		return nil
	}
	if !rec.Valid() {
		return errors.New(fmt.Sprintf("Record %+v was invalid", rec))
	}
	//log.Noticef("Inserting Record %+v", rec)
	mm.rl.Lock()
	defer mm.rl.Unlock()
	if _, found := mm.records[rec.URI]; found {
		if len(rec.Value) == 0 { // len of 0 means delete this key
			delete(mm.records, rec.URI)
			goto done
		}
	}
	mm.records[rec.URI] = *rec
done:
	return nil
}

func (mm *memoryMetadataDB) Exec(expr Expression) []MetadataRecord {
	ctx := newQueryContext(expr)
	if expr.N != nil {
		res := mm.getNode(ctx, expr.N)
		log.Noticef("RESULTS %+v", res)
	}
	return []MetadataRecord{}
}

func (mm *memoryMetadataDB) Query(query string) []MetadataRecord {
	return []MetadataRecord{}
}

func (mm *memoryMetadataDB) getNode(ctx *queryContext, n Node) []MetadataRecord {
	mm.rl.RLock()
	defer mm.rl.RUnlock()
	var matched []MetadataRecord
	for _, rec := range mm.records {
		// if the record is not in the right namespace, skip it
		if !ctx.hasNamespace(rec.Namespace) {
			log.Debugf("wrong namespace: %v vs %v", rec.Namespace, ctx.namespaces)
			continue
		}
		if n.Eval(&rec) {
			matched = append(matched, rec)
		}
	}
	return matched
}
