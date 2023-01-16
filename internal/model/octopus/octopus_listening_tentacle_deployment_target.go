package octopus

type ListeningEndpointResource struct {
	Id                              string
	Name                            string
	EnvironmentIds                  []string
	Roles                           []string
	TenantIds                       []string
	TenantTags                      []string
	TenantedDeploymentParticipation string
	Thumbprint                      *string
	Uri                             *string
	IsDisabled                      bool
	MachinePolicyId                 string
	HealthStatus                    string
	HasLatestCalamari               bool
	StatusSummary                   string
	IsInProcess                     bool
	OperatingSystem                 string
	ShellName                       string
	ShellVersion                    string
	Architecture                    string
	Endpoint                        ListeningTentacleEndpointResource
}

// ListeningTentacleEndpointResource is based on ListeningTentacleEndpointResource from the client library
type ListeningTentacleEndpointResource struct {
	CommunicationStyle string
	Uri                string
	ProxyId            string
}
