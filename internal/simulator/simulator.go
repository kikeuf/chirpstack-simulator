
package simulator

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	mrand "math/rand"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
	"strings"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/brocaar/chirpstack-api/go/v3/as/external/api"
	"github.com/brocaar/chirpstack-api/go/v3/common"
	"github.com/brocaar/chirpstack-api/go/v3/gw"
	"github.com/kikeuf/chirpstack-simulator/internal/as"
	"github.com/kikeuf/chirpstack-simulator/internal/config"
	"github.com/kikeuf/chirpstack-simulator/internal/ns"
	"github.com/kikeuf/chirpstack-simulator/internal/iapi"
	"github.com/kikeuf/chirpstack-simulator/simulator"
	"github.com/brocaar/lorawan"
)

var server string
var apitoken string

// Start starts the simulator.
func Start(ctx context.Context, wg *sync.WaitGroup, c config.Config) error {


	server = c.ApplicationServer.API.Server
	apitoken = iapi.Token //c.ApplicationServer.API.JWTToken

	for i, c := range c.Simulator {
		log.WithFields(log.Fields{
			"i": i+1,
		}).Info("simulator: starting simulation")

		
		wg.Add(1)
		

		spID, err := uuid.FromString(c.ServiceProfileID)
		if err != nil {
			return errors.Wrap(err, "uuid from string error")
		}

						
		var sdevs []simdevice

		for _, cd := range c.Device {

			
			if len(cd.Payload)==0 {
				cd.Payload = hex.EncodeToString([]byte(cd.StrPayload))
			}

			pl, err := hex.DecodeString(cd.Payload)
			if err != nil {
				return errors.Wrap(err, "decode uplink payload error")
			}

			if len(cd.DownlinkPayload)==0 {
				cd.DownlinkPayload = hex.EncodeToString([]byte(cd.DownlinkStrPayload))
			}

			var dpl []byte
			if (len(cd.DownlinkPayload)!=0) {
				dpl, err = hex.DecodeString(cd.DownlinkPayload)
				if err != nil {
					return errors.Wrap(err, "decode downlink payload error")
				}
			}
			
			var gws []string
			gws = strings.Split(cd.Gateways, ",")
			for id,g := range gws {
				gws[id]=strings.TrimSpace(g)
			}

			if cd.Count < 1 { cd.Count = 1}

			sdev := simdevice {
				prefix :	      cd.Prefix,				
				deviceCount:          cd.Count,				
				uplinkInterval:       cd.UplinkInterval,
				fPort:                cd.FPort,
				payload:              pl,
				frequency:            cd.Frequency,
				bandwidth:            cd.Bandwidth / 1000,
				spreadingFactor:      cd.SpreadingFactor,
				AppKeys:	      make(map[lorawan.EUI64]lorawan.AES128Key),
				gateways :	      gws,				
				downlinkActivate:     cd.DownlinkActivate,				
				downlinkInterval:     cd.DownlinkInterval,
				downlinkPayload:      dpl,
				}
			sdevs = append(sdevs, sdev)

		}

		var sgws []simgateway
		for _, cg := range c.Gateway {

			if cg.Count < 1 { cg.Count = 1} 
			
			sgw := simgateway {
			    groupId :		  cg.ID,
			    gatewayCount : 	  cg.Count,
			    prefix :		  cg.Prefix,  
			    eventTopicTemplate:   cg.EventTopicTemplate,
			    commandTopicTemplate: cg.CommandTopicTemplate,
			}
		   sgws = append(sgws, sgw)
		}

		sim := simulation{
			num :		      i+1,
			prefix : 	      c.Prefix,			
			ctx:                  ctx,
			wg:                   wg,
			serviceProfileID:     spID,
			activationTime:       c.ActivationTime,
			devices:	      sdevs,
			gateways:	      sgws,
			duration:             c.Duration,
			deviceAppKeys:        make(map[lorawan.EUI64]lorawan.AES128Key),
                        applicationID:        c.ApplicationID,  
			TearApplication:      (c.ApplicationID==0),
		}

		go sim.start()
		
	}

	return nil
}

type simulation struct {
	num		 int
	prefix		 string	
	ctx              context.Context
	wg               *sync.WaitGroup
	serviceProfileID uuid.UUID
	duration         time.Duration
	activationTime   time.Duration

	devices		[]simdevice	
	gateways	[]simgateway

	serviceProfile       *api.ServiceProfile
	deviceProfileID      uuid.UUID
	applicationID        int64
	applicationName	     string
	gatewayIDs           []lorawan.EUI64
	deviceAppKeys        map[lorawan.EUI64]lorawan.AES128Key
	TearApplication	     bool
}

type simdevice struct {
	prefix		   string
	deviceCount        int
	fPort              uint8
	payload            []byte
	strpayload	   string
	uplinkInterval     time.Duration
	frequency          int
	bandwidth          int
	spreadingFactor    int
	AppKeys		   map[lorawan.EUI64]lorawan.AES128Key
	gateways	   []string
	downlinkActivate   bool
	downlinkInterval   time.Duration
	downlinkPayload    []byte

}

type simgateway struct {
	groupId		string
	gatewayCount	int	
	prefix		string
	eventTopicTemplate   string
	commandTopicTemplate string
	gatewayIDs           []lorawan.EUI64
}

type DownlinkDevices struct {
	Active		bool	
	FPort		uint8
	Payload		[]byte
	Confirmed 	bool
	Interval 	time.Duration
	devEUI		[]lorawan.EUI64
	//devName	[]string
	prefix		string
}
	

func (s *simulation) start() {
	if err := s.init(); err != nil {
		log.WithError(err).Error("simulator: init simulation error")
	}
	if err := s.runSimulation(); err != nil {
		log.WithError(err).Error("simulator: simulation error")
	}

	log.WithFields(log.Fields{"simulation": s.num, }).Info("simulator: simulation completed")

	if err := s.tearDown(); err != nil {
		log.WithError(err).Error("simulator: tear-down simulation error")
	}

	s.wg.Done()

	log.WithFields(log.Fields{"simulation": s.num, }).Info("simulation: tear-down completed")
	
}

func (s *simulation) init() error {
	log.WithFields(log.Fields{"simulation": s.num, }).Info("simulation: setting up")

	if err := s.setupServiceProfile(); err != nil {
		return err
	}

	if err := s.setupGateways(); err != nil {
		return err
	}

	if err := s.setupDeviceProfile(); err != nil {
		return err
	}

	if err := s.setupApplication(); err != nil {
		return err
	}

	if err := s.setupDevices(); err != nil {
		return err
	}

	if err := s.setupApplicationIntegration(); err != nil {
		return err
	}

	return nil
}

func (s *simulation) tearDown() error {
	log.WithFields(log.Fields{"simulation": s.num, }).Info("simulation: cleaning up")

	if err := s.tearDownApplicationIntegration(); err != nil {
		return err
	}

	if err := s.tearDownDevices(); err != nil {
		return err
	}

	if err := s.tearDownApplication(); err != nil {
		return err
	}

	if err := s.tearDownDeviceProfile(); err != nil {
		return err
	}

	if err := s.tearDownGateways(); err != nil {
		return err
	}

	return nil
}

func (s *simulation) runSimulation() error {
	var gateways []*simulator.Gateway
	var devices []*simulator.Device
	var gateway_groupids []string
	var downDevices []DownlinkDevices

	//Create all gateways for this simulation
	for _, g := range s.gateways {	

	    for _, gatewayID := range g.gatewayIDs {
		gw, err := simulator.NewGateway(
			simulator.WithGatewayID(gatewayID),
			simulator.WithGatewayName(g.prefix + gatewayID.String()),
			simulator.WithMQTTClient(ns.Client()),
			simulator.WithEventTopicTemplate(g.eventTopicTemplate),
			simulator.WithCommandTopicTemplate(g.commandTopicTemplate),
		)
		if err != nil {
			return errors.Wrap(err, "new gateway error")
		}
		gateways = append(gateways, gw)
		gateway_groupids = append(gateway_groupids,g.groupId)
	   }	
	}

	//Manage cancellation
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(s.ctx)
	if s.duration != 0 {
		ctx, cancel = context.WithTimeout(ctx, s.duration)
	}
	defer cancel()
	
	for _, sdev := range s.devices {

		//Create a group of gateways used by all devices of this group
		var gws []*simulator.Gateway
		for k := range gateways {
		    for _, gid := range sdev.gateways {
			if gid == gateway_groupids[k] {
				gws = append(gws, gateways[k])
			}
		   }
		}

		var downDev DownlinkDevices
		
		ks := 0
		for devEUI, appKey := range sdev.AppKeys {

			d, err := simulator.NewDevice(ctx, &wg,
				simulator.WithDevEUI(devEUI),
				simulator.WithDevName(sdev.prefix + devEUI.String()),
				simulator.WithAppKey(appKey),
				simulator.WithUplinkInterval(sdev.uplinkInterval),
				simulator.WithOTAADelay(time.Duration(mrand.Int63n(int64(s.activationTime)))),
				simulator.WithUplinkPayload(false, sdev.fPort, sdev.payload),
				//simulator.WithDownlinkPayload(sdev.downlinkActivate,false,sdev.downlinkInterval,sdev.downlinkPayload),
				simulator.WithGateways(gws),
				simulator.WithUplinkTXInfo(gw.UplinkTXInfo{
					Frequency:  uint32(sdev.frequency),
					Modulation: common.Modulation_LORA,
					ModulationInfo: &gw.UplinkTXInfo_LoraModulationInfo{
						LoraModulationInfo: &gw.LoRaModulationInfo{
							Bandwidth:       uint32(sdev.bandwidth),
							SpreadingFactor: uint32(sdev.spreadingFactor),
							CodeRate:        "3/4",
						},
					},
				}),
			)
			if err != nil {
				return errors.Wrap(err, "new device error")
			}

			devices = append(devices, d)

			if ks==0 {
				downDev.Active = sdev.downlinkActivate	
				downDev.FPort = sdev.fPort
				downDev.Payload = sdev.downlinkPayload
				downDev.Confirmed = false
				downDev.Interval = sdev.downlinkInterval
				downDev.prefix = sdev.prefix
			}
			downDev.devEUI = append(downDev.devEUI, devEUI)
			ks++
			
		}
		downDevices = append(downDevices, downDev)

	}
	
	for k:=0; k<len(downDevices);k++ {
		var d DownlinkDevices
		d =  downDevices[k]
		if d.Active { iapi.SendDownlinkLoop(s.activationTime, s.duration,server,apitoken,d.Interval,d.devEUI,d.prefix,d.FPort,d.Payload,d.Confirmed) }
	}

	go func() {
		sigChan := make(chan os.Signal)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

		select {
		case sig := <-sigChan:
			log.WithField("signal", sig).Info("signal received, stopping simulators")
			cancel()
		case <-ctx.Done():
		}
	}()

	wg.Wait()

	return nil
}

func (s *simulation) setupServiceProfile() error {
	log.WithFields(log.Fields{
		"simulation": s.num,
		"service_profile_id": s.serviceProfileID,
	}).Info("simulator: retrieving service-profile")
	sp, err := as.ServiceProfile().Get(context.Background(), &api.GetServiceProfileRequest{
		Id: s.serviceProfileID.String(),
	})
	if err != nil {
		return errors.Wrap(err, "get service-profile error")
	}
	s.serviceProfile = sp.ServiceProfile

	return nil
}

func (s *simulation) setupGateways() error {
	log.WithFields(log.Fields{"simulation": s.num, }).Info("simulator: creating gateways")

    for k, g := range s.gateways {

	for i := 0; i < g.gatewayCount; i++ {
		var gatewayID lorawan.EUI64
		if _, err := rand.Read(gatewayID[:]); err != nil {
			return errors.Wrap(err, "read random bytes error")
		}

		_, err := as.Gateway().Create(context.Background(), &api.CreateGatewayRequest{
			Gateway: &api.Gateway{
				Id:              gatewayID.String(),
				Name:            g.prefix + gatewayID.String(),
				Description:     g.prefix + gatewayID.String(),
				OrganizationId:  s.serviceProfile.OrganizationId,
				NetworkServerId: s.serviceProfile.NetworkServerId,
				Location:        &common.Location{},
			},
		})
		if err != nil {
			return errors.Wrap(err, "create gateway error")
		}
		
		s.gatewayIDs = append(s.gatewayIDs, gatewayID)
		//g.gatewayIDs = append(g.gatewayIDs, gatewayID)
		s.gateways[k].gatewayIDs = append(s.gateways[k].gatewayIDs, gatewayID)

	    }
	}

	return nil
}

func (s *simulation) tearDownGateways() error {
	log.WithFields(log.Fields{"simulation": s.num, }).Info("simulator: tear-down gateways")

	for _, gatewayID := range s.gatewayIDs {
		_, err := as.Gateway().Delete(context.Background(), &api.DeleteGatewayRequest{
			Id: gatewayID.String(),
		})
		if err != nil {
			return errors.Wrap(err, "delete gateway error")
		}
	}

	return nil
}

func (s *simulation) setupDeviceProfile() error {
	log.WithFields(log.Fields{"simulation": s.num, }).Info("simulator: creating device-profile")

	dpName, _ := uuid.NewV4()

	resp, err := as.DeviceProfile().Create(context.Background(), &api.CreateDeviceProfileRequest{
		DeviceProfile: &api.DeviceProfile{
			Name:              dpName.String(),
			OrganizationId:    s.serviceProfile.OrganizationId,
			NetworkServerId:   s.serviceProfile.NetworkServerId,
			MacVersion:        "1.0.3",
			RegParamsRevision: "B",
			SupportsJoin:      true,
		},
	})
	if err != nil {
		return errors.Wrap(err, "create device-profile error")
	}

	dpID, err := uuid.FromString(resp.Id)
	if err != nil {
		return err
	}
	s.deviceProfileID = dpID

	return nil
}

func (s *simulation) tearDownDeviceProfile() error {
	log.WithFields(log.Fields{"simulation": s.num, }).Info("simulator: tear-down device-profile")

	_, err := as.DeviceProfile().Delete(context.Background(), &api.DeleteDeviceProfileRequest{
		Id: s.deviceProfileID.String(),
	})
	if err != nil {
		return errors.Wrap(err, "delete device-profile error")
	}

	return nil
}

func (s *simulation) setupApplication() error {
	log.WithFields(log.Fields{"simulation": s.num, }).Info("simulator: init application")

	if s.applicationID == 0 {
		
		appName, err := uuid.NewV4()
		if err != nil {
			return err
		}
		createAppResp, err := as.Application().Create(context.Background(), &api.CreateApplicationRequest{
			Application: &api.Application{
				Name:             s.prefix + appName.String(),
				Description:      s.prefix + appName.String(),
				OrganizationId:   s.serviceProfile.OrganizationId,
				ServiceProfileId: s.serviceProfile.Id,
			},
		})
			//log.Info("simulator: init application - after create")

		if err != nil {
			return errors.Wrap(err, "create application error")
		}
		s.applicationName = s.prefix + appName.String()
		s.applicationID = createAppResp.Id
	}
	return nil
}


func (s *simulation) tearDownApplication() error {
	log.WithFields(log.Fields{"simulation": s.num, }).Info("simulator: tear-down application")

	if s.TearApplication {

		_, err := as.Application().Delete(context.Background(), &api.DeleteApplicationRequest{
			Id: s.applicationID,
		})
		if err != nil {
			return errors.Wrap(err, "delete application error")
		}
	}
	return nil
}

/*
func (s *simulation) setupDevicesGroup() error {
	log.WithFields(log.Fields{"simulation": s.num, }).Info("simulator: init devices group")

	
}
*/

func (s *simulation) setupDevices() error {
	log.WithFields(log.Fields{"simulation": s.num, }).Info("simulator: init devices")

    for _, d := range s.devices {

	for i := 0; i < d.deviceCount; i++ {
		var devEUI lorawan.EUI64
		var appKey lorawan.AES128Key

		if _, err := rand.Read(devEUI[:]); err != nil {
			return err
		}
		if _, err := rand.Read(appKey[:]); err != nil {
			return err
		}

		_, err := as.Device().Create(context.Background(), &api.CreateDeviceRequest{
			Device: &api.Device{
				DevEui:          devEUI.String(),
				Name:            d.prefix + devEUI.String(),
				Description:     d.prefix + devEUI.String(),
				ApplicationId:   s.applicationID,
				DeviceProfileId: s.deviceProfileID.String(),
			},
		})
		if err != nil {
			return errors.Wrap(err, "create device error")
		}

		_, err = as.Device().CreateKeys(context.Background(), &api.CreateDeviceKeysRequest{
			DeviceKeys: &api.DeviceKeys{
				DevEui: devEUI.String(),

				// yes, this is correct for LoRaWAN 1.0.x!
				// see the API documentation
				NwkKey: appKey.String(),
			},
		})
		if err != nil {
			return errors.Wrap(err, "create device keys error")
		}

		s.deviceAppKeys[devEUI] = appKey
		d.AppKeys[devEUI] = appKey
	}
    }

	return nil
}

func (s *simulation) tearDownDevices() error {
	log.WithFields(log.Fields{"simulation": s.num, }).Info("simulator: tear-down devices")

	for k := range s.deviceAppKeys {
		_, err := as.Device().Delete(context.Background(), &api.DeleteDeviceRequest{
			DevEui: k.String(),
		})
		if err != nil {
			return errors.Wrap(err, "delete device error")
		}
	}

	return nil
}

func (s *simulation) setupApplicationIntegration() error {
	log.WithFields(log.Fields{"simulation": s.num, }).Info("simulator: setting up application integration")

	token := as.MQTTClient().Subscribe(fmt.Sprintf("application/%d/device/+/rx", s.applicationID), 0, func(client mqtt.Client, msg mqtt.Message) {
		applicationUplinkCounter().Inc()
	})
	token.Wait()
	if token.Error() != nil {
		return errors.Wrap(token.Error(), "subscribe application integration error")
	}

	return nil
}

func (s *simulation) tearDownApplicationIntegration() error {
	log.WithFields(log.Fields{"simulation": s.num, }).Info("simulator: tear-down application integration")

	token := as.MQTTClient().Unsubscribe(fmt.Sprintf("application/%d/device/+/rx", s.applicationID))
	token.Wait()
	if token.Error() != nil {
		return errors.Wrap(token.Error(), "unsubscribe application integration error")
	}

	return nil
}
