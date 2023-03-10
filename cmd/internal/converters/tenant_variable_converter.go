package converters

import (
	"fmt"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/model/octopus"
	terraform2 "github.com/mcasperson/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/sanitizer"
)

type TenantVariableConverter struct {
	Client client.OctopusClient
}

func (c TenantVariableConverter) ToHcl(dependencies *ResourceDetailsCollection) error {
	collection := []octopus.TenantVariable{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection)

	if err != nil {
		return err
	}

	for _, resource := range collection {
		err = c.toHcl(resource, false, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c TenantVariableConverter) ToHclByTenantId(id string, dependencies *ResourceDetailsCollection) error {
	resource := octopus.TenantVariable{}
	err := c.Client.GetAllResources("Tenants/"+id+"/Variables", &resource)

	if err != nil {
		return err
	}

	return c.toHcl(resource, true, dependencies)
}

func (c TenantVariableConverter) toHcl(tenant octopus.TenantVariable, recursive bool, dependencies *ResourceDetailsCollection) error {

	for _, p := range tenant.ProjectVariables {

		projectVariableIndex := 0

		for env, variable := range p.Variables {
			for templateId, value := range variable {
				value := value

				projectVariableIndex++
				variableName := "tenantprojectvariable_" + fmt.Sprint(projectVariableIndex) + "_" + sanitizer.SanitizeName(tenant.TenantName)

				thisResource := ResourceDetails{}
				thisResource.FileName = "space_population/" + variableName + ".tf"
				thisResource.Id = templateId
				thisResource.ResourceType = c.GetResourceType()
				thisResource.Lookup = "${octopusdeploy_tenant_project_variable." + variableName + ".id}"
				thisResource.ToHcl = func() (string, error) {
					file := hclwrite.NewEmptyFile()

					terraformResource := terraform2.TerraformTenantProjectVariable{
						Type:          "octopusdeploy_tenant_project_variable",
						Name:          variableName,
						Id:            nil,
						EnvironmentId: dependencies.GetResource("Environments", env),
						ProjectId:     dependencies.GetResource("Projects", p.ProjectId),
						TemplateId:    dependencies.GetResource("ProjectTemplates", templateId),
						TenantId:      dependencies.GetResource("Tenants", tenant.TenantId),
						Value:         &value,
					}
					file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))
					return string(file.Bytes()), nil
				}
				dependencies.AddResource(thisResource)
			}
		}
	}

	for _, l := range tenant.LibraryVariables {
		commonVariableIndex := 0

		for id, value := range l.Variables {
			commonVariableIndex++
			variableName := "tenantcommonvariable" + fmt.Sprint(commonVariableIndex) + "_" + sanitizer.SanitizeName(tenant.TenantName)

			thisResource := ResourceDetails{}
			thisResource.FileName = "space_population/" + variableName + ".tf"
			thisResource.Id = id
			thisResource.ResourceType = c.GetResourceType()
			thisResource.Lookup = "${octopusdeploy_tenant_common_variable." + variableName + ".id}"
			thisResource.ToHcl = func() (string, error) {
				file := hclwrite.NewEmptyFile()
				terraformResource := terraform2.TerraformTenantCommonVariable{
					Type:                 "octopusdeploy_tenant_common_variable",
					Name:                 variableName,
					Id:                   nil,
					LibraryVariableSetId: dependencies.GetResource("LibraryVariableSets", l.LibraryVariableSetId),
					TemplateId:           dependencies.GetResource("CommonTemplateMap", id),
					TenantId:             dependencies.GetResource("Tenants", tenant.TenantId),
					Value:                &value,
				}
				file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))
				return string(file.Bytes()), nil
			}
			dependencies.AddResource(thisResource)
		}
	}

	return nil
}

func (c TenantVariableConverter) GetResourceType() string {
	return "TenantVariables/All"
}
