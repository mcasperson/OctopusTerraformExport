package converters

import (
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/octopus"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/terraform"
	"github.com/mcasperson/OctopusTerraformExport/internal/util"
)

type SshTargetConverter struct {
	Client client.OctopusClient
}

func (c SshTargetConverter) ToHcl(dependencies *ResourceDetailsCollection) error {
	collection := octopus.GeneralCollection[octopus.SshEndpointResource]{}
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

func (c SshTargetConverter) ToHclById(id string, dependencies *ResourceDetailsCollection) error {
	resource := octopus.SshEndpointResource{}
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	return c.toHcl(resource, true, dependencies)
}

func (c SshTargetConverter) toHcl(target octopus.SshEndpointResource, recursive bool, dependencies *ResourceDetailsCollection) error {
	if target.Endpoint.CommunicationStyle == "Ssh" {

		if recursive {
			err := c.exportDependencies(target, dependencies)

			if err != nil {
				return err
			}
		}

		targetName := "target_" + util.SanitizeName(target.Name)
		thisResource := ResourceDetails{}
		thisResource.FileName = "space_population/" + targetName + ".tf"
		thisResource.Id = target.Id
		thisResource.ResourceType = c.GetResourceType()
		thisResource.Lookup = "${octopusdeploy_ssh_connection_deployment_target." + targetName + ".id}"
		thisResource.ToHcl = func() (string, error) {

			terraformResource := terraform.TerraformSshConnectionDeploymentTarget{
				Type:               "octopusdeploy_ssh_connection_deployment_target",
				Name:               targetName,
				AccountId:          c.getAccount(target.Endpoint.AccountId, dependencies),
				Environments:       c.lookupEnvironments(target.EnvironmentIds, dependencies),
				Fingerprint:        target.Endpoint.Fingerprint,
				Host:               target.Endpoint.Host,
				ResourceName:       target.Name,
				Roles:              target.Roles,
				DotNetCorePlatform: &target.Endpoint.DotNetCorePlatform,
				MachinePolicyId:    c.getMachinePolicy(target.MachinePolicyId, dependencies),
			}
			file := hclwrite.NewEmptyFile()
			file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

			return string(file.Bytes()), nil
		}

		dependencies.AddResource(thisResource)
	}

	return nil
}

func (c SshTargetConverter) GetResourceType() string {
	return "Machines"
}

func (c SshTargetConverter) lookupEnvironments(envs []string, dependencies *ResourceDetailsCollection) []string {
	newEnvs := make([]string, len(envs))
	for i, v := range envs {
		newEnvs[i] = dependencies.GetResource("Environments", v)
	}
	return newEnvs
}

func (c SshTargetConverter) getMachinePolicy(machine string, dependencies *ResourceDetailsCollection) *string {
	machineLookup := dependencies.GetResource("MachinePolicies", machine)
	if machineLookup == "" {
		return nil
	}

	return &machineLookup
}

func (c SshTargetConverter) getAccount(account string, dependencies *ResourceDetailsCollection) string {
	accountLookup := dependencies.GetResource("Accounts", account)
	if accountLookup == "" {
		return ""
	}

	return accountLookup
}

func (c SshTargetConverter) exportDependencies(target octopus.SshEndpointResource, dependencies *ResourceDetailsCollection) error {

	// The machine policies need to be exported
	err := MachinePolicyConverter{
		Client: c.Client,
	}.ToHclById(target.MachinePolicyId, dependencies)

	if err != nil {
		return err
	}

	// Export the accounts
	err = AccountConverter{
		Client: c.Client,
	}.ToHclById(target.Endpoint.AccountId, dependencies)

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

	return nil
}
