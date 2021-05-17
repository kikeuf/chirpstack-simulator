package as

import (
	"context"
	"crypto/tls"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/brocaar/chirpstack-api/go/v3/as/external/api"
	"github.com/kikeuf/chirpstack-simulator/internal/config"
	"github.com/kikeuf/chirpstack-simulator/internal/iapi"
	
)

var clientConn *grpc.ClientConn
var mqttClient mqtt.Client

type jwtCredentials struct {
	token string
}

func (j *jwtCredentials) GetRequestMetadata(ctx context.Context, url ...string) (map[string]string, error) {
	return map[string]string{
		"authorization": j.token,
	}, nil
}

func (j *jwtCredentials) RequireTransportSecurity() bool {
	return false
}

// Setup configures the AS API client.
func Setup(c config.Config) error {
	
	conf := c.ApplicationServer

	token := iapi.GetToken(conf.API.Server,conf.API.User,conf.API.Password)
	if len(token)!=0 {
		conf.API.JWTToken = token
	}

	// connect gRPC
	log.WithFields(log.Fields{
		"server":   conf.API.Server,
		"insecure": conf.API.Insecure,
	}).Info("as: connecting api client")

	dialOpts := []grpc.DialOption{
		grpc.WithBlock(),
		grpc.WithPerRPCCredentials(&jwtCredentials{token: conf.API.JWTToken}),
	}

	if conf.API.Insecure {
		dialOpts = append(dialOpts, grpc.WithInsecure())
	} else {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})))
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, conf.API.Server, dialOpts...)
	if err != nil {
		return errors.Wrap(err, "grpc dial error")
	}

	clientConn = conn

	// connect MQTT
	opts := mqtt.NewClientOptions()
	opts.AddBroker(conf.Integration.MQTT.Server)
	opts.SetUsername(conf.Integration.MQTT.Username)
	opts.SetPassword(conf.Integration.MQTT.Password)
	opts.SetCleanSession(true)
	opts.SetAutoReconnect(true)

	log.WithFields(log.Fields{
		"server": conf.Integration.MQTT.Server,
	}).Info("as: connecting to mqtt broker")

	mqttClient = mqtt.NewClient(opts)
	if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
		return errors.Wrap(token.Error(), "mqtt client connect error")
	}

	return nil
}

func ServiceProfile() api.ServiceProfileServiceClient {
	return api.NewServiceProfileServiceClient(clientConn)
}

func Gateway() api.GatewayServiceClient {
	return api.NewGatewayServiceClient(clientConn)
}

func NetworkServer() api.NetworkServerServiceClient {
	return api.NewNetworkServerServiceClient(clientConn)
}

func DeviceProfile() api.DeviceProfileServiceClient {
	return api.NewDeviceProfileServiceClient(clientConn)
}

func Application() api.ApplicationServiceClient {
	return api.NewApplicationServiceClient(clientConn)
}

func Device() api.DeviceServiceClient {
	return api.NewDeviceServiceClient(clientConn)
}

// MQTTClient returns the MQTT client for the Application Server MQTT integration.
func MQTTClient() mqtt.Client {
	return mqttClient
}
