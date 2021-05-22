package listener

import (
    "bytes"
    "encoding/hex"
    "io/ioutil"
    "net/http"
    "strconv"
    log "github.com/sirupsen/logrus"

    "github.com/golang/protobuf/jsonpb"
    "github.com/golang/protobuf/proto"

    "github.com/brocaar/chirpstack-api/go/v3/as/integration"

    "github.com/kikeuf/chirpstack-simulator/internal/iapi"
)

type handler struct {
    json bool
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    b, err := ioutil.ReadAll(r.Body)
    if err != nil {
        panic(err)
    }

    event := r.URL.Query().Get("event")

    switch event {
    case "up":
        err = h.up(b)
    case "join":
        err = h.join(b)
    default:
        //log.Printf("handler for event %s is not implemented", event)
	log.WithFields(log.Fields{
		"event": event,
	}).Warn("handler for event is not implemented")
        return
    }

    if err != nil {
        //log.Printf("handling event '%s' returned error: %s", event, err)
	log.WithFields(log.Fields{
		"event": event,
		"error": err,
	}).Fatal("handling event returned error")
    }
}

			
func (h *handler) up(b []byte) error {
    var up integration.UplinkEvent
    if err := h.unmarshal(b, &up); err != nil {
        return err
    }
    //log.Printf("Uplink received from %s with payload: %s", hex.EncodeToString(up.DevEui), decodedData(up.Data))
    log.WithFields(log.Fields{
	"from_devEUI": hex.EncodeToString(up.DevEui),
	"payload": hex.EncodeToString(up.Data),
	"payload_decoded": iapi.DecodedData(up.Data),
    }).Info("uplink data received")
    return nil
}

func (h *handler) join(b []byte) error {
    var join integration.JoinEvent
    if err := h.unmarshal(b, &join); err != nil {
        return err
    }
    //log.Printf("Device %s joined with DevAddr %s", hex.EncodeToString(join.DevEui), hex.EncodeToString(join.DevAddr))
    log.WithFields(log.Fields{
	"DevEUI": hex.EncodeToString(join.DevEui),
	"DevAddr": hex.EncodeToString(join.DevAddr),
    }).Info("device joined")
    return nil
}

func (h *handler) unmarshal(b []byte, v proto.Message) error {
    if h.json {
        unmarshaler := &jsonpb.Unmarshaler{
            AllowUnknownFields: true, // we don't want to fail on unknown fields
        }
        return unmarshaler.Unmarshal(bytes.NewReader(b), v)
    }
    return proto.Unmarshal(b, v)
}

func Listen(port uint64, jsonFormat bool) {
    // json: false   - to handle Protobuf payloads (binary)
    // json: true    - to handle JSON payloads (Protobuf JSON mapping)
    http.Handle("/", &handler{json: jsonFormat})
    log.Fatal(http.ListenAndServe(":" + strconv.FormatUint(port,10), nil))
}
