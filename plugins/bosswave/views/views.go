package views

import (
	"github.com/op/go-logging"
	bw "gopkg.in/immesys/bw2bind.v5"
	"os"
	"regexp"
	"sync"
)

// logger
var log *logging.Logger

// stores metadata
var VM *ViewManager

var vmInitializer sync.Once

// set up logging facilities
func init() {
	log = logging.MustGetLogger("views")
	var format = "%{color}%{level} %{time:Jan 02 15:04:05} %{shortfile}%{color:reset} ▶ %{message}"
	var logBackend = logging.NewLogBackend(os.Stderr, "", 0)
	logBackendLeveled := logging.AddModuleLevel(logBackend)
	logging.SetBackend(logBackendLeveled)
	logging.SetFormatter(logging.MustStringFormatter(format))
}

// A view is a group of URIs where the metadata of each URI fulfills the view's predicate.
// View documentation is listed at github.com/immesys/bw2/wiki/Views; we want to provide a
// similar interface. The problem with current views is they look for service interfaces, which
// is too specific -- we want to be able to subscribe to a set of URIs defined but metadata.
//
// A predicate defines a View. A Predicate has 3 components: a list of namespaces, a list of URI
// attributes, and a list of metadata operations. Like the native BW2 views, the the list of namespaces
// specifies the "scope" of the URI and metadata components. URI attributes specify equality,
// and glob/regex matching. Metadata operations capture equality, glob/regex, existence, and possibly some set operations.
// On top of this, URI and metadata predicates support and/or/not expressions.
//
// We create a view using some specification of a predicate. Under the covers, this will likely
// be something very similar to the native BW2 Views: set of nested structs/dictionaries. In the future,
// we may want to write a quick little query language so that they can be specified as strings.
type View struct {
	// the bosswave client so that we can create subscriptions
	client *bw.BW2Client
	// the channel of messages received on this View
	C chan *bw.SimpleMessage
	// the expression that defines this view
	Expr Expression
	// This is the collection of URIs that this view matches
	MatchSet map[string]bool
}

// The next issue is: how do we define and evaluate predicates? The view instance will subscribe to all
// metadata on all URIs that match the URI predicates in the provided namespaces, and then filter down
// the URIs based on their metadata and the view's metadata predicates from there.
//
// How does this start? First, we subscribe to *all* metadata in all namespaces mentioned in the View:
//   for each ns in namespaces:
//       subscribe to ns/*/!meta/+
//   for each ns in namespaces:
//       query ns/*/!meta/+
//
// We build up a large index of URIs mapped to namespaces along with the metadata on each URI. For now, lets do this in-memory,
// but we can imagine more elaborate indexing structures and even external storage that tracks the history.
//
// What information do we need from the metadata messages so that we can perform searches on them?
// What do we have? For each metadata message:
//  - metadata key
//  - metadata values
//  - URI
//  - VK of publisher
func CreateView(client *bw.BW2Client, expr Expression) (v *View, err error) {
	v = new(View)
	v.client = client
	v.C = make(chan *bw.SimpleMessage, 1000)
	v.Expr = expr
	v.MatchSet = make(map[string]bool)
	vmInitializer.Do(func() { VM = NewViewManager(client) })

	VM.Handle(v)

	return
}

// marks all matchset entries as false so we can tell which of them
// changed
func (v *View) _setMatchSetTo(val bool) {
	for uri := range v.MatchSet {
		v.MatchSet[uri] = val
	}
}

func (v *View) Subscribe() chan *bw.SimpleMessage {
	return v.C
}

func (v *View) SubscribeCallback(fxn func(msg *bw.SimpleMessage)) error {
	go func() {
		for msg := range v.C {
			fxn(msg)
		}
	}()
	return nil
}

func (v *View) Publish(pos []bw.PayloadObject) error {
	return nil
}

//TODO: see what metadata changed?
func (v *View) OnChange(func(added, removed []string)) {
}

// if N is not nil, then evaluate it. Else, use P. Cannot have
// both defined. Either N or P are evaluated across all namespaces
// in NamespaceList
type Expression struct {
	NamespaceList []string
	N             Node
}

func (e *Expression) Hash() string {
	return e.N.Hash()
}

type Node interface {
	// Eval returns true if the record is matched by this node, and false otherwise
	Eval(rec *MetadataRecord) bool
	// returns a consistent representation of the node
	Hash() string
}

// Nil fields are ignored. For non-nil fields, this matches
// records whose fields are equal to those specified here
type EqualsNode struct {
	Key       *string
	Value     []byte
	Namespace *string
	URI       *string
	VK        *string
}

func (n *EqualsNode) Eval(rec *MetadataRecord) bool {
	var matches = true
	if n.Key == nil && n.Value == nil && n.Namespace == nil && n.URI == nil && n.VK == nil {
		return true
	}
	if n.Key != nil {
		matches = matches && (*n.Key == rec.Key)
	}
	if n.Value != nil {
		matches = matches && (byteSliceMatch(n.Value, rec.Value))
	}
	if n.Namespace != nil {
		matches = matches && (*n.Namespace == rec.Namespace)
	}
	if n.URI != nil {
		matches = matches && (*n.URI == rec.URI)
	}
	if n.VK != nil {
		matches = matches && (*n.VK == rec.VK)
	}
	return matches
}

func (n *EqualsNode) Hash() string {
	var h = "equals"

	if n.Key != nil {
		h += *n.Key
	}
	if n.Value != nil {
		h += string(n.Value)
	}
	if n.Namespace != nil {
		h += *n.Namespace
	}
	if n.URI != nil {
		h += *n.URI
	}
	if n.VK != nil {
		h += *n.VK
	}
	return h
}

type RegexNode struct {
	Key       *regexp.Regexp
	Value     *regexp.Regexp
	Namespace *regexp.Regexp
	URI       *regexp.Regexp
	VK        *regexp.Regexp
}

func (n *RegexNode) Eval(rec *MetadataRecord) bool {
	var matches = true
	if n.Key == nil && n.Value == nil && n.Namespace == nil && n.URI == nil && n.VK == nil {
		return true
	}
	if n.Key != nil {
		matches = matches && n.Key.MatchString(rec.Key)
	}
	if n.Value != nil {
		matches = matches && n.Value.Match(rec.Value)
	}
	if n.Namespace != nil {
		matches = matches && n.Namespace.MatchString(rec.Namespace)
	}
	if n.URI != nil {
		matches = matches && n.URI.MatchString(rec.URI)
	}
	if n.VK != nil {
		matches = matches && n.VK.MatchString(rec.VK)
	}
	return matches
}

func (n *RegexNode) Hash() string {
	var h = "regex"

	if n.Key != nil {
		h += n.Key.String()
	}
	if n.Value != nil {
		h += n.Value.String()
	}
	if n.Namespace != nil {
		h += n.Namespace.String()
	}
	if n.URI != nil {
		h += n.URI.String()
	}
	if n.VK != nil {
		h += n.VK.String()
	}
	return h
}

type AndNode struct {
	Left, Right Node
}

func (n *AndNode) Eval(rec *MetadataRecord) bool {
	return n.Left.Eval(rec) && n.Right.Eval(rec)
}

func (n *AndNode) Hash() string {
	return "and" + n.Left.Hash() + n.Right.Hash()
}

type OrNode struct {
	Left, Right Node
}

func (n *OrNode) Eval(rec *MetadataRecord) bool {
	return n.Left.Eval(rec) || n.Right.Eval(rec)
}

func (n *OrNode) Hash() string {
	return "or" + n.Left.Hash() + n.Right.Hash()
}
