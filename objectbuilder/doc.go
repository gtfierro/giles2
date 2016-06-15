//go:generate go tool yacc -o expr.go -p ex expr.y
package objectbuilder

const NAMESPACE_UUID = "b26d2e62-333e-11e6-b557-0cc47a0f7eea"

// TODO: need a PO type for each of the main serialization methods: JSON,
// MsgPack, YAML, CapnProto

// This object is a set of instructions for how to create an archivable message
// from some received PayloadObject, though really this should be able to
// operate on any object. Each ObjectTemplate acts as a translator for received
// messages into a single timeseries stream
type ObjectTemplate struct {
	// AUTOPOPULATED. The entity that requested the URI to be archived.
	FromVK string
	// OPTIONAL. the URI to subscribe to. Requires building a chain on the URI
	// from the .FromVK field. If not provided, uses the base URI of where this
	// ObjectTemplate was stored. For example, if this template was published
	// on <uri>/!meta/giles, then if the URI field was elided it would default
	// to <uri>.
	URI string
	// Extracts objects of the given Payload Object type from all messages
	// published on the URI. If elided, operates on all PO types.
	PO int
	// OPTIONAL. If provided, this is used as the stream UUID.  If not
	// provided, then a UUIDv3 with NAMESPACE_UUID and the URI, PO type and
	// Value are used.
	UUID string
	// expression determining how to extract the value from the received
	// message
	Value string
	// OPTIONAL. Expression determining how to extract the value from the
	// received message. If not included, it uses the time the message was
	// received on the server.
	Time string
	// OPTIONAL. Golang time parse string
	TimeParse string
}

/*

 */

//   - URI (optional): the URI to subscribe to for data to be archived. If not specified, subscribes to the {uri} part of {uri}/!meta/archive
//		TODO: build chain on this URI to the person who published the metadata to tell if they are allowed to archive the URI
//   - UUID (optional): the UUID to use for the incoming data. If not specified, does a UUIDv3 on a namespace uuid and the URI that published the data. Do we want to
//   - value: the key (if string) or index (if number) into the message published on this URI that should be archived. This can be a list of keys or a list of indices
//   - time: key or index of the timestamp field. If ignored, it will use the time the message was received at the archiver.
//   - time parse: the go-style string specifying the time format
//   - keep data: duration for how long data should be archived for (want some sort of method here)
//
// URI and UUID are easy enough, but we want a more general solution for a generic selector into arbitrary(ish) objects.
// 	 - /key: retrieve value in base struct/map
//	 - /key1/key2: retrieve key2 from the struct/map in indexed by key1
//	 - [3]: retrieve element at index 3 into the array
//	 - /key[1]: retrieve element at index 1 in the array stored under key1 in the struct/map
//	 - [0][3]: retrieve the 4th element in the list stored in the first element of the root list
//	 - /key/@pairs [time, val]
//
// implement decoding for all the PO types, using the masks
