package enhancedconverter

import (
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/octopus"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/terraform"
	"github.com/mcasperson/OctopusTerraformExport/internal/util"
)

type GitCredentialsConverter struct {
	Client            client.OctopusClient
	SpaceResourceName string
}

func (c GitCredentialsConverter) ToHcl(dependencies *ResourceDetailsCollection) error {
	collection := octopus.GeneralCollection[octopus.GitCredentials]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection)

	if err != nil {
		return err
	}

	for _, gitCredentials := range collection.Items {
		err = c.toHcl(gitCredentials, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c GitCredentialsConverter) ToHclById(id string, dependencies *ResourceDetailsCollection) error {
	gitCredentials := octopus.GitCredentials{}
	err := c.Client.GetResourceById(c.GetResourceType(), id, &gitCredentials)

	if err != nil {
		return err
	}

	return c.toHcl(gitCredentials, dependencies)
}

func (c GitCredentialsConverter) toHcl(gitCredentials octopus.GitCredentials, dependencies *ResourceDetailsCollection) error {

	gitCredentialsName := "gitcredential_" + util.SanitizeName(gitCredentials.Name)

	thisResource := ResourceDetails{}
	thisResource.FileName = "space_population/" + gitCredentialsName + ".tf"
	thisResource.Id = gitCredentials.Id
	thisResource.ResourceType = c.GetResourceType()
	thisResource.Lookup = "${octopusdeploy_git_credential." + gitCredentialsName + ".id}"
	thisResource.ToHcl = func() (string, error) {

		terraformResource := terraform.TerraformGitCredentials{
			Type:         "octopusdeploy_git_credential",
			Name:         gitCredentialsName,
			Description:  util.NilIfEmptyPointer(gitCredentials.Description),
			ResourceName: gitCredentials.Name,
			ResourceType: gitCredentials.Details.Type,
			Username:     gitCredentials.Details.Username,
			Password:     "${var." + gitCredentialsName + "}",
		}
		file := hclwrite.NewEmptyFile()
		file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

		secretVariableResource := terraform.TerraformVariable{
			Name:        gitCredentialsName,
			Type:        "string",
			Nullable:    false,
			Sensitive:   true,
			Description: "The secret variable value associated with the git credential \"" + gitCredentials.Name + "\"",
		}

		block := gohcl.EncodeAsBlock(secretVariableResource, "variable")
		util.WriteUnquotedAttribute(block, "type", "string")
		file.Body().AppendBlock(block)

		return string(file.Bytes()), nil
	}

	dependencies.AddResource(thisResource)
	return nil
}

func (c GitCredentialsConverter) GetResourceType() string {
	return "Git-Credentials"
}