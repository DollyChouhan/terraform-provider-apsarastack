package apsarastack

import (
	"github.com/aliyun/alibaba-cloud-sdk-go/services/vpc"
	"github.com/aliyun/terraform-provider-apsarastack/apsarastack/connectivity"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
)

func resourceApsaraStackRouteTable() *schema.Resource {
	return &schema.Resource{
		Create: resourceApsaraStackRouteTableCreate,
		Read:   resourceApsaraStackRouteTableRead,
		Update: resourceApsaraStackRouteTableUpdate,
		Delete: resourceApsaraStackRouteTableDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"description": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringLenBetween(2, 256),
			},
			"name": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringLenBetween(2, 128),
			},

			"vpc_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"tags": tagsSchema(),
		},
	}
}

func resourceApsaraStackRouteTableCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*connectivity.ApsaraStackClient)
	vpcService := VpcService{client}

	request := vpc.CreateCreateRouteTableRequest()
	request.RegionId = client.RegionId

	request.VpcId = d.Get("vpc_id").(string)
	request.RouteTableName = d.Get("name").(string)
	request.Description = d.Get("description").(string)
	request.ClientToken = buildClientToken(request.GetActionName())

	raw, err := client.WithVpcClient(func(vpcClient *vpc.Client) (interface{}, error) {
		return vpcClient.CreateRouteTable(request)
	})
	if err != nil {
		return WrapErrorf(err, DefaultErrorMsg, "apsarastack_route_table", request.GetActionName(), ApsaraStackSdkGoERROR)
	}
	addDebug(request.GetActionName(), raw, request.RpcRequest, request)
	response, _ := raw.(*vpc.CreateRouteTableResponse)
	d.SetId(response.RouteTableId)

	if err := vpcService.WaitForRouteTable(d.Id(), Available, DefaultTimeoutMedium); err != nil {
		return WrapError(err)
	}

	return resourceApsaraStackRouteTableUpdate(d, meta)
}

func resourceApsaraStackRouteTableRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*connectivity.ApsaraStackClient)
	vpcService := VpcService{client}
	object, err := vpcService.DescribeRouteTable(d.Id())
	if err != nil {
		if NotFoundError(err) {
			d.SetId("")
			return nil
		}
		return WrapError(err)
	}
	d.Set("vpc_id", object.VpcId)
	d.Set("name", object.RouteTableName)
	d.Set("description", object.Description)
	d.Set("tags", vpcTagsToMap(object.Tags.Tag))
	return nil
}

func resourceApsaraStackRouteTableUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*connectivity.ApsaraStackClient)
	vpcService := VpcService{client}
	if err := vpcService.setInstanceTags(d, TagResourceRouteTable); err != nil {
		return WrapError(err)
	}
	if d.IsNewResource() {
		d.Partial(false)
		return resourceApsaraStackRouteTableRead(d, meta)
	}
	request := vpc.CreateModifyRouteTableAttributesRequest()
	request.RegionId = client.RegionId
	request.RouteTableId = d.Id()

	if d.HasChange("description") {
		request.Description = d.Get("description").(string)
	}

	if d.HasChange("name") {
		request.RouteTableName = d.Get("name").(string)
	}

	raw, err := client.WithVpcClient(func(vpcClient *vpc.Client) (interface{}, error) {
		return vpcClient.ModifyRouteTableAttributes(request)
	})
	if err != nil {
		return WrapErrorf(err, DefaultErrorMsg, d.Id(), request.GetActionName(), ApsaraStackSdkGoERROR)
	}
	addDebug(request.GetActionName(), raw, request.RpcRequest, request)

	return resourceApsaraStackRouteTableRead(d, meta)
}

func resourceApsaraStackRouteTableDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*connectivity.ApsaraStackClient)
	routeTableService := VpcService{client}
	request := vpc.CreateDeleteRouteTableRequest()
	request.RegionId = client.RegionId
	request.RouteTableId = d.Id()

	raw, err := client.WithVpcClient(func(vpcClient *vpc.Client) (interface{}, error) {
		return vpcClient.DeleteRouteTable(request)
	})
	if err != nil {
		return WrapErrorf(err, DefaultErrorMsg, d.Id(), request.GetActionName(), ApsaraStackSdkGoERROR)
	}
	addDebug(request.GetActionName(), raw, request.RpcRequest, request)
	return WrapError(routeTableService.WaitForRouteTable(d.Id(), Deleted, DefaultTimeoutMedium))
}
