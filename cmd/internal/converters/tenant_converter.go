package converters

import (
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hcl/hclsyntax"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/hcl"
	octopus2 "github.com/mcasperson/OctopusTerraformExport/cmd/internal/model/octopus"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/sanitizer"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/strutil"
	"k8s.io/utils/strings/slices"
	"strings"
)

type TenantConverter struct {
	Client                  client.OctopusClient
	TenantVariableConverter ConverterByTenantId
	EnvironmentConverter    ConverterById
	TagSetConverter         ConvertToHclByResource[octopus2.TagSet]
}

func (c TenantConverter) ToHcl(dependencies *ResourceDetailsCollection) error {
	collection := octopus2.GeneralCollection[octopus2.Tenant]{}
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

func (c TenantConverter) ToHclByProjectId(projectId string, dependencies *ResourceDetailsCollection) error {
	collection := octopus2.GeneralCollection[octopus2.Tenant]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection, []string{"projectId", projectId})

	if err != nil {
		return nil
	}

	for _, tenant := range collection.Items {
		err = c.toHcl(tenant, true, dependencies)
		if err != nil {
			return nil
		}
	}
	return nil
}

func (c TenantConverter) ToHclById(id string, dependencies *ResourceDetailsCollection) error {
	tenant := octopus2.Tenant{}
	found, err := c.Client.GetResourceById(c.GetResourceType(), id, &tenant)

	if err != nil {
		return nil
	}

	if found {
		return c.toHcl(tenant, true, dependencies)
	}

	return nil
}

func (c TenantConverter) toHcl(tenant octopus2.Tenant, recursive bool, dependencies *ResourceDetailsCollection) error {

	if recursive {
		// Export the tenant variables
		err := c.TenantVariableConverter.ToHclByTenantId(tenant.Id, dependencies)

		if err != nil {
			return err
		}

		// Export the tenant environments
		for _, environments := range tenant.ProjectEnvironments {
			for _, environment := range environments {
				err = c.EnvironmentConverter.ToHclById(environment, dependencies)
			}
		}

		if err != nil {
			return err
		}
	}

	tagSetDependencies, err := c.addTagSetDependencies(tenant, recursive, dependencies)

	if err != nil {
		return err
	}

	tenantName := "tenant_" + sanitizer.SanitizeName(tenant.Name)

	thisResource := ResourceDetails{}
	thisResource.FileName = "space_population/" + tenantName + ".tf"
	thisResource.Id = tenant.Id
	thisResource.ResourceType = c.GetResourceType()
	thisResource.Lookup = "${octopusdeploy_tenant." + tenantName + ".id}"
	thisResource.ToHcl = func() (string, error) {
		terraformResource := terraform.TerraformTenant{
			Type:               "octopusdeploy_tenant",
			Name:               tenantName,
			ResourceName:       tenant.Name,
			Id:                 nil,
			ClonedFromTenantId: nil,
			Description:        strutil.NilIfEmptyPointer(tenant.Description),
			TenantTags:         tenant.TenantTags,
			ProjectEnvironment: c.getProjects(tenant.ProjectEnvironments, dependencies),
		}
		file := hclwrite.NewEmptyFile()

		// Add a comment with the import command
		baseUrl, _ := c.Client.GetSpaceBaseUrl()
		file.Body().AppendUnstructuredTokens([]*hclwrite.Token{{
			Type: hclsyntax.TokenComment,
			Bytes: []byte("# Import existing resources with the following commands:\n" +
				"# RESOURCE_ID=$(curl -H \"X-Octopus-ApiKey: ${OCTOPUS_CLI_API_KEY}\" " + baseUrl + "/" + c.GetResourceType() + " | jq -r '.Items[] | select(.Name==\"" + tenant.Name + "\") | .Id')\n" +
				"# terraform import octopusdeploy_tenant." + tenantName + " ${RESOURCE_ID}\n"),
			SpacesBefore: 0,
		}})

		block := gohcl.EncodeAsBlock(terraformResource, "resource")

		// Explicitly describe the dependency between a target and a tag set
		dependsOn := []string{}
		for resourceType, terraformDependencies := range tagSetDependencies {
			for _, terraformDependency := range terraformDependencies {
				dependency := dependencies.GetResource(resourceType, terraformDependency)
				dependency = hcl.RemoveId(hcl.RemoveInterpolation(dependency))
				dependsOn = append(dependsOn, dependency)
			}
		}

		hcl.WriteUnquotedAttribute(block, "depends_on", "["+strings.Join(dependsOn[:], ",")+"]")
		file.Body().AppendBlock(block)

		return string(file.Bytes()), nil
	}

	dependencies.AddResource(thisResource)

	return nil
}

func (c TenantConverter) GetResourceType() string {
	return "Tenants"
}

func (c TenantConverter) getProjects(tags map[string][]string, dependencies *ResourceDetailsCollection) []terraform.TerraformProjectEnvironment {
	terraformProjectEnvironments := make([]terraform.TerraformProjectEnvironment, len(tags))
	index := 0
	for k, v := range tags {
		terraformProjectEnvironments[index] = terraform.TerraformProjectEnvironment{
			Environments: c.lookupEnvironments(v, dependencies),
			ProjectId:    dependencies.GetResource("Projects", k),
		}
		index++
	}
	return terraformProjectEnvironments
}

func (c TenantConverter) lookupEnvironments(envs []string, dependencies *ResourceDetailsCollection) []string {
	newEnvs := make([]string, len(envs))
	for i, v := range envs {
		newEnvs[i] = dependencies.GetResource("Environments", v)
	}
	return newEnvs
}

// addTagSetDependencies finds the tag sets that contains the tags associated with a tenant. These dependencies are
// captured, as Terraform has no other way to map the dependency between a tagset and a tenant.
func (c TenantConverter) addTagSetDependencies(tenant octopus2.Tenant, recursive bool, dependencies *ResourceDetailsCollection) (map[string][]string, error) {
	collection := octopus2.GeneralCollection[octopus2.TagSet]{}
	err := c.Client.GetAllResources("TagSets", &collection)

	if err != nil {
		return nil, err
	}

	terraformDependencies := map[string][]string{}

	for _, tagSet := range collection.Items {
		for _, tag := range tagSet.Tags {
			for _, tenantTag := range tenant.TenantTags {
				if tag.CanonicalTagName == tenantTag {

					if !slices.Contains(terraformDependencies["TagSets"], tagSet.Id) {
						terraformDependencies["TagSets"] = append(terraformDependencies["TagSets"], tagSet.Id)
					}

					if !slices.Contains(terraformDependencies["Tags"], tag.Id) {
						terraformDependencies["Tags"] = append(terraformDependencies["Tags"], tag.Id)
					}

					if recursive {
						err = c.TagSetConverter.ToHclByResource(tagSet, dependencies)

						if err != nil {
							return nil, err
						}
					}
				}
			}
		}
	}

	return terraformDependencies, nil
}
