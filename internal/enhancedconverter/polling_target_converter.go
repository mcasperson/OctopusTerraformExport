package enhancedconverter

import (
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/octopus"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/terraform"
	"github.com/mcasperson/OctopusTerraformExport/internal/util"
)

type PollingTargetConverter struct {
	Client client.OctopusClient
}

func (c PollingTargetConverter) ToHcl(dependencies *ResourceDetailsCollection) error {
	collection := octopus.GeneralCollection[octopus.PollingEndpointResource]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection)

	if err != nil {
		return err
	}

	for _, resource := range collection.Items {
		err = c.toHcl(resource, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c PollingTargetConverter) ToHclById(id string, dependencies *ResourceDetailsCollection) error {
	resource := octopus.PollingEndpointResource{}
	err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	return c.toHcl(resource, dependencies)
}

func (c PollingTargetConverter) toHcl(target octopus.PollingEndpointResource, dependencies *ResourceDetailsCollection) error {
	if target.Endpoint.CommunicationStyle == "TentacleActive" {
		// The machine policies need to be exported
		err := MachinePolicyConverter{
			Client: c.Client,
		}.ToHclById(target.MachinePolicyId, dependencies)

		if err != nil {
			return err
		}

		// Export the environments
		for _, e := range target.EnvironmentIds {
			err = EnvironmentConverter{
				Client: c.Client,
			}.ToHclById(e, dependencies)

			if err != nil {
				return err
			}
		}

		targetName := "target_" + util.SanitizeName(target.Name)

		thisResource := ResourceDetails{}
		thisResource.FileName = "space_population/" + targetName + ".tf"
		thisResource.Id = target.Id
		thisResource.ResourceType = c.GetResourceType()
		thisResource.Lookup = "${octopusdeploy_polling_tentacle_deployment_target." + targetName + ".id}"
		thisResource.ToHcl = func() (string, error) {

			terraformResource := terraform.TerraformPollingTentacleDeploymentTarget{
				Type:                            "octopusdeploy_polling_tentacle_deployment_target",
				Name:                            targetName,
				Environments:                    c.lookupEnvironments(target.EnvironmentIds, dependencies),
				ResourceName:                    target.Name,
				Roles:                           target.Roles,
				TentacleUrl:                     target.Endpoint.Uri,
				CertificateSignatureAlgorithm:   nil,
				HealthStatus:                    nil,
				IsDisabled:                      &target.IsDisabled,
				MachinePolicyId:                 c.getMachinePolicy(target.MachinePolicyId, dependencies),
				OperatingSystem:                 nil,
				ShellName:                       &target.ShellName,
				ShellVersion:                    &target.ShellVersion,
				SpaceId:                         nil,
				Status:                          nil,
				StatusSummary:                   nil,
				TenantTags:                      target.TenantTags,
				TenantedDeploymentParticipation: &target.TenantedDeploymentParticipation,
				Tenants:                         target.TenantIds,
				TentacleVersionDetails:          terraform.TerraformTentacleVersionDetails{},
				Uri:                             nil,
				Thumbprint:                      target.Thumbprint,
			}
			file := hclwrite.NewEmptyFile()
			file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

			return string(file.Bytes()), nil
		}

		dependencies.AddResource(thisResource)
	}

	return nil
}

func (c PollingTargetConverter) GetResourceType() string {
	return "Machines"
}

func (c PollingTargetConverter) lookupEnvironments(envs []string, dependencies *ResourceDetailsCollection) []string {
	newEnvs := make([]string, len(envs))
	for i, v := range envs {
		newEnvs[i] = dependencies.GetResource("Environments", v)
	}
	return newEnvs
}

func (c PollingTargetConverter) getMachinePolicy(machine string, dependencies *ResourceDetailsCollection) *string {
	machineLookup := dependencies.GetResource("MachinePolicies", machine)
	if machineLookup == "" {
		return nil
	}

	return &machineLookup
}
