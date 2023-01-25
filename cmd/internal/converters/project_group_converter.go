package converters

import (
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/client"
	octopus2 "github.com/mcasperson/OctopusTerraformExport/cmd/internal/model/octopus"
	terraform2 "github.com/mcasperson/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/sanitizer"
)

type ProjectGroupConverter struct {
	Client client.OctopusClient
}

func (c ProjectGroupConverter) ToHcl(dependencies *ResourceDetailsCollection) error {
	collection := octopus2.GeneralCollection[octopus2.ProjectGroup]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection)

	if err != nil {
		return err
	}

	for _, resource := range collection.Items {
		err = c.toHcl(resource, false, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c ProjectGroupConverter) ToHclById(id string, dependencies *ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(c.GetResourceType(), id) {
		return nil
	}

	resource := octopus2.ProjectGroup{}
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	return c.toHcl(resource, false, dependencies)
}

func (c ProjectGroupConverter) toHcl(resource octopus2.ProjectGroup, recursive bool, dependencies *ResourceDetailsCollection) error {
	thisResource := ResourceDetails{}

	projectName := "project_group_" + sanitizer.SanitizeName(resource.Name)

	thisResource.FileName = "space_population/projectgroup_" + projectName + ".tf"
	thisResource.Id = resource.Id
	thisResource.ResourceType = c.GetResourceType()
	if resource.Name == "Default Project Group" {
		thisResource.Lookup = "${data.octopusdeploy_project_groups." + projectName + ".project_groups[0].id}"
	} else {
		thisResource.Lookup = "${octopusdeploy_project_group." + projectName + ".id}"
	}
	thisResource.ToHcl = func() (string, error) {

		if resource.Name == "Default Project Group" {
			terraformResource := terraform2.TerraformProjectGroupData{
				Type:        "octopusdeploy_project_groups",
				Name:        projectName,
				Ids:         nil,
				PartialName: resource.Name,
				Skip:        0,
				Take:        1,
			}
			file := hclwrite.NewEmptyFile()
			file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "data"))

			return string(file.Bytes()), nil
		} else {
			terraformResource := terraform2.TerraformProjectGroup{
				Type:         "octopusdeploy_project_group",
				Name:         projectName,
				ResourceName: resource.Name,
				Description:  resource.Description,
			}
			file := hclwrite.NewEmptyFile()
			file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

			return string(file.Bytes()), nil
		}
	}

	if recursive {
		// export child projects
	}

	dependencies.AddResource(thisResource)
	return nil
}

func (c ProjectGroupConverter) GetResourceType() string {
	return "ProjectGroups"
}