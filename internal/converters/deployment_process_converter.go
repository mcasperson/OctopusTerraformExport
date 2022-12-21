package converters

import (
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/internal"
	"github.com/mcasperson/OctopusTerraformExport/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/internal/model"
	"github.com/mcasperson/OctopusTerraformExport/internal/util"
)

type DeploymentProcessConverter struct {
	Client client.OctopusClient
}

func (c DeploymentProcessConverter) ToHclById(id string, parentName string) (map[string]string, error) {
	resource := model.DeploymentProcess{}
	err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return nil, err
	}

	resourceName := util.SanitizeName(resource.Id)

	terraformResource := model.TerraformDeploymentProcess{
		Type:      "octopusdeploy_deployment_process",
		Name:      resourceName,
		ProjectId: "octopusdeploy_project." + parentName + ".id",
		Step:      make([]model.TerraformStep, len(resource.Steps)),
	}

	for i, s := range resource.Steps {
		terraformResource.Step[i] = model.TerraformStep{
			Name:               s.Name,
			PackageRequirement: s.PackageRequirement,
			Properties:         s.Properties,
			Condition:          s.Condition,
			StartTrigger:       s.StartTrigger,
			Action:             make([]model.TerraformAction, len(s.Actions)),
		}

		for j, a := range s.Actions {
			terraformResource.Step[i].Action[j] = model.TerraformAction{
				Name:                          a.Name,
				ActionType:                    a.ActionType,
				Notes:                         a.Notes,
				IsDisabled:                    a.IsDisabled,
				CanBeUsedForProjectVersioning: a.CanBeUsedForProjectVersioning,
				IsRequired:                    a.IsRequired,
				WorkerPoolId:                  a.WorkerPoolId,
				Container:                     c.convertContainer(a.Container),
				WorkerPoolVariable:            a.WorkerPoolVariable,
				Environments:                  a.Environments,
				ExcludedEnvironments:          a.ExcludedEnvironments,
				Channels:                      a.Channels,
				TenantTags:                    a.TenantTags,
				Package:                       make([]model.TerraformPackage, len(a.Packages)),
				Condition:                     a.Condition,
				Properties:                    a.Properties,
			}

			for k, p := range a.Packages {
				terraformResource.Step[i].Action[j].Package[k] = model.TerraformPackage{
					Name:                    p.Name,
					PackageID:               p.PackageId,
					AcquisitionLocation:     p.AcquisitionLocation,
					ExtractDuringDeployment: p.ExtractDuringDeployment,
					FeedId:                  p.FeedId,
					Id:                      p.Id,
					Properties:              p.Properties,
				}
			}
		}
	}

	file := hclwrite.NewEmptyFile()
	file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

	return map[string]string{
		internal.PopulateSpaceDir + "/" + resourceName + ".tf": string(file.Bytes()),
	}, nil
}

func (c DeploymentProcessConverter) GetResourceType() string {
	return "DeploymentProcesses"
}

func (c DeploymentProcessConverter) convertContainer(container model.Container) *model.TerraformContainer {
	if container.Image != nil || container.FeedId != nil {
		return &model.TerraformContainer{
			FeedId: container.FeedId,
			Image:  container.Image,
		}
	}

	return nil
}