package calico

import (
	"log"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/projectcalico/libcalico-go/lib/errors"
	"github.com/projectcalico/libcalico-go/lib/options"
	api "github.com/projectcalico/libcalico-go/lib/apis/v3"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func resourceCalicoIpPool() *schema.Resource {
	return &schema.Resource{
		Create: resourceCalicoIpPoolCreate,
		Read:   resourceCalicoIpPoolRead,
		Update: resourceCalicoIpPoolUpdate,
		Delete: resourceCalicoIpPoolDelete,

		Schema: map[string]*schema.Schema{
			"metadata": &schema.Schema {
				Type: schema.TypeList,
				Required: true,
				ForceNew: false,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: false,
						},
					},
				},
			},
			"spec": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: false,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"cidr": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"nat_outgoing": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
						},
						"ipip_mode": &schema.Schema{
							Type: schema.TypeString,
							Optional: true,
						},
						"disabled": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
						},
					},
				},
			},
		},
	}
}

// dToIpPoolSpec return the spec of the ippool
func dToIpPoolSpec(d *schema.ResourceData) (api.IPPoolSpec, error) {
	spec := api.IPPoolSpec{}

	cidr := d.Get("spec.0.cidr").(string)
	spec.CIDR = cidr

	ipipMode := d.Get("spec.0.ipip_mode").(api.IPIPMode)
	spec.IPIPMode = ipipMode

	natOutgoing := d.Get("spec.0.nat_outgoing").(bool)
	spec.NATOutgoing = natOutgoing

	disabled := d.Get("spec.0.disabled").(bool)
	spec.Disabled = disabled

	return spec, nil
}

// dToIpPoolSpec return the metadata of the ippool
func dToIpPoolTypeMeta(d *schema.ResourceData) (meta.ObjectMeta, error) {
	objectMeta := meta.ObjectMeta{}

	objectMeta.Name = d.Get("metadata.0.name").(string)

	return objectMeta, nil
}

// resourceCalicoIpPoolCreate create a new ippool in Calico
func resourceCalicoIpPoolCreate(d *schema.ResourceData, m interface{}) error {
	calicoClient := m.(config).Client
	ipPoolInterface := calicoClient.IPPools()

	ipPool, err := createIpPoolApiRequest(d)
	if err != nil {
		return err
	}

	_, err = ipPoolInterface.Create(ctx, ipPool, opts)
	if err != nil {
		return err
	}

	d.SetId(ipPool.ObjectMeta.Name)
	return resourceCalicoIpPoolRead(d, m)
}

// resourceCalicoIpPoolRead get a specific ippool
func resourceCalicoIpPoolRead(d *schema.ResourceData, m interface{}) error {
	calicoClient := m.(config).Client
	ipPoolInterface := calicoClient.IPPools()

	nameIpPool := d.Get("metadata.0.name").(string)

	ipPool, err := ipPoolInterface.Get(ctx, nameIpPool, options.GetOptions{})
	log.Printf("Obj: %+v", d)

	// Handle endpoint does not exist
	if err != nil {
		if _, ok := err.(errors.ErrorResourceDoesNotExist); ok {
			d.SetId("")
			return nil
		} else {
			return err
		}
	}

	d.SetId(nameIpPool)
	d.Set("metadata.0.name", ipPool.ObjectMeta.Name)
	d.Set("spec.0.cidr", ipPool.Spec.CIDR)
	d.Set("spec.0.ipip_mode", ipPool.Spec.IPIPMode)
	d.Set("spec.0.nat_outgoing", ipPool.Spec.NATOutgoing)
	d.Set("spec.0.disabled", ipPool.Spec.Disabled)

	return nil
}

// resourceCalicoIpPoolUpdate update an ippool in Calico
func resourceCalicoIpPoolUpdate(d *schema.ResourceData, m interface{}) error {
	d.Partial(false)

	calicoClient := m.(config).Client
	ipPoolInterface := calicoClient.IPPools()

	ipPool, err := createIpPoolApiRequest(d)
	if err != nil {
		return err
	}

	_, err = ipPoolInterface.Update(ctx, ipPool, opts)
	if err != nil {
		return err
	}

	return nil
}

// resourceCalicoIpPoolDelete delete an ippool in Calico
func resourceCalicoIpPoolDelete(d *schema.ResourceData, m interface{}) error {
	calicoClient := m.(config).Client
	ipPoolInterface := calicoClient.IPPools()

	nameIpPool := d.Get("metadata.0.name").(string)

	_, err := ipPoolInterface.Delete(ctx, nameIpPool, options.DeleteOptions{})
	if err != nil {
		return err
	}

	return nil
}

// createIpPoolApiRequest prepare the request of creation and update
func createIpPoolApiRequest(d *schema.ResourceData) (*api.IPPool, error) {
	// Set Spec to IpPool Spec
	spec, err := dToIpPoolSpec(d)
	if err != nil {
		return nil, err
	}

	// Set Metadata to Kubernetes Metadata
	objectMeta, err := dToIpPoolTypeMeta(d)
	if err != nil {
		return nil, err
	}

	// Create a new IP Pool, with TypeMeta filled in
	// Then, fill the metadata and the spec
	newIpPool := api.NewIPPool()
	newIpPool.ObjectMeta = objectMeta
	newIpPool.Spec = spec

	return newIpPool, nil
}
