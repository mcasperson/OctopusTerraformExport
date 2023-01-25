package converters

import (
	"fmt"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/model/octopus"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/model/terraform"
	sanitizer2 "github.com/mcasperson/OctopusTerraformExport/cmd/internal/sanitizer"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/strutil"
	"regexp"
	"strings"
)

type DeploymentProcessConverter struct {
	Client client.OctopusClient
}

func (c DeploymentProcessConverter) ToHclByIdAndName(id string, projectName string, dependencies *ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus.DeploymentProcess{}
	found, err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	// Projects with no deployment process will not have a deployment process resources.
	// This is expected, so just return.
	if !found {
		return nil
	}

	return c.toHcl(resource, true, projectName, dependencies)
}

func (c DeploymentProcessConverter) toHcl(resource octopus.DeploymentProcess, recursive bool, projectName string, dependencies *ResourceDetailsCollection) error {
	resourceName := "deployment_process_" + sanitizer2.SanitizeName(projectName)

	thisResource := ResourceDetails{}

	if recursive {
		// Export linked accounts
		err := c.exportAccounts(resource, dependencies)
		if err != nil {
			return err
		}

		// Export linked feeds
		err = c.exportFeeds(resource, dependencies)
		if err != nil {
			return err
		}

		// Export linked worker pools
		err = c.exportWorkerPools(resource, dependencies)
		if err != nil {
			return err
		}
	}

	thisResource.FileName = "space_population/" + resourceName + ".tf"
	thisResource.Id = resource.Id
	thisResource.ResourceType = c.GetResourceType()
	thisResource.Lookup = "${octopusdeploy_deployment_process." + resourceName + ".id}"
	thisResource.ToHcl = func() (string, error) {

		terraformResource := terraform.TerraformDeploymentProcess{
			Type:      "octopusdeploy_deployment_process",
			Name:      resourceName,
			ProjectId: dependencies.GetResource("Projects", resource.ProjectId),
			Step:      make([]terraform.TerraformStep, len(resource.Steps)),
		}

		for i, s := range resource.Steps {
			terraformResource.Step[i] = terraform.TerraformStep{
				Name:               s.Name,
				PackageRequirement: s.PackageRequirement,
				Properties:         c.replaceFeedIds(s.Properties, dependencies),
				Condition:          s.Condition,
				StartTrigger:       s.StartTrigger,
				Action:             make([]terraform.TerraformAction, len(s.Actions)),
				TargetRoles:        c.getRoles(s.Properties),
			}

			for j, a := range s.Actions {

				actionResource := ResourceDetails{}
				actionResource.FileName = ""
				actionResource.Id = a.Id
				actionResource.ResourceType = "Actions"
				actionResource.Lookup = "${octopusdeploy_deployment_process." + resourceName + ".step[" + fmt.Sprint(i) + "].action[" + fmt.Sprint(j) + "].id}"
				dependencies.AddResource(actionResource)

				terraformResource.Step[i].Action[j] = terraform.TerraformAction{
					Name:                          a.Name,
					ActionType:                    a.ActionType,
					Notes:                         a.Notes,
					IsDisabled:                    a.IsDisabled,
					CanBeUsedForProjectVersioning: a.CanBeUsedForProjectVersioning,
					IsRequired:                    a.IsRequired,
					WorkerPoolId:                  dependencies.GetResource("WorkerPools", a.WorkerPoolId),
					Container:                     c.convertContainer(a.Container),
					WorkerPoolVariable:            a.WorkerPoolVariable,
					Environments:                  a.Environments,
					ExcludedEnvironments:          a.ExcludedEnvironments,
					Channels:                      a.Channels,
					TenantTags:                    a.TenantTags,
					Package:                       make([]terraform.TerraformPackage, len(a.Packages)),
					Condition:                     a.Condition,
					RunOnServer:                   c.getRunOnServer(a.Properties),
					Properties:                    c.removeUnnecessaryFields(c.replaceIds(c.escapeDollars(sanitizer2.SanitizeMap(a.Properties)), dependencies)),
					Features:                      c.getFeatures(a.Properties),
				}

				for k, p := range a.Packages {
					terraformResource.Step[i].Action[j].Package[k] = terraform.TerraformPackage{
						Name:                    p.Name,
						PackageID:               p.PackageId,
						AcquisitionLocation:     p.AcquisitionLocation,
						ExtractDuringDeployment: p.ExtractDuringDeployment,
						FeedId:                  dependencies.GetResourcePointer("Feeds", p.FeedId),
						Id:                      p.Id,
						Properties:              c.replaceIds(p.Properties, dependencies),
					}
				}
			}
		}

		file := hclwrite.NewEmptyFile()
		file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))
		return string(file.Bytes()), nil
	}

	dependencies.AddResource(thisResource)
	return nil
}

func (c DeploymentProcessConverter) GetResourceType() string {
	return "DeploymentProcesses"
}

func (c DeploymentProcessConverter) exportFeeds(resource octopus.DeploymentProcess, dependencies *ResourceDetailsCollection) error {
	feedRegex, _ := regexp.Compile("Feeds-\\d+")
	for _, step := range resource.Steps {
		for _, action := range step.Actions {

			for _, pack := range action.Packages {
				if pack.FeedId != nil {
					err := FeedConverter{
						Client: c.Client,
					}.ToHclById(strutil.EmptyIfNil(pack.FeedId), dependencies)

					if err != nil {
						return err
					}
				}
			}

			for _, prop := range action.Properties {
				for _, feed := range feedRegex.FindAllString(fmt.Sprint(prop), -1) {
					err := FeedConverter{
						Client: c.Client,
					}.ToHclById(feed, dependencies)

					if err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}

func (c DeploymentProcessConverter) exportAccounts(resource octopus.DeploymentProcess, dependencies *ResourceDetailsCollection) error {
	accountRegex, _ := regexp.Compile("Accounts-\\d+")
	for _, step := range resource.Steps {
		for _, action := range step.Actions {
			for _, prop := range action.Properties {
				for _, account := range accountRegex.FindAllString(fmt.Sprint(prop), -1) {
					err := AccountConverter{
						Client: c.Client,
					}.ToHclById(account, dependencies)

					if err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}

func (c DeploymentProcessConverter) exportWorkerPools(resource octopus.DeploymentProcess, dependencies *ResourceDetailsCollection) error {
	for _, step := range resource.Steps {
		for _, action := range step.Actions {
			if action.WorkerPoolId != "" {
				err := WorkerPoolConverter{
					Client: c.Client,
				}.ToHclById(action.WorkerPoolId, dependencies)

				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (c DeploymentProcessConverter) convertContainer(container octopus.Container) *terraform.TerraformContainer {
	if container.Image != nil || container.FeedId != nil {
		return &terraform.TerraformContainer{
			FeedId: container.FeedId,
			Image:  container.Image,
		}
	}

	return nil
}

func (c DeploymentProcessConverter) replaceIds(properties map[string]string, dependencies *ResourceDetailsCollection) map[string]string {
	return c.replaceFeedIds(c.replaceAccountIds(c.replaceAccountIds(properties, dependencies), dependencies), dependencies)
}

func (c DeploymentProcessConverter) escapeDollars(properties map[string]string) map[string]string {
	sanitisedProperties := map[string]string{}
	for k, v := range properties {
		sanitisedProperties[k] = strings.ReplaceAll(v, "$", "$$")
	}
	return sanitisedProperties
}

// removeUnnecessaryFields removes generic property bag values that have more specific terraform properties
func (c DeploymentProcessConverter) removeUnnecessaryFields(properties map[string]string) map[string]string {
	sanitisedProperties := map[string]string{}
	for k, v := range properties {
		if k != "Octopus.Action.RunOnServer" && k != "Octopus.Action.EnabledFeatures" && k != "Octopus.Action.TargetRoles" {
			sanitisedProperties[k] = v
		}
	}
	return sanitisedProperties
}

func (c DeploymentProcessConverter) getRunOnServer(properties map[string]any) bool {
	v, ok := properties["Octopus.Action.RunOnServer"]
	if ok {
		return strings.ToLower(fmt.Sprint(v)) == "true"
	}

	return true
}

// replaceFeedIds looks for any property value that is a valid feed ID and replaces it with a resource ID lookup.
// This also looks in the property values, for instance when you export a JSON blob that has feed references.
func (c DeploymentProcessConverter) replaceFeedIds(properties map[string]string, dependencies *ResourceDetailsCollection) map[string]string {
	for k, v := range properties {
		for _, v2 := range dependencies.GetAllResource("Feeds") {
			if strings.Contains(v, v2.Id) {
				properties[k] = strings.ReplaceAll(v, v2.Id, v2.Lookup)
			}
		}
	}

	return properties
}

// replaceAccountIds looks for any property value that is a valid account ID and replaces it with a resource ID lookup.
// This also looks in the property values, for instance when you export a JSON blob that has feed references.
func (c DeploymentProcessConverter) replaceAccountIds(properties map[string]string, dependencies *ResourceDetailsCollection) map[string]string {
	for k, v := range properties {
		for _, v2 := range dependencies.GetAllResource("Accounts") {
			if strings.Contains(v, v2.Id) {
				properties[k] = strings.ReplaceAll(v, v2.Id, v2.Lookup)
			}
		}
	}

	return properties
}

func (c DeploymentProcessConverter) getFeatures(properties map[string]any) []string {
	f, ok := properties["Octopus.Action.EnabledFeatures"]
	if ok {
		return strings.Split(fmt.Sprint(f), ",")
	}

	return []string{}
}

func (c DeploymentProcessConverter) getRoles(properties map[string]string) []string {
	f, ok := properties["Octopus.Action.TargetRoles"]
	if ok {
		return strings.Split(fmt.Sprint(f), ",")
	}

	return []string{}
}
