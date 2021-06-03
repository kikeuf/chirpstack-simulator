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
    case "ack":
	err = h.ack(b)
    case "txack":
	err = h.txack(b)
    case "status":
	err = h.status(b)
    case "error":
	err = h.Xerror(b)
    default:
        //log.Printf("handler for event %s is not implemented", event)
	log.WithFields(log.Fields{
		"event": event,
	}).Warn("listener: handler for event is not implemented")
        return
    }

    if err != nil {
        //log.Printf("handling event '%s' returned error: %s", event, err)
	log.WithFields(log.Fields{
		"event": event,
		"error": err,
	}).Fatal("listener: handling event returned error")
    }
}

			
func (h *handler) up(b []byte) error {
    var up integration.UplinkEvent
    if err := h.unmarshal(b, &up); err != nil {
        return err
    }
    //log.Printf("Uplink received from %s with payload: %s", hex.EncodeToString(up.DevEui), decodedData(up.Data))
    log.WithFields(log.Fields{
	//"from_dev": hex.EncodeToString(up.DevEui),
	"application_name": up.ApplicationName,	
	"from_device": up.DeviceName,
	"payload": hex.EncodeToString(up.Data),
	"payload_decoded": iapi.DecodedData(up.Data),
    }).Info("listener: uplink data received")
    return nil
}

func (h *handler) status(b []byte) error {
    var status integration.StatusEvent
    if err := h.unmarshal(b, &status); err != nil {
        return err
    }
    log.WithFields(log.Fields{
	//"from_dev": hex.EncodeToString(status.DevEui),
	"application_name": status.ApplicationName,	
	"from_device": status.DeviceName,
	"battery_level": status.BatteryLevel,
    }).Info("listener: uplink data received")
    return nil
}


func (h *handler) join(b []byte) error {
    var join integration.JoinEvent
    if err := h.unmarshal(b, &join); err != nil {
        return err
    }
    //log.Printf("Device %s joined with DevAddr %s", hex.EncodeToString(join.DevEui), hex.EncodeToString(join.DevAddr))
    log.WithFields(log.Fields{
	//"DevEUI": hex.EncodeToString(join.DevEui),
	"application_name": join.ApplicationName,
	"device_name": join.DeviceName,
	"device_addr": hex.EncodeToString(join.DevAddr),
	//"RxInfo": join.RxInfo,
	//"TxInfo": join.TxInfo,
    }).Info("listener: device joined")
    return nil
}

func (h *handler) ack(b []byte) error {
    var ack integration.AckEvent 
    if err := h.unmarshal(b, &ack); err != nil {
        return err
    }
    log.WithFields(log.Fields{
	//"DevEUI": hex.EncodeToString(ack.DevEui),
	"application_name": ack.ApplicationName,
	"from_device": ack.DeviceName,
    }).Info("listener: acknowledgement received")
    return nil
}

func (h *handler) txack(b []byte) error {
    var txack integration.TxAckEvent 
    if err := h.unmarshal(b, &txack); err != nil {
        return err
    }
    log.WithFields(log.Fields{
	//"DevEUI": hex.EncodeToString(txack.DevEui),
	"application_name": txack.ApplicationName,
	"from_device": txack.DeviceName,
        //"GatewayId": hex.EncodeToString(txack.GatewayId),
    }).Info("listener: tx acknowledgement received")
    return nil
}

func (h *handler) Xerror(b []byte) error {
    var Xerror integration.ErrorEvent
    if err := h.unmarshal(b, &Xerror); err != nil {
        return err
    }
    log.WithFields(log.Fields{
	//"DevEUI": hex.EncodeToString(Xerror.DevEui),
	"application_name": Xerror.ApplicationName,		
	"from_device": Xerror.DeviceName,
	"error_type": Xerror.Type,
	"error_desc": Xerror.Error,
    }).Warn("listener: error message received")
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
