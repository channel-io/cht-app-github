package github

type InstallationContext struct {
	InstallationId int64
	OrgLogin       string
}

func NewInstallationContext(installationId int64, orgLogin string) InstallationContext {
	return InstallationContext{
		InstallationId: installationId,
		OrgLogin:       orgLogin,
	}
}
