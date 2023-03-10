package converters

import (
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hcl/hclsyntax"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/hcl"
	octopus2 "github.com/mcasperson/OctopusTerraformExport/cmd/internal/model/octopus"
	terraform2 "github.com/mcasperson/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/sanitizer"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/strutil"
)

type GitCredentialsConverter struct {
	Client            client.OctopusClient
	SpaceResourceName string
}

func (c GitCredentialsConverter) ToHcl(dependencies *ResourceDetailsCollection) error {
	collection := octopus2.GeneralCollection[octopus2.GitCredentials]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection)

	if err != nil {
		return err
	}

	for _, gitCredentials := range collection.Items {
		err = c.toHcl(gitCredentials, false, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c GitCredentialsConverter) ToHclById(id string, dependencies *ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	gitCredentials := octopus2.GitCredentials{}
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &gitCredentials)

	if err != nil {
		return err
	}

	return c.toHcl(gitCredentials, true, dependencies)
}

func (c GitCredentialsConverter) toHcl(gitCredentials octopus2.GitCredentials, recursive bool, dependencies *ResourceDetailsCollection) error {

	gitCredentialsName := "gitcredential_" + sanitizer.SanitizeName(gitCredentials.Name)

	thisResource := ResourceDetails{}
	thisResource.FileName = "space_population/" + gitCredentialsName + ".tf"
	thisResource.Id = gitCredentials.Id
	thisResource.ResourceType = c.GetResourceType()
	thisResource.Lookup = "${octopusdeploy_git_credential." + gitCredentialsName + ".id}"
	thisResource.ToHcl = func() (string, error) {

		terraformResource := terraform2.TerraformGitCredentials{
			Type:         "octopusdeploy_git_credential",
			Name:         gitCredentialsName,
			Description:  strutil.NilIfEmptyPointer(gitCredentials.Description),
			ResourceName: gitCredentials.Name,
			ResourceType: gitCredentials.Details.Type,
			Username:     gitCredentials.Details.Username,
			Password:     "${var." + gitCredentialsName + "}",
		}
		file := hclwrite.NewEmptyFile()

		// Add a comment with the import command
		baseUrl, _ := c.Client.GetSpaceBaseUrl()
		file.Body().AppendUnstructuredTokens([]*hclwrite.Token{{
			Type: hclsyntax.TokenComment,
			Bytes: []byte("# Import existing resources with the following commands:\n" +
				"# RESOURCE_ID=$(curl -H \"X-Octopus-ApiKey: ${OCTOPUS_CLI_API_KEY}\" " + baseUrl + "/" + c.GetResourceType() + " | jq -r '.Items[] | select(.Name==\"" + gitCredentials.Name + "\") | .Id')\n" +
				"# terraform import octopusdeploy_git_credential." + gitCredentialsName + " ${RESOURCE_ID}\n"),
			SpacesBefore: 0,
		}})

		file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

		secretVariableResource := terraform2.TerraformVariable{
			Name:        gitCredentialsName,
			Type:        "string",
			Nullable:    false,
			Sensitive:   true,
			Description: "The secret variable value associated with the git credential \"" + gitCredentials.Name + "\"",
		}

		block := gohcl.EncodeAsBlock(secretVariableResource, "variable")
		hcl.WriteUnquotedAttribute(block, "type", "string")
		file.Body().AppendBlock(block)

		return string(file.Bytes()), nil
	}

	dependencies.AddResource(thisResource)
	return nil
}

func (c GitCredentialsConverter) GetResourceType() string {
	return "Git-Credentials"
}
