package terraform

type TerraformProject struct {
	Type                                   string                                           `hcl:"type,label"`
	Name                                   string                                           `hcl:"name,label"`
	ResourceName                           string                                           `hcl:"name"`
	AutoCreateRelease                      bool                                             `hcl:"auto_create_release"`
	DefaultGuidedFailureMode               *string                                          `hcl:"default_guided_failure_mode"`
	DefaultToSkipIfAlreadyInstalled        bool                                             `hcl:"default_to_skip_if_already_installed"`
	Description                            *string                                          `hcl:"description"`
	DiscreteChannelRelease                 bool                                             `hcl:"discrete_channel_release"`
	IsDisabled                             bool                                             `hcl:"is_disabled"`
	IsVersionControlled                    bool                                             `hcl:"is_version_controlled"`
	LifecycleId                            string                                           `hcl:"lifecycle_id"`
	ProjectGroupId                         string                                           `hcl:"project_group_id"`
	IncludedLibraryVariableSets            []string                                         `hcl:"included_library_variable_sets"`
	TenantedDeploymentParticipation        *string                                          `hcl:"tenanted_deployment_participation"`
	Template                               []TerraformTemplate                              `hcl:"template,block"`
	ConnectivityPolicy                     TerraformConnectivityPolicy                      `hcl:"connectivity_policy,block"`
	GitLibraryPersistenceSettings          *TerraformGitLibraryPersistenceSettings          `hcl:"git_library_persistence_settings,block"`
	GitAnonymousPersistenceSettings        *TerraformGitAnonymousPersistenceSettings        `hcl:"git_anonymous_persistence_settings,block"`
	GitUsernamePasswordPersistenceSettings *TerraformGitUsernamePasswordPersistenceSettings `hcl:"git_username_password_persistence_settings,block"`
}

type TerraformTemplate struct {
	Id              *string           `hcl:"id"`
	Name            *string           `hcl:"name"`
	Label           *string           `hcl:"label"`
	HelpText        *string           `hcl:"help_text"`
	DefaultValue    *string           `hcl:"default_value"`
	DisplaySettings map[string]string `hcl:"display_settings"`
}

type TerraformConnectivityPolicy struct {
	AllowDeploymentsToNoTargets bool   `hcl:"allow_deployments_to_no_targets"`
	ExcludeUnhealthyTargets     bool   `hcl:"exclude_unhealthy_targets"`
	SkipMachineBehavior         string `hcl:"skip_machine_behavior"`
}

type TerraformGitLibraryPersistenceSettings struct {
	GitCredentialId   string   `hcl:"git_credential_id"`
	Url               string   `hcl:"url"`
	BasePath          string   `hcl:"base_path"`
	DefaultBranch     string   `hcl:"default_branch"`
	ProtectedBranches []string `hcl:"protected_branches"`
}

type TerraformGitAnonymousPersistenceSettings struct {
	Url               string   `hcl:"url"`
	BasePath          string   `hcl:"base_path"`
	DefaultBranch     string   `hcl:"default_branch"`
	ProtectedBranches []string `hcl:"protected_branches"`
}

type TerraformGitUsernamePasswordPersistenceSettings struct {
	Url               string   `hcl:"url"`
	Username          string   `hcl:"username"`
	Password          string   `hcl:"password"`
	BasePath          string   `hcl:"base_path"`
	DefaultBranch     string   `hcl:"default_branch"`
	ProtectedBranches []string `hcl:"protected_branches"`
}
