package converters

import (
	"fmt"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/octopus"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/terraform"
	"github.com/mcasperson/OctopusTerraformExport/internal/util"
	"strings"
)

type LibraryVariableSetConverter struct {
	Client            client.OctopusClient
	SpaceResourceName string
	AccountsMap       map[string]string
}

func (c LibraryVariableSetConverter) ToHcl() (map[string]string, map[string]string, map[string]string, error) {
	resource := octopus.GeneralCollection[octopus.LibraryVariableSet]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &resource)

	if err != nil {
		return nil, nil, nil, err
	}

	resources := map[string]string{}
	resourcesMap := map[string]string{}
	templatesMap := map[string]string{}

	for _, v := range resource.Items {
		file := hclwrite.NewEmptyFile()

		resourceName := "library_variable_set_" + util.SanitizeName(v.Name)
		resourceIdProperty := "${octopusdeploy_library_variable_set." + resourceName + ".id}"
		templates, myTemplatesMap := c.convertTemplate(resourceName, v.Templates)
		if util.EmptyIfNil(v.ContentType) == "Variables" {
			terraformResource := terraform.TerraformLibraryVariableSet{
				Type:         "octopusdeploy_library_variable_set",
				Name:         resourceName,
				ResourceName: v.Name,
				Description:  v.Description,
				Template:     templates,
			}

			file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

			resources["space_population/"+resourceName+".tf"] = string(file.Bytes())
			resourcesMap[v.Id] = resourceIdProperty

			// Export variable set
			variableSet, err := VariableSetConverter{
				Client:      c.Client,
				AccountsMap: c.AccountsMap,
			}.ToHclById(v.VariableSetId, util.SanitizeName(v.Name), resourceIdProperty)

			if err != nil {
				return nil, nil, nil, err
			}

			// merge the maps
			for k, v := range variableSet {
				resources[k] = v
			}

			for k, v := range myTemplatesMap {
				templatesMap[k] = v
			}
		} else if util.EmptyIfNil(v.ContentType) == "ScriptModule" {
			variable := octopus.VariableSet{}
			err = c.Client.GetResourceById("Variables", v.VariableSetId, &variable)

			script := ""
			scriptLanguage := ""
			for _, u := range variable.Variables {
				if u.Name == "Octopus.Script.Module["+v.Name+"]" {
					script = strings.Clone(*u.Value)
				}

				if u.Name == "Octopus.Script.Module.Language["+v.Name+"]" {
					scriptLanguage = strings.Clone(*u.Value)
				}
			}

			terraformResource := terraform.TerraformScriptModule{
				Type:         "octopusdeploy_script_module",
				Name:         resourceName,
				ResourceName: v.Name,
				Description:  v.Description,
				Script: terraform.TerraformScriptModuleScript{
					Body:   script,
					Syntax: scriptLanguage,
				},
			}

			file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

			resources["space_population/"+resourceName+".tf"] = string(file.Bytes())
			resourcesMap[v.Id] = resourceIdProperty
		}
	}

	return resources, resourcesMap, templatesMap, nil
}

func (c LibraryVariableSetConverter) ToHclById(id string) (map[string]string, map[string]string, map[string]string, error) {
	resource := octopus.LibraryVariableSet{}
	err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return nil, nil, nil, err
	}

	file := hclwrite.NewEmptyFile()

	resourceName := "library_variable_set_" + util.SanitizeName(resource.Name)
	templates, templatesMap := c.convertTemplate(resourceName, resource.Templates)
	terraformResource := terraform.TerraformLibraryVariableSet{
		Type:         "octopusdeploy_library_variable_set",
		Name:         resourceName,
		ResourceName: resource.Name,
		Description:  resource.Description,
		Template:     templates,
	}

	file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

	return map[string]string{
			resourceName + ".tf": string(file.Bytes()),
		}, map[string]string{
			resource.Id: "${octopusdeploy_library_variable_set." + resourceName + ".id}",
		}, templatesMap, nil
}

func (c LibraryVariableSetConverter) GetResourceType() string {
	return "LibraryVariableSets"
}

func (c LibraryVariableSetConverter) convertTemplate(parentName string, template []octopus.Template) ([]terraform.TerraformTemplate, map[string]string) {
	templatesMap := map[string]string{}
	terraformTemplates := make([]terraform.TerraformTemplate, 0)
	for i, v := range template {
		terraformTemplates = append(terraformTemplates, terraform.TerraformTemplate{
			Name:            v.Name,
			Label:           v.Label,
			HelpText:        v.HelpText,
			DefaultValue:    v.DefaultValue,
			DisplaySettings: v.DisplaySettings,
		})
		templatesMap[v.Id] = "${octopusdeploy_library_variable_set." + parentName + ".template[" + fmt.Sprint(i) + "].id}"
	}

	return terraformTemplates, templatesMap
}
