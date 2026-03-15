package pubkeyprovider

import "context"

// Too github specific, will need to be refactoring when adding new providers
type UsersPublicKeysOptions struct {
	TeamName    string
	Username    string
	MinUserRole string
}

type UsersPublicKeysOption func(*UsersPublicKeysOptions)

func WithTeamName(teamName string) UsersPublicKeysOption {
	return func(opts *UsersPublicKeysOptions) {
		opts.TeamName = teamName
	}
}

func WithUsername(username string) UsersPublicKeysOption {
	return func(opts *UsersPublicKeysOptions) {
		opts.Username = username
	}
}

func WithMinUserRole(role string) UsersPublicKeysOption {
	return func(opts *UsersPublicKeysOptions) {
		opts.MinUserRole = role
	}
}

type Provider interface {
	GetUsersPublicKeys(ctx context.Context, orgName string, opts ...UsersPublicKeysOption) ([]string, error)
}
