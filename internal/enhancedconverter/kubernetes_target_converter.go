package enhancedconverter

import (
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/octopus"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/terraform"
	"github.com/mcasperson/OctopusTerraformExport/internal/util"
)

type KubernetesTargetConverter struct {
	Client client.OctopusClient
}

func (c KubernetesTargetConverter) ToHcl(dependencies *ResourceDetailsCollection) error {
	collection := octopus.GeneralCollection[octopus.KubernetesEndpointResource]{}
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

func (c KubernetesTargetConverter) ToHclById(id string, dependencies *ResourceDetailsCollection) error {
	resource := octopus.KubernetesEndpointResource{}
	err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	return c.toHcl(resource, dependencies)
}

func (c KubernetesTargetConverter) toHcl(target octopus.KubernetesEndpointResource, dependencies *ResourceDetailsCollection) error {

	// The machine policies need to be exported
	err := MachinePolicyConverter{
		Client: c.Client,
	}.ToHclById(target.MachinePolicyId, dependencies)

	if err != nil {
		return err
	}

	// Export the accounts
	if target.Endpoint.Authentication.AccountId != nil {
		err = AccountConverter{
			Client: c.Client,
		}.ToHclById(*target.Endpoint.Authentication.AccountId, dependencies)

		if err != nil {
			return err
		}
	}

	// Export the certificate
	if target.Endpoint.Authentication.ClientCertificate != nil {
		err = CertificateConverter{
			Client: c.Client,
		}.ToHclById(*target.Endpoint.Authentication.ClientCertificate, dependencies)

		if err != nil {
			return err
		}
	}

	if target.Endpoint.ClusterCertificate != nil {
		err = CertificateConverter{
			Client: c.Client,
		}.ToHclById(*target.Endpoint.ClusterCertificate, dependencies)

		if err != nil {
			return err
		}
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
	thisResource.Lookup = "${octopusdeploy_project." + targetName + ".id}"
	thisResource.ToHcl = func() (string, error) {
		if target.Endpoint.CommunicationStyle == "Kubernetes" {

			terraformResource := terraform.TerraformKubernetesEndpointResource{
				Type:                            "octopusdeploy_kubernetes_cluster_deployment_target",
				Name:                            targetName,
				ClusterUrl:                      util.EmptyIfNil(target.Endpoint.ClusterUrl),
				Environments:                    c.lookupEnvironments(target.EnvironmentIds, dependencies),
				ResourceName:                    target.Name,
				Roles:                           target.Roles,
				ClusterCertificate:              dependencies.GetResourcePointer("Certificate", target.Endpoint.ClusterCertificate),
				DefaultWorkerPoolId:             c.getWorkerPool(target.Endpoint.DefaultWorkerPoolId, dependencies),
				HealthStatus:                    nil,
				Id:                              nil,
				IsDisabled:                      util.NilIfFalse(target.IsDisabled),
				MachinePolicyId:                 c.getMachinePolicy(target.MachinePolicyId, dependencies),
				Namespace:                       util.NilIfEmptyPointer(target.Endpoint.Namespace),
				OperatingSystem:                 nil,
				ProxyId:                         nil,
				RunningInContainer:              nil,
				ShellName:                       nil,
				ShellVersion:                    nil,
				SkipTlsVerification:             util.ParseBoolPointer(target.Endpoint.SkipTlsVerification),
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
			file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

			return string(file.Bytes()), nil
		}

		return "", nil
	}

	dependencies.AddResource(thisResource)
	return nil
}

func (c KubernetesTargetConverter) GetResourceType() string {
	return "Machines"
}

func (c KubernetesTargetConverter) getAwsAuth(target *octopus.KubernetesEndpointResource, dependencies *ResourceDetailsCollection) *terraform.TerraformAwsAccountAuthentication {
	if target.Endpoint.Authentication.AuthenticationType == "KubernetesAws" {
		return &terraform.TerraformAwsAccountAuthentication{
			AccountId:                 c.getAccount(target.Endpoint.Authentication.AccountId, dependencies),
			ClusterName:               util.EmptyIfNil(target.Endpoint.Authentication.ClusterName),
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

func (c KubernetesTargetConverter) getK8sAuth(target *octopus.KubernetesEndpointResource, dependencies *ResourceDetailsCollection) *terraform.TerraformAccountAuthentication {
	if target.Endpoint.Authentication.AuthenticationType == "KubernetesStandard" {
		return &terraform.TerraformAccountAuthentication{
			AccountId: c.getAccount(target.Endpoint.Authentication.AccountId, dependencies),
		}
	}

	return nil
}

func (c KubernetesTargetConverter) getGoogleAuth(target *octopus.KubernetesEndpointResource, dependencies *ResourceDetailsCollection) *terraform.TerraformGcpAccountAuthentication {
	if target.Endpoint.Authentication.AuthenticationType == "KubernetesGoogleCloud" {
		return &terraform.TerraformGcpAccountAuthentication{
			AccountId:                 c.getAccount(target.Endpoint.Authentication.AccountId, dependencies),
			ClusterName:               util.EmptyIfNil(target.Endpoint.Authentication.ClusterName),
			Project:                   util.EmptyIfNil(target.Endpoint.Authentication.Project),
			ImpersonateServiceAccount: target.Endpoint.Authentication.ImpersonateServiceAccount,
			Region:                    target.Endpoint.Authentication.Region,
			ServiceAccountEmails:      target.Endpoint.Authentication.ServiceAccountEmails,
			Zone:                      target.Endpoint.Authentication.Zone,
			UseVmServiceAccount:       target.Endpoint.Authentication.UseVmServiceAccount,
		}
	}

	return nil
}

func (c KubernetesTargetConverter) getCertAuth(target *octopus.KubernetesEndpointResource, dependencies *ResourceDetailsCollection) *terraform.TerraformCertificateAuthentication {
	if target.Endpoint.Authentication.AuthenticationType == "KubernetesCertificate" {
		return &terraform.TerraformCertificateAuthentication{
			ClientCertificate: dependencies.GetResourcePointer("Certificates", target.Endpoint.Authentication.ClientCertificate),
		}
	}

	return nil
}

func (c KubernetesTargetConverter) getAzureAuth(target *octopus.KubernetesEndpointResource, dependencies *ResourceDetailsCollection) *terraform.TerraformAzureServicePrincipalAuthentication {
	if target.Endpoint.Authentication.AuthenticationType == "KubernetesAzure" {
		return &terraform.TerraformAzureServicePrincipalAuthentication{
			AccountId:            c.getAccount(target.Endpoint.Authentication.AccountId, dependencies),
			ClusterName:          util.EmptyIfNil(target.Endpoint.Authentication.ClusterName),
			ClusterResourceGroup: util.EmptyIfNil(target.Endpoint.Authentication.ClusterResourceGroup),
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