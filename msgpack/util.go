package msgpack

import (
	"bytes"
	"errors"
	"github.com/gtfierro/giles2/archiver"
	"gopkg.in/vmihailenco/msgpack.v2"
	"strings"
)

var KeyNotFound = errors.New("Key not found")
var KeyWasNotString = errors.New("Key wasn't a string")
var ReadingsNotFound = errors.New("Readings not found")
var MetadataNotFound = errors.New("Metadata not found")
var PropertiesNotFound = errors.New("Properties not found")

func getStringValue(msg map[string]interface{}, key string) (string, error) {
	if val, found := msg[key]; found {
		if strVal, ok := val.(string); ok {
			return strVal, nil
		} else {
			return "", KeyWasNotString
		}
	} else {
		return "", KeyNotFound
	}
}

func getReadings(msg map[string]interface{}) ([]archiver.Reading, error) {
	var (
		ret       []archiver.Reading
		val       interface{}
		readings  []interface{}
		found, ok bool
	)
	if val, found = msg["Readings"]; !found {
		return ret, ReadingsNotFound
	}

	if readings, ok = val.([]interface{}); !ok {
		return ret, ReadingsNotFound
	}

	for _, rdg := range readings {
		log.Debugf("got rdg %#v", rdg)
	}

	return ret, nil
}

func getMetadata(msg map[string]interface{}) (archiver.Dict, error) {
	var metadata archiver.Dict
	if md, found := msg["Metadata"]; found {
		if stringmap, ok := md.(map[string]interface{}); ok {
			// once we find it, copy it in, skipping non-strings
			//TODO: we will probably have to handle numbers here
			metadata = make(archiver.Dict, len(stringmap))
			for k, v := range stringmap {
				if vs, ok := v.(string); ok {
					k = strings.Replace(k, ".", "|", -1)
					k = strings.Replace(k, "/", "|", -1)
					metadata[k] = vs
				}
			}
			return metadata, nil
		} else {
			return metadata, MetadataNotFound
		}
	}
	return metadata, MetadataNotFound
}

func getProperties(msg map[string]interface{}) (properties *archiver.SmapProperties, err error) {
	properties = &archiver.SmapProperties{}
	if prop, found := msg["Properties"]; found {
		if propmap, ok := prop.(map[string]interface{}); ok {
			// UnitofTime
			if uot, found := propmap["UnitofTime"]; found {
				uotstr, ok := uot.(string)
				if !ok {
					err = errors.New("UnitofTime was not string")
				} else {
					properties.UnitOfTime, err = archiver.ParseUOT(uotstr)
				}
			}
			// UnitofMeasure
			if uom, found := propmap["UnitofMeasure"]; found {
				uomstr, ok := uom.(string)
				if !ok {
					err = errors.New("UnitofMeasure was not string")
				} else {
					properties.UnitOfMeasure = uomstr
				}
			}
			// StreamType
			if st, found := propmap["StreamType"]; found {
				ststr, ok := st.(string)
				if !ok || (ststr != "numeric" && ststr != "object") {
					err = errors.New("StreamType was not 'numeric' or 'object'")
				} else if ststr == "numeric" {
					properties.StreamType = archiver.NUMERIC_STREAM
				} else if ststr == "object" {
					properties.StreamType = archiver.OBJECT_STREAM
				}
			}
		}
		return properties, nil
	}
	return properties, PropertiesNotFound
}

func getValue(msg map[string]interface{}) (float64, error) {
	if val, found := msg["Value"]; found {
		if f64, ok := val.(float64); ok {
			return f64, nil
		} else if u64, ok := val.(uint64); ok {
			return float64(u64), nil
		} else if i64, ok := val.(int64); ok {
			return float64(i64), nil
		} else {
			return float64(0), ReadingsNotFound
		}
	}
	return float64(0), ReadingsNotFound
}

func doDecode(buffer []byte) (map[string]interface{}, error) {
	var msgMap map[string]interface{}
	decoder := msgpack.NewDecoder(bytes.NewBuffer(buffer))
	decoder.DecodeMapFunc = func(d *msgpack.Decoder) (interface{}, error) {
		n, err := d.DecodeMapLen()
		if err != nil {
			return nil, err
		}

		m := make(map[string]interface{}, n)
		for i := 0; i < n; i++ {
			mk, err := d.DecodeString()
			if err != nil {
				return nil, err
			}

			mv, err := d.DecodeInterface()
			if err != nil {
				return nil, err
			}

			m[mk] = mv
		}
		return m, nil
	}
	iface, err := decoder.DecodeInterface()
	if err != nil {
		return msgMap, err
	}

	msgMap, ok := iface.(map[string]interface{})
	if !ok {
		return msgMap, errors.New("Decoded packet wasn't map[string]interface{}")
	}

	return msgMap, nil
}
