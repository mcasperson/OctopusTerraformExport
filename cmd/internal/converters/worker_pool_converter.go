package converters

import (
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hcl/hclsyntax"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/client"
	octopus2 "github.com/mcasperson/OctopusTerraformExport/cmd/internal/model/octopus"
	terraform2 "github.com/mcasperson/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/sanitizer"
)

type WorkerPoolConverter struct {
	Client client.OctopusClient
}

func (c WorkerPoolConverter) ToHcl(dependencies *ResourceDetailsCollection) error {
	collection := octopus2.GeneralCollection[octopus2.WorkerPool]{}
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

func (c WorkerPoolConverter) ToHclById(id string, dependencies *ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	pool := octopus2.WorkerPool{}
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &pool)

	if err != nil {
		return err
	}

	return c.toHcl(pool, true, dependencies)
}

func (c WorkerPoolConverter) toHcl(pool octopus2.WorkerPool, recursive bool, dependencies *ResourceDetailsCollection) error {
	resourceName := "workerpool_" + sanitizer.SanitizeNamePointer(&pool.Name)

	thisResource := ResourceDetails{}
	thisResource.FileName = "space_population/" + resourceName + ".tf"
	thisResource.Id = pool.Id
	thisResource.ResourceType = c.GetResourceType()

	if pool.WorkerPoolType == "DynamicWorkerPool" {
		if pool.Name == "Hosted Windows" || pool.Name == "Hosted Ubuntu" {
			thisResource.Lookup = "${data.octopusdeploy_worker_pools." + resourceName + ".worker_pools[0].id}"
		} else {
			thisResource.Lookup = "${octopusdeploy_dynamic_worker_pool." + resourceName + ".id}"
		}
	} else if pool.WorkerPoolType == "StaticWorkerPool" {
		if pool.Name == "Default Worker Pool" {
			thisResource.Lookup = "${data.octopusdeploy_worker_pools." + resourceName + ".worker_pools[0].id}"
		} else {
			thisResource.Lookup = "${octopusdeploy_static_worker_pool." + resourceName + ".id}"
		}
	}

	thisResource.ToHcl = func() (string, error) {
		if pool.WorkerPoolType == "DynamicWorkerPool" {
			/*
				These default pools are expected to be created in a new space, so
				we use a data lookup to reference them rather than create them.
			*/
			if pool.Name == "Hosted Windows" || pool.Name == "Hosted Ubuntu" {
				data := terraform2.TerraformWorkerPoolData{
					Type:         "octopusdeploy_worker_pools",
					Name:         resourceName,
					ResourceName: &pool.Name,
					Ids:          nil,
					PartialName:  nil,
					Skip:         0,
					Take:         1,
				}
				file := hclwrite.NewEmptyFile()
				file.Body().AppendBlock(gohcl.EncodeAsBlock(data, "data"))

				return string(file.Bytes()), nil
			} else {
				terraformResource := terraform2.TerraformWorkerPool{
					Type:         "octopusdeploy_dynamic_worker_pool",
					Name:         resourceName,
					ResourceName: pool.Name,
					Description:  pool.Description,
					IsDefault:    pool.IsDefault,
					SortOrder:    pool.SortOrder,
					WorkerType:   pool.WorkerType,
				}
				file := hclwrite.NewEmptyFile()

				// Add a comment with the import command
				baseUrl, _ := c.Client.GetSpaceBaseUrl()
				file.Body().AppendUnstructuredTokens([]*hclwrite.Token{{
					Type: hclsyntax.TokenComment,
					Bytes: []byte("# Import existing resources with the following commands:\n" +
						"# RESOURCE_ID=$(curl -H \"X-Octopus-ApiKey: ${OCTOPUS_CLI_API_KEY}\" " + baseUrl + "/" + c.GetResourceType() + " | jq -r '.Items[] | select(.Name==\"" + pool.Name + "\") | .Id')\n" +
						"# terraform import octopusdeploy_dynamic_worker_pool." + resourceName + " ${RESOURCE_ID}\n"),
					SpacesBefore: 0,
				}})

				file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

				return string(file.Bytes()), nil
			}
		}

		if pool.WorkerPoolType == "StaticWorkerPool" {
			/*
				This is the default pool available in every space. Use a data lookup for this pool.
			*/
			if pool.Name == "Default Worker Pool" {
				data := terraform2.TerraformWorkerPoolData{
					Type:         "octopusdeploy_worker_pools",
					Name:         resourceName,
					ResourceName: &pool.Name,
					Ids:          nil,
					PartialName:  nil,
					Skip:         0,
					Take:         1,
				}
				file := hclwrite.NewEmptyFile()
				file.Body().AppendBlock(gohcl.EncodeAsBlock(data, "data"))

				return string(file.Bytes()), nil
			} else {
				terraformResource := terraform2.TerraformWorkerPool{
					Type:         "octopusdeploy_static_worker_pool",
					Name:         resourceName,
					ResourceName: pool.Name,
					Description:  pool.Description,
					IsDefault:    pool.IsDefault,
					SortOrder:    pool.SortOrder,
					WorkerType:   pool.WorkerType,
				}
				file := hclwrite.NewEmptyFile()

				// Add a comment with the import command
				baseUrl, _ := c.Client.GetSpaceBaseUrl()
				file.Body().AppendUnstructuredTokens([]*hclwrite.Token{{
					Type: hclsyntax.TokenComment,
					Bytes: []byte("# Import existing resources with the following commands:\n" +
						"# RESOURCE_ID=$(curl -H \"X-Octopus-ApiKey: ${OCTOPUS_CLI_API_KEY}\" " + baseUrl + "/" + c.GetResourceType() + " | jq -r '.Items[] | select(.Name==\"" + pool.Name + "\") | .Id')\n" +
						"# terraform import octopusdeploy_static_worker_pool." + resourceName + " ${RESOURCE_ID}\n"),
					SpacesBefore: 0,
				}})

				file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

				return string(file.Bytes()), nil
			}
		}

		return "", nil
	}

	dependencies.AddResource(thisResource)
	return nil
}

func (c WorkerPoolConverter) GetResourceType() string {
	return "WorkerPools"
}
