package internal

import (
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/internal/model"
	"github.com/mcasperson/OctopusTerraformExport/internal/util"
)

type spaceTerraform struct {
	Type                     string   `hcl:"type,label"`
	Name                     string   `hcl:"name,label"`
	Description              string   `hcl:"description"`
	ResourceName             string   `hcl:"name"`
	IsDefault                bool     `hcl:"is_default"`
	IsTaskQueueStopped       bool     `hcl:"is_task_queue_stopped"`
	SpaceManagersTeamMembers []string `hcl:"space_managers_team_members"`
	SpaceManagersTeams       []string `hcl:"space_managers_teams"`
}

type SpaceConverter struct {
	Client OctopusClient
}

func (c SpaceConverter) ToHcl() (string, error) {
	space := model.Space{}
	err := c.Client.GetSpace(&space)

	if err != nil {
		return "", err
	}

	output := ""

	terraformResource := spaceTerraform{
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
	output += string(file.Bytes())

	// Convert the projects
	projects, err := c.processProjects()
	if err != nil {
		return "", err
	}
	output += projects

	return output, nil
}

func (c SpaceConverter) getResourceType() string {
	return "Spaces"
}

func (c SpaceConverter) processProjects() (string, error) {
	return ProjectConverter{
		Client: c.Client,
	}.ToHcl()
}
