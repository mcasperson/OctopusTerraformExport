package enhancedconverter

import (
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/octopus"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/terraform"
	"github.com/mcasperson/OctopusTerraformExport/internal/util"
)

type TagSetConverter struct {
	Client client.OctopusClient
}

func (c TagSetConverter) ToHcl(dependencies *ResourceDetailsCollection) error {
	collection := octopus.GeneralCollection[octopus.TagSet]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection)

	if err != nil {
		return err
	}

	for _, tagSet := range collection.Items {
		err = c.toHcl(tagSet, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c TagSetConverter) ToHclById(id string, dependencies *ResourceDetailsCollection) error {
	tagSet := octopus.TagSet{}
	err := c.Client.GetResourceById(c.GetResourceType(), id, &tagSet)

	if err != nil {
		return err
	}

	return c.toHcl(tagSet, dependencies)
}

func (c TagSetConverter) toHcl(tagSet octopus.TagSet, dependencies *ResourceDetailsCollection) error {
	tagSetName := "tagset_" + util.SanitizeName(tagSet.Name)

	thisResource := ResourceDetails{}
	thisResource.FileName = "space_population/" + tagSetName + ".tf"
	thisResource.Id = tagSet.Id
	thisResource.ResourceType = c.GetResourceType()
	thisResource.Lookup = "${octopusdeploy_tag_set." + tagSetName + ".id}"
	thisResource.ToHcl = func() (string, error) {
		terraformResource := terraform.TerraformTagSet{
			Type:         "octopusdeploy_tag_set",
			Name:         tagSetName,
			ResourceName: tagSet.Name,
			Description:  util.NilIfEmptyPointer(tagSet.Description),
			SortOrder:    tagSet.SortOrder,
		}
		file := hclwrite.NewEmptyFile()
		file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

		return string(file.Bytes()), nil
	}
	dependencies.AddResource(thisResource)

	for _, tag := range tagSet.Tags {
		tagName := "tag_" + util.SanitizeName(tag.Name)

		tagResource := ResourceDetails{}
		tagResource.FileName = "space_population/" + tagName + ".tf"
		tagResource.Id = tagSet.Id
		tagResource.ResourceType = c.GetResourceType()
		tagResource.Lookup = "${octopusdeploy_tag." + tagName + ".id}"
		tagResource.ToHcl = func() (string, error) {
			terraformResource := terraform.TerraformTag{
				Type:         "octopusdeploy_tag",
				Name:         tagName,
				ResourceName: tag.Name,
				TagSetId:     "${octopusdeploy_tag_set." + tagSetName + ".id}",
				Color:        tag.Color,
				Description:  tag.Description,
				SortOrder:    tag.SortOrder,
			}
			file := hclwrite.NewEmptyFile()
			file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

			return string(file.Bytes()), nil
		}
		dependencies.AddResource(tagResource)
	}

	return nil
}

func (c TagSetConverter) GetResourceType() string {
	return "TagSets"
}

func (c TagSetConverter) getTagSetTags(tags []octopus.Tag) []terraform.TerraformTag {
	terraformTags := make([]terraform.TerraformTag, len(tags))
	for i, v := range tags {
		terraformTags[i] = terraform.TerraformTag{
			ResourceName: v.Name,
			Color:        v.Color,
			Description:  util.NilIfEmptyPointer(v.Description),
			SortOrder:    v.SortOrder,
		}
	}
	return terraformTags
}