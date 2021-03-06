// With many thanks to the example code from
// https://github.com/spaceapegames/terraform-provider-example
package provider

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	client "github.com/shakenfist/client-go"
)

func resourceInstance() *schema.Resource {
	fmt.Print()
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the instance",
				ForceNew:    true,
			},
			"uuid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The UUID of the instance",
			},
			"cpus": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "The number of CPUs for the instance",
				ForceNew:    true,
			},
			"memory": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "The amount of RAM for the instance in GB",
				ForceNew:    true,
			},
			"disk": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"size": {
							Type:        schema.TypeInt,
							Required:    true,
							ForceNew:    true,
							Description: "Size of disk in GB",
						},
						"base": {
							Type:        schema.TypeString,
							Optional:    true,
							ForceNew:    true,
							Description: "URL of disk image (or shortcut)",
						},
						"bus": {
							Type:        schema.TypeString,
							Required:    true,
							ForceNew:    true,
							Description: "Bus type of disk",
						},
						"type": {
							Type:        schema.TypeString,
							Required:    true,
							ForceNew:    true,
							Description: "Type of disk",
						},
					},
				},
			},
			"video": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"memory": {
							Type:        schema.TypeInt,
							Optional:    true,
							ForceNew:    true,
							Default:     16384,
							Description: "The amount of video card RAM in KB",
						},
						"model": {
							Type:        schema.TypeString,
							Optional:    true,
							ForceNew:    true,
							Default:     "cirrus",
							Description: "The video card model",
						},
					},
				},
			},
			"network": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"network_uuid": {
							Type:        schema.TypeString,
							Required:    true,
							ForceNew:    true,
							Description: "The UUID of the network",
						},
						"ipv4": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
							Computed: true,
							Description: "The " +
								"IPv4 address of the network interface",
							ValidateFunc: validateIPAddr,
						},
						"mac": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
							Computed: true,
							Description: "The " +
								"MAC address of the network interface",
							ValidateFunc: validateMAC,
						},
						"model": {
							Type:        schema.TypeString,
							Optional:    true,
							ForceNew:    true,
							Computed:    true,
							Description: "The model of the network interface",
						},
						"interface_uuid": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The UUID of the network interface",
						},
						"state": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The state of the network interface",
						},
					},
				},
			},
			"ssh_key": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Description: "The " +
					"ssh key to embed into the instance via config drive",
			},
			"node": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Shaken Fist node running this instance",
			},
			"user_data": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Description: "User data to pass " +
					"to the instance via config drive, encoded as base64",
			},
			"metadata": {
				Type:     schema.TypeMap,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString},
			},
			"console_port": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Console port number",
			},
			"vdi_port": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "VDI port number",
			},
			"state": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "State of the instance",
			},
			"power_state": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Power state of the instance",
			},
		},
		Create: resourceCreateInstance,
		Read:   resourceReadInstance,
		Delete: resourceDeleteInstance,
		Exists: resourceExistsInstance,
		Update: resourceUpdateInstance,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(5 * time.Minute),
			Delete: schema.DefaultTimeout(3 * time.Minute),
		},
	}
}

func validateIPAddr(v interface{}, k string) ([]string, []error) {
	var errs []error
	var warns []string

	value, ok := v.(string)
	if !ok {
		errs = append(errs, fmt.Errorf("Expected IP address to be a string"))
		return warns, errs
	}

	netblock := regexp.MustCompile(
		`^[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}$`)
	if !netblock.Match([]byte(value)) {
		errs = append(errs,
			fmt.Errorf("Netblock must be IPv4 address. Got %s", value))
		return warns, errs
	}
	return warns, errs
}

func validateMAC(v interface{}, k string) ([]string, []error) {
	var errs []error
	var warns []string

	value, ok := v.(string)
	if !ok {
		errs = append(errs, fmt.Errorf("Expected MAC address to be a string"))
		return warns, errs
	}

	netblock := regexp.MustCompile(
		`^[0-9a-fA-F]{2}\:[0-9a-fA-F]{2}\:[0-9a-fA-F]{2}\:` +
			`[0-9a-fA-F]{2}\:[0-9a-fA-F]{2}\:[0-9a-fA-F]{2}$`)
	if !netblock.Match([]byte(value)) {
		errs = append(errs,
			fmt.Errorf("MAC address must be a valid MAC. Got %s", value))
		return warns, errs
	}

	return warns, errs
}

func resourceCreateInstance(d *schema.ResourceData, m interface{}) error {
	apiClient := m.(*client.Client)

	var disks []client.DiskSpec
	var err error

	for _, d := range d.Get("disk").([]interface{}) {
		disk := d.(map[string]interface{})

		diskSpec := client.DiskSpec{
			Base: disk["base"].(string),
			Size: disk["size"].(int),
			Bus:  disk["bus"].(string),
			Type: disk["type"].(string),
		}

		disks = append(disks, diskSpec)
	}

	var networks []client.NetworkSpec
	for _, n := range d.Get("network").([]interface{}) {
		net := n.(map[string]interface{})

		netSpec := client.NetworkSpec{
			NetworkUUID: net["network_uuid"].(string),
		}

		if v, ok := net["ipv4"]; ok {
			netSpec.Address = v.(string)
		}
		if v, ok := net["mac"]; ok {
			netSpec.MACAddress = v.(string)
		}
		if v, ok := net["model"]; ok {
			netSpec.Model = v.(string)
		}

		networks = append(networks, netSpec)
	}

	videoConf := d.Get("video").([]interface{})
	if len(videoConf) > 1 {
		return fmt.Errorf("Instances only accept one video card ")
	}
	video := client.VideoSpec{}
	if len(videoConf) == 0 {
		video.Model = "cirrus"
		video.Memory = 16384
	} else {
		v := videoConf[0].(map[string]interface{})
		video.Model = v["model"].(string)
		video.Memory = v["memory"].(int)
	}

	inst, err := apiClient.CreateInstance(d.Get("name").(string),
		d.Get("cpus").(int), d.Get("memory").(int), networks, disks, video,
		d.Get("ssh_key").(string), d.Get("user_data").(string))
	if err != nil {
		return fmt.Errorf("Unable to create instance: %v", err)
	}

	// If Shaken Fist has a server error, it can return a blank UUID
	if inst.UUID == "" {
		return fmt.Errorf("Shaken Fist has returned a null instance UUID")
	}

	// Signal to TF that the instance exists by setting the ID.
	if err := d.Set("uuid", inst.UUID); err != nil {
		return fmt.Errorf("Instance UUID cannot be set: %v", err)
	}
	d.SetId(inst.UUID)

	// Set metadata on the instance
	for k, v := range d.Get("metadata").(map[string]interface{}) {
		val, ok := v.(string)
		if !ok {
			return fmt.Errorf("Tag value is not a string")
		}

		err := apiClient.SetMetadata(client.TypeInstance, inst.UUID, k, val)
		if err != nil {
			return fmt.Errorf("CreateInstance cannot store metadata: %v", err)
		}
	}

	return resource.Retry(d.Timeout(schema.TimeoutCreate),
		func() *resource.RetryError {

			i, err := apiClient.GetInstance(d.Id())
			if err != nil {
				if strings.Contains(err.Error(), "not found") {
					return resource.NonRetryableError(nil)
				} else {
					return resource.NonRetryableError(fmt.Errorf(
						"Unable to check instance existence: %v", err))
				}
			}

			if i.State == "error" {
				return resource.NonRetryableError(fmt.Errorf(
					"instance in error state"))
			}
			if i.State != "created" {
				return resource.RetryableError(fmt.Errorf(
					"instance not created"))
			}

			return resource.NonRetryableError(resourceReadInstance(d, m))
		},
	)
}

func resourceReadInstance(d *schema.ResourceData, m interface{}) error {
	apiClient := m.(*client.Client)

	inst, err := apiClient.GetInstance(d.Id())
	if err != nil {
		return fmt.Errorf("Unable to retrieve instance: %v", err)
	}

	if err := d.Set("uuid", inst.UUID); err != nil {
		return fmt.Errorf("Instance UUID cannot be set: %v", err)
	}
	if err := d.Set("name", inst.Name); err != nil {
		return fmt.Errorf("Instance Name cannot be set: %v", err)
	}
	if err := d.Set("cpus", inst.CPUs); err != nil {
		return fmt.Errorf("Instance CPUs cannot be set: %v", err)
	}
	if err := d.Set("memory", inst.Memory); err != nil {
		return fmt.Errorf("Instance Memory cannot be set: %v", err)
	}

	var disks []map[string]interface{}
	for _, d := range inst.DiskSpecs {
		disks = append(disks, map[string]interface{}{
			"size": d.Size,
			"base": d.Base,
			"bus":  d.Bus,
			"type": d.Type,
		})
	}
	if err := d.Set("disk", disks); err != nil {
		return fmt.Errorf("Instance DiskSpecs cannot be set: %v", err)
	}

	video := []map[string]interface{}{
		{
			"model":  inst.Video.Model,
			"memory": inst.Video.Memory,
		},
	}
	if err := d.Set("video", video); err != nil {
		return fmt.Errorf("Instance Video cannot be set: %v", err)
	}

	if err := d.Set("ssh_key", inst.SSHKey); err != nil {
		return fmt.Errorf("Instance SSHKey cannot be set: %v", err)
	}
	if err := d.Set("node", inst.Node); err != nil {
		return fmt.Errorf("Instance Node cannot be set: %v", err)
	}
	if err := d.Set("console_port", inst.ConsolePort); err != nil {
		return fmt.Errorf("Instance ConsolePort cannot be set: %v", err)
	}
	if err := d.Set("vdi_port", inst.VDIPort); err != nil {
		return fmt.Errorf("Instance VDIPort cannot be set: %v", err)
	}
	if err := d.Set("user_data", inst.UserData); err != nil {
		return fmt.Errorf("Instance UserData cannot be set: %v", err)
	}
	if err := d.Set("state", inst.State); err != nil {
		return fmt.Errorf("Instance State cannot be set: %v", err)
	}
	if err := d.Set("power_state", inst.PowerState); err != nil {
		return fmt.Errorf("Instance PowerState cannot be set: %v", err)
	}

	// Retrieve Interface UUID's
	uuid, err := getInterfaceUUIDs(apiClient, d.Id())
	if err != nil {
		return fmt.Errorf("ReadInstance error: %v", err)
	}

	var networks []map[string]interface{}
	for _, u := range uuid {
		n, err := apiClient.GetInterface(u)
		if err != nil {
			return fmt.Errorf("Cannot retrieve interface: %v", err)
		}

		networks = append(networks, map[string]interface{}{
			"network_uuid":   n.NetworkUUID,
			"ipv4":           n.IPv4,
			"mac":            n.MACAddress,
			"model":          n.Model,
			"interface_uuid": n.UUID,
			"state":          n.State,
		})
	}

	if err := d.Set("network", networks); err != nil {
		return fmt.Errorf("Instance networks cannot be set: %v", err)
	}

	// Retrieve metadata
	metadata, err := apiClient.GetMetadata(client.TypeInstance, inst.UUID)
	if err != nil {
		return fmt.Errorf("ReadInstance unable to retrieve metadata: %v", err)
	}
	if err := d.Set("metadata", metadata); err != nil {
		return fmt.Errorf("Instance Metadata cannot be set: %v", err)
	}

	return nil
}

func resourceDeleteInstance(d *schema.ResourceData, m interface{}) error {
	apiClient := m.(*client.Client)

	err := apiClient.DeleteInstance(d.Id())
	if err != nil {
		return fmt.Errorf("Unable to retrieve network: %v", err)
	}

	return resource.Retry(d.Timeout(schema.TimeoutDelete),
		func() *resource.RetryError {

			i, err := apiClient.GetInstance(d.Id())
			if err != nil {
				if strings.Contains(err.Error(), "not found") {
					return resource.NonRetryableError(nil)
				} else {
					return resource.NonRetryableError(fmt.Errorf(
						"Unable to check instance existence: %v", err))
				}
			}

			if i.State == "error" {
				return resource.NonRetryableError(fmt.Errorf(
					"instance in error state"))
			}
			if i.State != "deleted" {
				return resource.RetryableError(fmt.Errorf(
					"instance not deleted"))
			}

			d.SetId("")

			return resource.NonRetryableError(nil)
		},
	)
}

func resourceExistsInstance(d *schema.ResourceData, m interface{}) (bool, error) {
	apiClient := m.(*client.Client)

	i, err := apiClient.GetInstance(d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return false, nil
		} else {
			return false, fmt.Errorf("Unable to check instance existence: %v", err)
		}
	}

	if i.State == "deleted" {
		return false, nil
	}

	return true, nil
}

func resourceUpdateInstance(d *schema.ResourceData, m interface{}) error {
	apiClient := m.(*client.Client)

	if d.HasChange("metadata") {
		if err := updateMetadata(client.TypeInstance, d, apiClient); err != nil {
			return fmt.Errorf("UpdateInstance error: %v", err)
		}
	}

	return nil
}

// getInterfaceUUIDS returns a list of network UUID's as connected to the
// interfaces on the instance.
//
// The returned list uses the order as returned by the Shaken Fist server. This
// is required for Terraform to accurately report UUID's to other resources eg.
// Float resources.
func getInterfaceUUIDs(apiClient *client.Client, instanceUUID string) ([]string, error) {
	interfaces, err := apiClient.GetInstanceInterfaces(instanceUUID)
	if err != nil {
		return []string{}, fmt.Errorf(
			"unable to retrieve instance interfaces: %v", err)
	}

	// Ensure ordering is the same as the Shaken Fist order.
	sort.Slice(interfaces, func(i, j int) bool {
		return interfaces[i].Order < interfaces[j].Order
	})

	var uuid []string
	for _, i := range interfaces {
		uuid = append(uuid, i.UUID)
	}

	return uuid, nil
}
