package converters

import (
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/internal"
	"github.com/mcasperson/OctopusTerraformExport/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/internal/model"
	"github.com/mcasperson/OctopusTerraformExport/internal/util"
)

type SpaceConverter struct {
	Client client.OctopusClient
}

func (c SpaceConverter) ToHcl() (map[string]string, error) {
	space := model.Space{}
	err := c.Client.GetSpace(&space)

	if err != nil {
		return nil, err
	}

	terraformResource := model.TerraformSpace{
		Description:              space.Description,
		IsDefault:                space.IsDefault,
		IsTaskQueueStopped:       space.TaskQueueStopped,
		Name:                     "octopus_space_" + util.SanitizeName(space.Name),
		SpaceManagersTeamMembers: space.SpaceManagersTeamMembers,
		SpaceManagersTeams:       space.SpaceManagersTeams,
		ResourceName:             space.Name,
		Type:                     "octopusdeploy_space",
	}
	file := hclwrite.NewEmptyFile()
	file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

	results := map[string]string{
		internal.CreateSpaceDir + "/space.tf": string(file.Bytes()),
	}

	// Convert the projects
	projects, err := c.processProjects()
	if err != nil {
		return nil, err
	}

	// merge the maps
	for k, v := range projects {
		results[k] = v
	}

	return results, nil
}

func (c SpaceConverter) getResourceType() string {
	return "Spaces"
}

func (c SpaceConverter) processProjects() (map[string]string, error) {
	return ProjectConverter{
		Client: c.Client,
	}.ToHcl()
}
