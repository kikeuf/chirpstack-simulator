package iapi

import (
    "context"
    "fmt"
    "time"
    "encoding/hex"
    log "github.com/sirupsen/logrus"


    "google.golang.org/grpc"
    "github.com/pkg/errors"

    "github.com/brocaar/chirpstack-api/go/v3/as/external/api"
    "github.com/brocaar/lorawan"

    //"github.com/kikeuf/chirpstack-simulator/internal/iapi"

)


type APIToken string


func (a APIToken) GetRequestMetadata(ctx context.Context, url ...string) (map[string]string, error) {
    return map[string]string{
        "authorization": fmt.Sprintf("Bearer %s", a),
    }, nil
}

func (a APIToken) RequireTransportSecurity() bool {
    return false
}

//func SendDownlink(server string, apitoken string, downDevices DownlinkDevices) {
func SendDownlink(server string, apitoken string, devEUI []lorawan.EUI64, prefix string, fport uint8, data []byte, confirmed bool) {

//log.Println("senddownlink")
     if len(server)==0 { server="localhost:8080" } 

     // define gRPC dial options
     dialOpts := []grpc.DialOption{
        grpc.WithBlock(),
        grpc.WithPerRPCCredentials(APIToken(apitoken)),
        grpc.WithInsecure(), // remove this when using TLS
    }

    // connect to the gRPC server
    conn, err := grpc.Dial(server, dialOpts...)
    if err != nil {
	msg := errors.Wrap(err, "simulator: downlink to devices group has failed")
	//log.Println(msg)
	log.Warn(msg)

    }

    // define the DeviceQueueService client
    queueClient := api.NewDeviceQueueServiceClient(conn)

    for k:=0;k < len(devEUI);k++ {
	    // make an Enqueue api call
	    resp, err := queueClient.Enqueue(context.Background(), &api.EnqueueDeviceQueueItemRequest{
		DeviceQueueItem: &api.DeviceQueueItem{
		    DevEui:    devEUI[k].String(), 
		    FPort:     uint32(fport), 
		    Confirmed: confirmed, 
		    Data:      data, 
		},
	    })
	    if err != nil {
		log.WithFields(log.Fields{
		    //"dev_eui": devEUI[k].String(),
		    "to_device": prefix + devEUI[k].String(),
 		}).Warn("simulator: downlink has failed")
	    } else {
		log.WithFields(log.Fields{
		    "confirmed": confirmed,
		    "data": hex.EncodeToString(data),
		    "data_decoded": DecodedData(data),
		    //"dev_eui": devEUI[k].String(),
		    "to_device": prefix + devEUI[k].String(),
		    "f_cnt": resp.FCnt,
		    "f_port": fport,
 		}).Info("simulator: downlink data has been enqueud")
	    }
    }

}

func SendDownlinkLoop(waitduration time.Duration, sduration time.Duration, server string, apitoken string, interval time.Duration, devEUI []lorawan.EUI64, prefix string, fport uint8, data []byte, confirmed bool) {

	done := make(chan bool)	
	go func() {
		time.Sleep(sduration-interval)
		done <- true
	}()

	//Wait the time for devices to be active
	time.Sleep(waitduration + (5 * time.Second))

	ticker := time.NewTicker(interval)
	go func() {
	    for {
		
		//time.Sleep(interval)
	        select {
		case <- ticker.C:
		    SendDownlink(server,apitoken,devEUI, prefix, fport,data,confirmed)
		case <-done:
		   ticker.Stop()
		   return
		}
	    }
 	}()
}

