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
)

type CertificateConverter struct {
	Client client.OctopusClient
}

func (c CertificateConverter) ToHcl(dependencies *ResourceDetailsCollection) error {
	collection := octopus2.GeneralCollection[octopus2.Certificate]{}
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

func (c CertificateConverter) ToHclById(id string, dependencies *ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	certificate := octopus2.Certificate{}
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &certificate)

	if err != nil {
		return err
	}

	return c.toHcl(certificate, true, dependencies)
}

func (c CertificateConverter) toHcl(certificate octopus2.Certificate, recursive bool, dependencies *ResourceDetailsCollection) error {
	/*
		Note we don't export the tenants or environments that this certificate might be exposed to.
		It is assumed the exported project links up all required environments, and the certificate
		will link itself to any available environments or tenants.
	*/

	certificateName := "certificate_" + sanitizer.SanitizeName(certificate.Name)

	thisResource := ResourceDetails{}
	thisResource.FileName = "space_population/" + certificateName + ".tf"
	thisResource.Id = certificate.Id
	thisResource.ResourceType = c.GetResourceType()
	thisResource.Lookup = "${octopusdeploy_certificate." + certificateName + ".id}"
	thisResource.ToHcl = func() (string, error) {
		terraformResource := terraform2.TerraformCertificate{
			Type:            "octopusdeploy_certificate",
			Name:            certificateName,
			SpaceId:         nil,
			ResourceName:    certificate.Name,
			Password:        "${var." + certificateName + "_password}",
			CertificateData: "${var." + certificateName + "_data}",
			Archived:        &certificate.Archived,
			//CertificateDataFormat:           certificate.CertificateDataFormat,
			Environments: c.lookupEnvironments(certificate.EnvironmentIds, dependencies),
			//HasPrivateKey:                   certificate.HasPrivateKey,
			//IsExpired:                       certificate.IsExpired,
			//IssuerCommonName:                certificate.IssuerCommonName,
			//IssuerDistinguishedName:         certificate.IssuerDistinguishedName,
			//IssuerOrganization:              certificate.IssuerOrganization,
			//NotAfter:                        certificate.NotAfter,
			//NotBefore:                       certificate.NotBefore,
			Notes: &certificate.Notes,
			//ReplacedBy:                      nil, // ReplacedBy does not seem to be used
			//SelfSigned:                      certificate.SelfSigned,
			//SerialNumber:                    certificate.SerialNumber,
			//SignatureAlgorithmName:          certificate.SignatureAlgorithmName,
			//SubjectAlternativeNames:         certificate.SubjectAlternativeNames,
			//SubjectCommonName:               certificate.SubjectCommonName,
			//SubjectDistinguishedName:        certificate.SubjectDistinguishedName,
			//SubjectOrganization:             certificate.SubjectOrganization,
			TenantTags:                      &certificate.TenantTags,
			TenantedDeploymentParticipation: &certificate.TenantedDeploymentParticipation,
			Tenants:                         c.lookupTenants(certificate.TenantIds, dependencies),
			//Thumbprint:                      certificate.Thumbprint,
			//Version:                         certificate.Version,
		}
		file := hclwrite.NewEmptyFile()

		// Add a comment with the import command
		baseUrl, _ := c.Client.GetSpaceBaseUrl()
		file.Body().AppendUnstructuredTokens([]*hclwrite.Token{{
			Type: hclsyntax.TokenComment,
			Bytes: []byte("# Import existing resources with the following commands:\n" +
				"# RESOURCE_ID=$(curl -H \"X-Octopus-ApiKey: ${OCTOPUS_CLI_API_KEY}\" " + baseUrl + "/" + c.GetResourceType() + " | jq -r '.Items[] | select(.Name==\"" + certificate.Name + "\") | .Id')\n" +
				"# terraform import octopusdeploy_certificate." + certificateName + " ${RESOURCE_ID}\n"),
			SpacesBefore: 0,
		}})

		file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

		defaultPassword := ""
		certificatePassword := terraform2.TerraformVariable{
			Name:        certificateName + "_password",
			Type:        "string",
			Nullable:    true,
			Sensitive:   true,
			Description: "The password used by the certificate " + certificate.Name,
			Default:     &defaultPassword,
		}

		block := gohcl.EncodeAsBlock(certificatePassword, "variable")
		hcl.WriteUnquotedAttribute(block, "type", "string")
		file.Body().AppendBlock(block)

		certificateData := terraform2.TerraformVariable{
			Name:        certificateName + "_data",
			Type:        "string",
			Nullable:    false,
			Sensitive:   true,
			Description: "The certificate data used by the certificate " + certificate.Name,
		}

		block = gohcl.EncodeAsBlock(certificateData, "variable")
		hcl.WriteUnquotedAttribute(block, "type", "string")
		file.Body().AppendBlock(block)

		return string(file.Bytes()), nil
	}

	dependencies.AddResource(thisResource)
	return nil
}

func (c CertificateConverter) GetResourceType() string {
	return "Certificates"
}

func (c CertificateConverter) lookupEnvironments(envs []string, dependencies *ResourceDetailsCollection) []string {
	newEnvs := make([]string, 0)
	for _, v := range envs {
		environment := dependencies.GetResource("Environments", v)
		if environment != "" {
			newEnvs = append(newEnvs, environment)
		}
	}
	return newEnvs
}

func (c CertificateConverter) lookupTenants(envs []string, dependencies *ResourceDetailsCollection) []string {
	newTenants := make([]string, 0)
	for _, v := range envs {
		tenant := dependencies.GetResource("Tenants", v)
		if tenant != "" {
			newTenants = append(newTenants, tenant)
		}
	}
	return newTenants
}
