package converters

import (
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hcl/hclsyntax"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/client"
	octopus2 "github.com/mcasperson/OctopusTerraformExport/cmd/internal/model/octopus"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/sanitizer"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/strutil"
)

type KubernetesTargetConverter struct {
	Client                 client.OctopusClient
	MachinePolicyConverter ConverterById
	AccountConverter       ConverterById
	CertificateConverter   ConverterById
	EnvironmentConverter   ConverterById
}

func (c KubernetesTargetConverter) ToHcl(dependencies *ResourceDetailsCollection) error {
	collection := octopus2.GeneralCollection[octopus2.KubernetesEndpointResource]{}
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

func (c KubernetesTargetConverter) ToHclById(id string, dependencies *ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus2.KubernetesEndpointResource{}
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	return c.toHcl(resource, true, dependencies)
}

func (c KubernetesTargetConverter) toHcl(target octopus2.KubernetesEndpointResource, recursive bool, dependencies *ResourceDetailsCollection) error {

	if target.Endpoint.CommunicationStyle == "Kubernetes" {
		if recursive {
			err := c.exportDependencies(target, dependencies)

			if err != nil {
				return err
			}
		}

		targetName := "target_" + sanitizer.SanitizeName(target.Name)

		thisResource := ResourceDetails{}
		thisResource.FileName = "space_population/" + targetName + ".tf"
		thisResource.Id = target.Id
		thisResource.ResourceType = c.GetResourceType()
		thisResource.Lookup = "${octopusdeploy_kubernetes_cluster_deployment_target." + targetName + ".id}"
		thisResource.ToHcl = func() (string, error) {

			terraformResource := terraform.TerraformKubernetesEndpointResource{
				Type:                            "octopusdeploy_kubernetes_cluster_deployment_target",
				Name:                            targetName,
				ClusterUrl:                      strutil.EmptyIfNil(target.Endpoint.ClusterUrl),
				Environments:                    c.lookupEnvironments(target.EnvironmentIds, dependencies),
				ResourceName:                    target.Name,
				Roles:                           target.Roles,
				ClusterCertificate:              dependencies.GetResourcePointer("Certificate", target.Endpoint.ClusterCertificate),
				DefaultWorkerPoolId:             c.getWorkerPool(target.Endpoint.DefaultWorkerPoolId, dependencies),
				HealthStatus:                    nil,
				Id:                              nil,
				IsDisabled:                      strutil.NilIfFalse(target.IsDisabled),
				MachinePolicyId:                 c.getMachinePolicy(target.MachinePolicyId, dependencies),
				Namespace:                       strutil.NilIfEmptyPointer(target.Endpoint.Namespace),
				OperatingSystem:                 nil,
				ProxyId:                         nil,
				RunningInContainer:              nil,
				ShellName:                       nil,
				ShellVersion:                    nil,
				SkipTlsVerification:             strutil.ParseBoolPointer(target.Endpoint.SkipTlsVerification),
				SpaceId:                         nil,
				Status:                          nil,
				StatusSummary:                   nil,
				TenantTags:                      target.TenantTags,
				TenantedDeploymentParticipation: target.TenantedDeploymentParticipation,
				Tenants:                         target.TenantIds,
				Thumbprint:                      nil,
				Uri:                             target.Uri,
				Endpoint: terraform.TerraformKubernetesEndpoint{
					CommunicationStyle: "Kubernetes",
				},
				Container: terraform.TerraformKubernetesContainer{
					FeedId: target.Endpoint.Container.FeedId,
					Image:  target.Endpoint.Container.Image,
				},
				Authentication:                      c.getK8sAuth(&target, dependencies),
				AwsAccountAuthentication:            c.getAwsAuth(&target, dependencies),
				AzureServicePrincipalAuthentication: c.getAzureAuth(&target, dependencies),
				CertificateAuthentication:           c.getCertAuth(&target, dependencies),
				GcpAccountAuthentication:            c.getGoogleAuth(&target, dependencies),
			}
			file := hclwrite.NewEmptyFile()

			// Add a comment with the import command
			baseUrl, _ := c.Client.GetSpaceBaseUrl()
			file.Body().AppendUnstructuredTokens([]*hclwrite.Token{{
				Type: hclsyntax.TokenComment,
				Bytes: []byte("# Import existing resources with the following commands:\n" +
					"# RESOURCE_ID=$(curl -H \"X-Octopus-ApiKey: ${OCTOPUS_CLI_API_KEY}\" " + baseUrl + "/" + c.GetResourceType() + " | jq -r '.Items[] | select(.Name==\"" + target.Name + "\") | .Id')\n" +
					"# terraform import octopusdeploy_kubernetes_cluster_deployment_target." + targetName + " ${RESOURCE_ID}\n"),
				SpacesBefore: 0,
			}})

			file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

			return string(file.Bytes()), nil
		}

		dependencies.AddResource(thisResource)
	}

	return nil
}

func (c KubernetesTargetConverter) GetResourceType() string {
	return "Machines"
}

func (c KubernetesTargetConverter) getAwsAuth(target *octopus2.KubernetesEndpointResource, dependencies *ResourceDetailsCollection) *terraform.TerraformAwsAccountAuthentication {
	if target.Endpoint.Authentication.AuthenticationType == "KubernetesAws" {
		return &terraform.TerraformAwsAccountAuthentication{
			AccountId:                 c.getAccount(target.Endpoint.Authentication.AccountId, dependencies),
			ClusterName:               strutil.EmptyIfNil(target.Endpoint.Authentication.ClusterName),
			AssumeRole:                target.Endpoint.Authentication.AssumeRole,
			AssumeRoleExternalId:      target.Endpoint.Authentication.AssumeRoleExternalId,
			AssumeRoleSessionDuration: target.Endpoint.Authentication.AssumeRoleSessionDurationSeconds,
			AssumedRoleArn:            target.Endpoint.Authentication.AssumedRoleArn,
			AssumedRoleSession:        target.Endpoint.Authentication.AssumedRoleSession,
			UseInstanceRole:           target.Endpoint.Authentication.UseInstanceRole,
		}
	}

	return nil
}

func (c KubernetesTargetConverter) getK8sAuth(target *octopus2.KubernetesEndpointResource, dependencies *ResourceDetailsCollection) *terraform.TerraformAccountAuthentication {
	if target.Endpoint.Authentication.AuthenticationType == "KubernetesStandard" {
		return &terraform.TerraformAccountAuthentication{
			AccountId: c.getAccount(target.Endpoint.Authentication.AccountId, dependencies),
		}
	}

	return nil
}

func (c KubernetesTargetConverter) getGoogleAuth(target *octopus2.KubernetesEndpointResource, dependencies *ResourceDetailsCollection) *terraform.TerraformGcpAccountAuthentication {
	if target.Endpoint.Authentication.AuthenticationType == "KubernetesGoogleCloud" {
		return &terraform.TerraformGcpAccountAuthentication{
			AccountId:                 c.getAccount(target.Endpoint.Authentication.AccountId, dependencies),
			ClusterName:               strutil.EmptyIfNil(target.Endpoint.Authentication.ClusterName),
			Project:                   strutil.EmptyIfNil(target.Endpoint.Authentication.Project),
			ImpersonateServiceAccount: target.Endpoint.Authentication.ImpersonateServiceAccount,
			Region:                    target.Endpoint.Authentication.Region,
			ServiceAccountEmails:      target.Endpoint.Authentication.ServiceAccountEmails,
			Zone:                      target.Endpoint.Authentication.Zone,
			UseVmServiceAccount:       target.Endpoint.Authentication.UseVmServiceAccount,
		}
	}

	return nil
}

func (c KubernetesTargetConverter) getCertAuth(target *octopus2.KubernetesEndpointResource, dependencies *ResourceDetailsCollection) *terraform.TerraformCertificateAuthentication {
	if target.Endpoint.Authentication.AuthenticationType == "KubernetesCertificate" {
		return &terraform.TerraformCertificateAuthentication{
			ClientCertificate: dependencies.GetResourcePointer("Certificates", target.Endpoint.Authentication.ClientCertificate),
		}
	}

	return nil
}

func (c KubernetesTargetConverter) getAzureAuth(target *octopus2.KubernetesEndpointResource, dependencies *ResourceDetailsCollection) *terraform.TerraformAzureServicePrincipalAuthentication {
	if target.Endpoint.Authentication.AuthenticationType == "KubernetesAzure" {
		return &terraform.TerraformAzureServicePrincipalAuthentication{
			AccountId:            c.getAccount(target.Endpoint.Authentication.AccountId, dependencies),
			ClusterName:          strutil.EmptyIfNil(target.Endpoint.Authentication.ClusterName),
			ClusterResourceGroup: strutil.EmptyIfNil(target.Endpoint.Authentication.ClusterResourceGroup),
		}
	}

	return nil
}

func (c KubernetesTargetConverter) lookupEnvironments(envs []string, dependencies *ResourceDetailsCollection) []string {
	newEnvs := make([]string, len(envs))
	for i, v := range envs {
		newEnvs[i] = dependencies.GetResource("Environments", v)
	}
	return newEnvs
}

func (c KubernetesTargetConverter) getMachinePolicy(machine string, dependencies *ResourceDetailsCollection) *string {
	machineLookup := dependencies.GetResource("MachinePolicies", machine)
	if machineLookup == "" {
		return nil
	}

	return &machineLookup
}

func (c KubernetesTargetConverter) getAccount(account *string, dependencies *ResourceDetailsCollection) string {
	if account == nil {
		return ""
	}

	accountLookup := dependencies.GetResource("Accounts", *account)
	if accountLookup == "" {
		return ""
	}

	return accountLookup
}

func (c KubernetesTargetConverter) getWorkerPool(pool *string, dependencies *ResourceDetailsCollection) *string {
	if pool == nil {
		return nil
	}

	workerPoolLookup := dependencies.GetResource("WorkerPools", *pool)
	if workerPoolLookup == "" {
		return nil
	}

	return &workerPoolLookup
}

func (c KubernetesTargetConverter) exportDependencies(target octopus2.KubernetesEndpointResource, dependencies *ResourceDetailsCollection) error {
	// The machine policies need to be exported
	err := c.MachinePolicyConverter.ToHclById(target.MachinePolicyId, dependencies)

	if err != nil {
		return err
	}

	// Export the accounts
	if target.Endpoint.Authentication.AccountId != nil {
		err = c.AccountConverter.ToHclById(*target.Endpoint.Authentication.AccountId, dependencies)

		if err != nil {
			return err
		}
	}

	// Export the certificate
	if target.Endpoint.Authentication.ClientCertificate != nil {
		err = c.CertificateConverter.ToHclById(*target.Endpoint.Authentication.ClientCertificate, dependencies)

		if err != nil {
			return err
		}
	}

	if target.Endpoint.ClusterCertificate != nil {
		err = c.CertificateConverter.ToHclById(*target.Endpoint.ClusterCertificate, dependencies)

		if err != nil {
			return err
		}
	}

	// Export the environments
	for _, e := range target.EnvironmentIds {
		err = c.EnvironmentConverter.ToHclById(e, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}
