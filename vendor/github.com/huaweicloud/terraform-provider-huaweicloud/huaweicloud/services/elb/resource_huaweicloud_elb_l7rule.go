package elb

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/chnsz/golangsdk"
	"github.com/chnsz/golangsdk/openstack/elb/v3/l7policies"

	"github.com/huaweicloud/terraform-provider-huaweicloud/huaweicloud/common"
	"github.com/huaweicloud/terraform-provider-huaweicloud/huaweicloud/config"
)

func ResourceL7RuleV3() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceL7RuleV3Create,
		ReadContext:   resourceL7RuleV3Read,
		UpdateContext: resourceL7RuleV3Update,
		DeleteContext: resourceL7RuleV3Delete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceELBL7RuleImport,
		},

		Timeouts: &schema.ResourceTimeout{
			Update: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"region": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"type": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: validation.StringInSlice([]string{
					"HOST_NAME", "PATH",
				}, true),
			},

			"compare_type": {
				Type:     schema.TypeString,
				Required: true,
				ValidateFunc: validation.StringInSlice([]string{
					"STARTS_WITH", "EQUAL_TO", "REGEX",
				}, true),
			},

			"l7policy_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"value": {
				Type:     schema.TypeString,
				Required: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					if len(v.(string)) == 0 {
						errors = append(errors, fmt.Errorf("'value' field should not be empty"))
					}
					return
				},
			},
		},
	}
}

func resourceL7RuleV3Create(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	cfg := meta.(*config.Config)
	elbClient, err := cfg.ElbV3Client(cfg.GetRegion(d))
	if err != nil {
		return diag.Errorf("error creating ELB client: %s", err)
	}

	l7PolicyID := d.Get("l7policy_id").(string)
	ruleType := d.Get("type").(string)
	compareType := d.Get("compare_type").(string)

	createOpts := l7policies.CreateRuleOpts{
		RuleType:    l7policies.RuleType(ruleType),
		CompareType: l7policies.CompareType(compareType),
		Value:       d.Get("value").(string),
	}

	log.Printf("[DEBUG] Create Options: %#v", createOpts)
	l7Rule, err := l7policies.CreateRule(elbClient, l7PolicyID, createOpts).Extract()
	if err != nil {
		return diag.Errorf("error creating L7 Rule: %s", err)
	}

	timeout := d.Timeout(schema.TimeoutCreate)
	// Wait for L7 Rule to become active before continuing
	err = waitForElbV3Rule(ctx, elbClient, l7PolicyID, l7Rule.ID, "ACTIVE", timeout)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(l7Rule.ID)

	return resourceL7RuleV3Read(ctx, d, meta)
}

func resourceL7RuleV3Read(_ context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	cfg := meta.(*config.Config)
	elbClient, err := cfg.ElbV3Client(cfg.GetRegion(d))
	if err != nil {
		return diag.Errorf("error creating ELB client: %s", err)
	}

	l7PolicyID := d.Get("l7policy_id").(string)

	l7Rule, err := l7policies.GetRule(elbClient, l7PolicyID, d.Id()).Extract()
	if err != nil {
		return common.CheckDeletedDiag(d, err, "L7 Rule")
	}

	log.Printf("[DEBUG] Retrieved L7 Rule %s: %#v", d.Id(), l7Rule)

	mErr := multierror.Append(nil,
		d.Set("l7policy_id", l7PolicyID),
		d.Set("type", l7Rule.RuleType),
		d.Set("compare_type", l7Rule.CompareType),
		d.Set("value", l7Rule.Value),
	)
	if err := mErr.ErrorOrNil(); err != nil {
		return diag.Errorf("error setting Dedicated ELB l7rule fields: %s", err)
	}

	return nil
}

func resourceL7RuleV3Update(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	cfg := meta.(*config.Config)
	elbClient, err := cfg.ElbV3Client(cfg.GetRegion(d))
	if err != nil {
		return diag.Errorf("error creating ELB client: %s", err)
	}

	l7PolicyID := d.Get("l7policy_id").(string)
	var updateOpts l7policies.UpdateRuleOpts

	if d.HasChange("compare_type") {
		updateOpts.CompareType = l7policies.CompareType(d.Get("compare_type").(string))
	}
	if d.HasChange("value") {
		updateOpts.Value = d.Get("value").(string)
	}

	log.Printf("[DEBUG] Updating L7 Rule %s with options: %#v", d.Id(), updateOpts)
	_, err = l7policies.UpdateRule(elbClient, l7PolicyID, d.Id(), updateOpts).Extract()
	if err != nil {
		return diag.Errorf("unable to update L7 Rule %s: %s", d.Id(), err)
	}

	timeout := d.Timeout(schema.TimeoutUpdate)
	// Wait for L7 Rule to become active before continuing
	err = waitForElbV3Rule(ctx, elbClient, l7PolicyID, d.Id(), "ACTIVE", timeout)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceL7RuleV3Read(ctx, d, meta)
}

func resourceL7RuleV3Delete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	cfg := meta.(*config.Config)
	elbClient, err := cfg.ElbV3Client(cfg.GetRegion(d))
	if err != nil {
		return diag.Errorf("error creating ELB client: %s", err)
	}

	l7PolicyID := d.Get("l7policy_id").(string)
	log.Printf("[DEBUG] Attempting to delete L7 Rule %s", d.Id())
	err = l7policies.DeleteRule(elbClient, l7PolicyID, d.Id()).ExtractErr()
	if err != nil {
		return common.CheckDeletedDiag(d, err, "error deleting L7 Rule")
	}

	timeout := d.Timeout(schema.TimeoutDelete)
	err = waitForElbV3Rule(ctx, elbClient, l7PolicyID, d.Id(), "DELETED", timeout)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func waitForElbV3Rule(ctx context.Context, elbClient *golangsdk.ServiceClient, l7policyID string,
	id string, target string, timeout time.Duration) error {
	log.Printf("[DEBUG] Waiting for rule %s to become %s", id, target)

	stateConf := &resource.StateChangeConf{
		Target:       []string{target},
		Pending:      nil,
		Refresh:      resourceElbV3RuleRefreshFunc(elbClient, l7policyID, id),
		Timeout:      timeout,
		Delay:        5 * time.Second,
		PollInterval: 3 * time.Second,
	}

	_, err := stateConf.WaitForStateContext(ctx)
	if err != nil {
		if _, ok := err.(golangsdk.ErrDefault404); ok {
			switch target {
			case "DELETED":
				return nil
			default:
				return fmt.Errorf("error: rule %s not found: %s", id, err)
			}
		}
		return fmt.Errorf("error waiting for rule %s to become %s: %s", id, target, err)
	}

	return nil
}

func resourceElbV3RuleRefreshFunc(elbClient *golangsdk.ServiceClient,
	l7PolicyID string, id string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		rule, err := l7policies.GetRule(elbClient, l7PolicyID, id).Extract()
		if err != nil {
			return nil, "", err
		}

		return rule, rule.ProvisioningStatus, nil
	}
}

func resourceELBL7RuleImport(_ context.Context, d *schema.ResourceData, _ interface{}) ([]*schema.ResourceData, error) {
	parts := strings.SplitN(d.Id(), "/", 2)
	if len(parts) != 2 {
		err := fmt.Errorf("invalid format specified for L7 Rule. Format must be <policy_id>/<rule_id>")
		return nil, err
	}

	l7PolicyID := parts[0]
	l7RuleID := parts[1]

	d.SetId(l7RuleID)
	d.Set("l7policy_id", l7PolicyID)

	return []*schema.ResourceData{d}, nil
}
