package model

import "strings"

type Manager struct {
	ID                 string
	Name               string
	Email              *string
	GithubUsername     *string
	GithubOrganization *string
}

func (m *Manager) GetEmailLocalPart() *string {
	if m.Email != nil && strings.Contains(*m.Email, "@") {
		return &strings.Split(*m.Email, "@")[0]
	}
	return nil
}
