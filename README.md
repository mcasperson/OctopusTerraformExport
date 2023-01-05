# Octopus Terraform Exporter

This app exports an Octopus space to the associated Terraform resources for use with the 
[Octopus Terraform Provider](https://registry.terraform.io/providers/OctopusDeployLabs/octopusdeploy).

## Downloads

Get the compiled binaries from the [releases](https://github.com/mcasperson/OctopusTerraformExport/releases).

## Usage

```
./octoterra -url https://yourinstance.octopus.app -space Spaces-## -apiKey API-APIKEYGOESHERE
```

## To Do

The following resources have yet to be exported:

* octopusdeploy_tenant
* octopusdeploy_certificate
* octopusdeploy_tag
* octopusdeploy_tag_set
* octopusdeploy_tenant_common_variable
* octopusdeploy_tenant_project_variable

* octopusdeploy_git_credential
* octopusdeploy_machine_policy
* octopusdeploy_project_deployment_target_trigger
* octopusdeploy_scoped_user_role
* octopusdeploy_script_module
* octopusdeploy_team
* octopusdeploy_user
* octopusdeploy_user_role

