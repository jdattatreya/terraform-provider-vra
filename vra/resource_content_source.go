package vra

import (
	"fmt"

	"github.com/go-openapi/strfmt"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/vmware/vra-sdk-go/pkg/client/content_source"
	"github.com/vmware/vra-sdk-go/pkg/models"

	"log"
)

func resourceContentSource() *schema.Resource {
	return &schema.Resource{
		Create: resourceContentSourceCreate,
		Read:   resourceContentSourceRead,
		Delete: resourceContentSourceDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"type_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice([]string{"com.gitlab", "com.github", "com.vmware.marketplace", "org.bitbucket"}, true),
			},
			"project_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			//project_ids exists in the model but isn't actually fed back in a read operation
			//"project_ids": &schema.Schema{
			//	Type:     schema.TypeList,
			//	Optional: true,
			//	ForceNew: true,
			//	Elem: &schema.Schema{
			//		Type: schema.TypeString,
			//	},
			//},
			"created_at": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"created_by": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"last_updated_at": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"last_updated_by": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"org_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			//Treating this as a set requires us to do some gymnastics later with expanding/flattening
			"config": {
				Type:     schema.TypeSet,
				MaxItems: 1,
				Required: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"path": {
							Type:     schema.TypeString,
							Required: true,
						},
						"branch": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"repository": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"content_type": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validation.StringInSlice([]string{"BLUEPRINT", "IMAGE", "ABX_SCRIPTS", "TERRAFORM_CONFIGURATION"}, true),
						},
						"project_name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"integration_id": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"sync_enabled": {
				Type:     schema.TypeBool,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceContentSourceCreate(d *schema.ResourceData, m interface{}) error {
	log.Printf("Starting to create vra_ContentSource resource3")

	var projectIds []string
	apiClient := m.(*Client).apiClient

	name := d.Get("name").(string)
	typeID := d.Get("type_id").(string)
	projectID := d.Get("project_id").(string)

	config := expandContentSourceRepositoryConfig(d.Get("config").(*schema.Set).List())

	if v, ok := d.GetOk("project_ids"); ok {
		if !compareUnique(v.([]interface{})) {
			return fmt.Errorf("Specified project_ids are not unique")
		}
		projectIds = expandStringList(v.([]interface{}))
	}
	contentSourceSpecification := models.ContentSource{
		Name:        &name,
		TypeID:      &typeID,
		Config:      config[0],
		ProjectIds:  projectIds,
		ProjectID:   &projectID,
		SyncEnabled: d.Get("sync_enabled").(bool),
	}

	if v, ok := d.GetOk("description"); ok {
		contentSourceSpecification.Description = v.(string)
	}

	resp, err := apiClient.ContentSource.CreateContentSourceUsingPOST(content_source.NewCreateContentSourceUsingPOSTParams().WithSource(&contentSourceSpecification))

	if err != nil {
		return err
	}

	id := *resp.GetPayload().ID
	d.SetId(id.String())

	log.Printf("Finished creating vra_ContentSource resource with name %s", d.Get("name"))

	return resourceContentSourceRead(d, m)
}

func resourceContentSourceRead(d *schema.ResourceData, m interface{}) error {
	log.Printf("Reading the vra_ContentSource resource with name %s", d.Get("name"))
	apiClient := m.(*Client).apiClient

	id := d.Id()
	csUUID := strfmt.UUID(id)

	resp, err := apiClient.ContentSource.GetContentSourceUsingGET(content_source.NewGetContentSourceUsingGETParams().WithID(csUUID))

	if err != nil {
		switch err.(type) {
		case *content_source.GetContentSourceUsingGETNotFound:
			d.SetId("")
			return nil
		}
		return err
	}

	ContentSource := *resp.Payload
	d.Set("config", flattenContentsourceRepositoryConfig(ContentSource.Config))
	d.Set("id", ContentSource.ID)
	d.Set("last_updated_at", ContentSource.LastUpdatedAt)
	d.Set("last_updated_by", ContentSource.LastUpdatedBy)
	d.Set("created_at", ContentSource.CreatedAt)
	d.Set("created_by", ContentSource.CreatedBy)
	d.Set("description", ContentSource.Description)
	d.Set("name", ContentSource.Name)
	d.Set("org_id", ContentSource.OrgID)
	d.Set("project_id", ContentSource.ProjectID)
	d.Set("project_ids", ContentSource.ProjectIds)

	d.Set("sync_enabled", ContentSource.SyncEnabled)
	d.Set("type", ContentSource.Type)
	d.Set("type_id", ContentSource.TypeID)

	log.Printf("Finished reading the vra_ContentSource resource with name %s", d.Get("name"))
	return nil
}

func resourceContentSourceDelete(d *schema.ResourceData, m interface{}) error {
	log.Printf("Starting to delete the vra_ContentSource resource with name %s", d.Get("name"))
	apiClient := m.(*Client).apiClient

	id := d.Id()
	csUUID := strfmt.UUID(id)
	_, err := apiClient.ContentSource.DeleteContentSourceUsingDELETE(content_source.NewDeleteContentSourceUsingDELETEParams().WithID(csUUID))

	if err != nil {
		return err
	}

	d.SetId("")
	log.Printf("Finished deleting the vra_ContentSource resource with name %s", d.Get("name"))
	return nil
}
