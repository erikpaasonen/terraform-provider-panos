package panos

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/PaloAltoNetworks/pango"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

// Provider returns a terraform.ResourceProvider.
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"hostname": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("PANOS_HOSTNAME", nil),
				Description: "Hostname/IP address of the Palo Alto Networks firewall to connect to",
			},
			"username": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("PANOS_USERNAME", nil),
				Description: "The username (not used if the ApiKey is set)",
			},
			"password": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("PANOS_PASSWORD", nil),
				Description: "The password (not used if the ApiKey is set)",
			},
			"api_key": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("PANOS_API_KEY", nil),
				Description: "The api key of the firewall",
			},
			"protocol": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The protocol (https or http)",
			},
			"port": &schema.Schema{
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "If the port is non-standard for the protocol, the port number to use",
			},
			"timeout": &schema.Schema{
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "The timeout for all communications with the firewall",
			},
			"logging": &schema.Schema{
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional:    true,
				Description: "Logging options for the API connection",
			},
			"json_config_file": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Retrieve the provider configuration from this JSON file",
			},
		},

		DataSourcesMap: map[string]*schema.Resource{
			"panos_system_info": dataSourceSystemInfo(),
		},

		ResourcesMap: map[string]*schema.Resource{
			// Panorama resources.
			"panos_panorama_address_group":      resourcePanoramaAddressGroup(),
			"panos_panorama_address_object":     resourcePanoramaAddressObject(),
			"panos_panorama_administrative_tag": resourcePanoramaAdministrativeTag(),
			"panos_panorama_device_group":       resourcePanoramaDeviceGroup(),
			"panos_panorama_device_group_entry": resourcePanoramaDeviceGroupEntry(),
			"panos_panorama_nat_policy":         resourcePanoramaNatPolicy(),
			"panos_panorama_security_policies":  resourcePanoramaSecurityPolicies(),
			"panos_panorama_service_group":      resourcePanoramaServiceGroup(),
			"panos_panorama_service_object":     resourcePanoramaServiceObject(),

			// Firewall resources.
			"panos_address_group":      resourceAddressGroup(),
			"panos_address_object":     resourceAddressObject(),
			"panos_administrative_tag": resourceAdministrativeTag(),
			"panos_dag_tags":           resourceDagTags(),
			"panos_ethernet_interface": resourceEthernetInterface(),
			"panos_general_settings":   resourceGeneralSettings(),
			"panos_management_profile": resourceManagementProfile(),
			"panos_nat_policy":         resourceNatPolicy(),
			"panos_security_policies":  resourceSecurityPolicies(),
			"panos_service_group":      resourceServiceGroup(),
			"panos_service_object":     resourceServiceObject(),
			"panos_virtual_router":     resourceVirtualRouter(),
			"panos_zone":               resourceZone(),
		},

		ConfigureFunc: providerConfigure,
	}
}

type CredsSpec struct {
	Hostname string   `json:"hostname"`
	Username string   `json:"username"`
	Password string   `json:"password"`
	ApiKey   string   `json:"api_key"`
	Protocol string   `json:"protocol"`
	Port     uint     `json:"port"`
	Timeout  int      `json:"timeout"`
	Logging  []string `json:"logging"`
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	var (
		logging uint32
		err     error
	)

	lm := map[string]uint32{
		"quiet":   pango.LogQuiet,
		"action":  pango.LogAction,
		"query":   pango.LogQuery,
		"op":      pango.LogOp,
		"uid":     pango.LogUid,
		"xpath":   pango.LogXpath,
		"send":    pango.LogSend,
		"receive": pango.LogReceive,
	}

	// Get connection settings from the plan file or environment variables.
	hostname := d.Get("hostname").(string)
	username := d.Get("username").(string)
	password := d.Get("password").(string)
	apiKey := d.Get("api_key").(string)
	protocol := d.Get("protocol").(string)
	port := uint(d.Get("port").(int))
	timeout := d.Get("timeout").(int)
	lc := d.Get("logging")
	if lc != nil {
		ll := lc.([]interface{})
		for i := range ll {
			s := ll[i].(string)
			if v, ok := lm[s]; !ok {
				return nil, fmt.Errorf("Unknown logging artifact requested: %s", s)
			} else {
				logging |= v
			}
		}
	}

	// Pull config from the JSON credentials file.
	filename := d.Get("json_config_file").(string)
	if filename != "" {
		b, err := ioutil.ReadFile(filename)
		if err != nil {
			return nil, err
		}

		cs := CredsSpec{}
		if err = json.Unmarshal(b, &cs); err != nil {
			return nil, err
		}

		// Spec file settings have the lowest priority, so only take params
		// that have their zero values.
		if hostname == "" && cs.Hostname != "" {
			hostname = cs.Hostname
		}
		if username == "" && cs.Username != "" {
			username = cs.Username
		}
		if password == "" && cs.Password != "" {
			password = cs.Password
		}
		if apiKey == "" && cs.ApiKey != "" {
			apiKey = cs.ApiKey
		}
		if protocol == "" && cs.Protocol != "" {
			protocol = cs.Protocol
		}
		if port == 0 && cs.Port != 0 {
			port = cs.Port
		}
		if timeout == 0 && cs.Timeout != 0 {
			timeout = cs.Timeout
		}
		if logging == 0 && len(cs.Logging) > 0 {
			for i := range cs.Logging {
				if v, ok := lm[cs.Logging[i]]; !ok {
					return nil, fmt.Errorf("Unknown logging artifact requested: %d", v)
				} else {
					logging |= v
				}
			}
		}
	}

	// Create the client connection.
	con, err := pango.Connect(pango.Client{
		Hostname: hostname,
		Username: username,
		Password: password,
		ApiKey:   apiKey,
		Protocol: protocol,
		Port:     port,
		Timeout:  timeout,
		Logging:  logging,
	})
	if err != nil {
		return nil, err
	}

	return con, nil
}
