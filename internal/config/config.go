package config

import (
	"time"
)

// Version defines the version.
var Version string

// Config defines the configuration.
type Config struct {
	General struct {
		LogLevel int `mapstructure:"log_level"`
	}

	ApplicationServer struct {
		API struct {
			JWTToken string `mapstructure:"jwt_token"`
			Server   string `mapstructure:"server"`
			Insecure bool   `mapstructure:"insecure"`
		} `mapstructure:"api"`

		Integration struct {
			MQTT struct {
				Server   string `mapstructure:"server"`
				Username string `mapstructure:"username"`
				Password string `mapstructure:"password"`
			} `mapstructure:"mqtt"`
		} `mapstructure:"integration"`
	} `mapstructure:"application_server"`

	NetworkServer struct {
		Gateway struct {
			Backend struct {
				MQTT struct {
					Server   string `mapstructure:"server"`
					Username string `mapstructure:"username"`
					Password string `mapstructure:"password"`
				} `mapstructure:"mqtt"`
			} `mapstructure:"backend"`
		} `mapstructure:"gateway"`
	} `mapstructure:"network_server"`

	Simulator []struct {
		Prefix		 string        `mapstructure:"prefix"`
		ServiceProfileID string        `mapstructure:"service_profile_id"`
		Duration         time.Duration `mapstructure:"duration"`
		ActivationTime   time.Duration `mapstructure:"activation_time"`

		Device[] struct {
			Prefix		string	      `mapstructure:"prefix"`	
			Count           int           `mapstructure:"count"`
			UplinkInterval  time.Duration `mapstructure:"uplink_interval"`
			FPort           uint8         `mapstructure:"f_port"`
			StrPayload      string        `mapstructure:"strpayload"`
			Payload         string        `mapstructure:"payload"`
			Frequency       int           `mapstructure:"frequency"`
			Bandwidth       int           `mapstructure:"bandwidth"`
			SpreadingFactor int           `mapstructure:"spreading_factor"`
			Gateways	string        `mapstructure:"gateways"`

		} `mapstructure:"device"`

		Gateway[] struct {
			ID		     string `mapstructure:"group_id"`	
			Prefix		     string `mapstructure:"prefix"`							
			Count		     int    `mapstructure:"count"`			
			EventTopicTemplate   string `mapstructure:"event_topic_template"`
			CommandTopicTemplate string `mapstructure:"command_topic_template"`
		} `mapstructure:"gateway"`
	} `mapstructure:"simulator"`

	Prometheus struct {
		Bind string `mapstructure:"bind"`
	} `mapstructure:"prometheus"`
}

type DeviceConfig struct {
	DevEUI         string        `mapstructure:"dev_eui"`
	AppKey         string        `mapstructure:"app_key"`
	UplinkInterval time.Duration `mapstructure:"uplink_interval"`
	FPort          uint8         `mapstructure:"f_port"`
	Payload        string        `mapstructure:"payload"`
}

// C holds the global configuration.
var C Config
