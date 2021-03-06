package cmd

import (
	"os"
	"text/template"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/kikeuf/chirpstack-simulator/internal/config"
)

const configTemplate = `[general]
# Log level
#
# debug=5, info=4, warning=3, error=2, fatal=1, panic=0
log_level={{ .General.LogLevel }}

# Listen Http integration
[listener]

  # Activate listener
  activate=true

  # Listening port
  # Add http integration in applications to see the uplink and join messages
  port=8090

  #json=false   - to handle Protobuf payloads (binary)
  #json=true    - to handle JSON payloads (Protobuf JSON mapping)
  json=true

# Application Server configuration.
[application_server]

  # API configuration.
  #
  # This configuration is used to automatically create the:
  #   * Device profile
  #   * Gateways
  #   * Application
  #   * Devices
  [application_server.api]

  # JWT token.
  #
  # The JWT token to connect to the ChirpStack Application Server API. This
  # token can be generated using the login API endpoint. In the near-future
  # it will be possible to generate these tokens within the web-interface:
  # https://github.com/brocaar/chirpstack-application-server/pull/421
  # A token is valid only for 48 hours and need to be refreshed
  jwt_token="{{ .ApplicationServer.API.JWTToken }}"

  # Credentials to use for http requests on chirpstack api
  # Chirpstack-simulator will try to retrieve the jwt_token by this way to avoid manual refresh
  # if it doesn't manage the jwt_token specified above will be used instead 
  username="{{ .ApplicationServer.API.User }}"
  password="{{ .ApplicationServer.API.Password }}"

  # Server.
  #
  # This must point to the external API server of the ChirpStack Application
  # Server. When the server is running on the same machine, keep this to the
  # default value.
  server="{{ .ApplicationServer.API.Server }}"

  # Insecure.
  #
  # Set this to true when the endpoint is not using TLS.
  insecure={{ .ApplicationServer.API.Insecure }}


  # MQTT integration configuration.
  #
  # This integration is used for counting the number of uplinks that are
  # published by the ChirpStack Application Server integration.
  [application_server.integration.mqtt]

  # MQTT server.
  server="{{ .ApplicationServer.Integration.MQTT.Server }}"

  # Username.
  username="{{ .ApplicationServer.Integration.MQTT.Username }}"

  # Password.
  password="{{ .ApplicationServer.Integration.MQTT.Password }}"


# Network Server configuration.
#
# This configuration is used to simulate LoRa gateways using the MQTT gateway
# backend.
[network_server]

  # MQTT gateway backend.
  [network_server.gateway.backend.mqtt]

  # MQTT server.
  server="{{ .NetworkServer.Gateway.Backend.MQTT.Server }}"

  # Username.
  username="{{ .NetworkServer.Gateway.Backend.MQTT.Username }}"

  # Password.
  password="{{ .NetworkServer.Gateway.Backend.MQTT.Password }}"


# Simulator configuration (this section can be replicated)
# [[simulator]]
#
# # Application ID (facultative)
#
# # Set the application ID to use for this simulation
# # If not specified a new dedicated application will be created
# application_id=1
#
# Prefix (facultative)
# 
# Prefix for this simulation name
# # This parameter will be ignored if application_id is specified
# prefix="simulation1_"
#
# # Service-profile ID.
# #
# # It is recommended to create a new organization with a new service-profile
# # in the ChirpStack Application Server.
# service_profile_id="1f32476e-a112-4f00-bcc7-4aab4bfefa1d"
#
# # Duration.
# #
# # This defines the duration of the simulation. If set to '0s', the simulation
# # will run until terminated. This includes the activation time.
# duration="5m"
#
# # Activation time.
# #
# # This is the time that the simulator takes to activate the devices. This
# # value must be less than the simulator duration.
# activation_time="1m"
#
#
#   # Device configuration 
#   # this section can be replicated to simulate more devices with other parameters
#   [[simulator.device]]
#
#   # Prefix for this device name (facultative)
#   prefix="devicegroup1_"
#
#   # Number of devices to simulate 
#   # The device can be cloned with exactly same parameters as many times you specify here
#   count=1000
#
#   # Uplink interval.
#   uplink_interval="1m"
#
#   # FPort.
#   f_port=10
#
#   # Payload (HEX encoded).
#   payload="010203"
#
#   # StrPayload (String) 
#   # used only if 'payload' value is empty or not set otherwise this parameter is ignored
#   strpayload="text"
#
#   # Downlink activation.
#   downlink_activate=true
#
#   # Downlink Payload (HEX encoded).
#   downlink_payload="010203"
#
#   # Downlink Payload (String).
#   # Used only if 'downlink_payload' value is empty or not set otherwise this parameter is ignored
#   downlink_strpayload="ok"
#
#   # Downlink interval.
#   downlink_interval="2m"
#
#   # Frequency (Hz).
#   frequency=868100000
#
#   # Bandwidth (Hz).
#   bandwidth=125000
#
#   # Spreading-factor.
#   spreading_factor=7
#
#   # Ids of group of gateways connected with all devices of this group
#   # Ids separated by commas
#   gateways="1,2"
#
#
#   # Gateway configuration 
#   # This section can be replicated to simulate more gateways with other parameters
#   [[simulator.gateway]]
#
#   # Id of the group of gateways
#   # Id to add in 'gateways' fields under [simulator.device] section
#   group_id="1"
#
#   # Prefix for this devices group name (facultative)
#   prefix="gatewaygroup1_"
#
#   # Number of receiving gateways 
#   # The gateway can be cloned with exactly same parameters as many times you specify here
#   count=3
#
#   # Event topic template.
#   event_topic_template="{{ "gateway/{{ .GatewayID }}/event/{{ .Event }}" }}"
#
#   # Command topic template.
#   command_topic_template="{{ "gateway/{{ .GatewayID }}/command/{{ .Command }}" }}"

{{ range $index, $element := .Simulator }}
[[simulator]]
prefix="{{ $element.Prefix }}"
service_profile_id="{{ $element.ServiceProfileID }}"
duration="{{ $element.Duration }}"
activation_time="{{ $element.ActivationTime }}"

  [[simulator.device]]
  prefix="{{ $element.Device.Prefix }}"
  count={{ $element.Device.Count }}
  uplink_interval="{{ $element.Device.UplinkInterval }}"
  f_port="{{ $element.Device.FPort }}"
  payload="{{ $element.Device.Payload }}"
  strpayload="{{ $element.Device.StrPayload }}"
  downlink_activate="{{ $element.Device.DownlinkActivate }}"
  downlink_interval="{{ $element.Device.DownlinkInterval }}"
  downlink_payload="{{ $element.Device.DownlinkPayload }}"
  downlink_strpayload="{{ $element.Device.DownlinkStrPayload }}" 
  frequency={{ $element.Device.Frequency }}
  bandwidth={{ $element.Device.Bandwidth }}
  spreading_factor={{ $element.Device.SpreadingFactor }}
  gateways="{{ $element.Device.Gateways }}"

  [[simulator.gateway]]
  groupid"{{ $element.Gateway.GroupId }}"
  prefix"{{ $element.Gateway.Prefix }}"
  count={{ $element.Gateway.Count }}
  event_topic_template="{{ $element.Gateway.EventTopicTemplate }}"
  command_topic_template="{{ $element.Gateway.CommandTopicTemplate }}"
{{ end }}

# Prometheus metrics configuration.
#
# Using Prometheus (and Grafana), it is possible to visualize various
# simulation metrics like:
#   * Join-Requests sent
#   * Join-Accepts received
#   * Uplinks sent (by the devices)
#   * Uplinks sent (by the gateways)
#   * Uplinks sent (by the ChirpStack Application Server MQTT integration)
[prometheus]

# IP:port to bind the Prometheus endpoint to.
#
# Metrics can be retrieved from /metrics.
bind="{{ .Prometheus.Bind }}"
`

var configCmd = &cobra.Command{
	Use:   "configfile",
	Short: "Print the ChirpStack Network Server configuration file",
	RunE: func(cmd *cobra.Command, args []string) error {
		t := template.Must(template.New("config").Parse(configTemplate))
		err := t.Execute(os.Stdout, &config.C)
		if err != nil {
			return errors.Wrap(err, "execute config template error")
		}
		return nil
	},
}
