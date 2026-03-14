package pubkeyprovider

import (
	"context"
	"fmt"
	"iter"
	"log/slog"
	"strings"

	"github.com/google/go-github/v84/github"
	"golang.org/x/sync/errgroup"
)

type GitHubProvider struct {
	*github.Client
}

// GetUsersPublicKeys implements [Provider].
func (g *GitHubProvider) GetUsersPublicKeys(ctx context.Context, orgName string, opts ...UsersPublicKeysOption) ([]string, error) {
	if orgName == "" {
		return nil, fmt.Errorf("org name must be provided")
	}

	options := &UsersPublicKeysOptions{}
	for _, opt := range opts {
		opt(options)
	}

	usrRole := options.MinUserRole
	if usrRole == "" {
		usrRole = "member"
	}

	if options.Username != "" {
		isValid, err := g.validateOrgMember(ctx, orgName, options)
		if err != nil {
			return nil, fmt.Errorf("failed to validate org membership for user %s: %w", options.Username, err)
		}

		if !isValid {
			return nil, fmt.Errorf("user %s does not meet the required role in org %s", options.Username, orgName)
		}

		return g.getUserKeys(ctx, options)
	}

	if options.TeamName != "" {
		return g.getTeamKeys(ctx, orgName, options, usrRole)
	}

	g.Organizations.ListMembersIter(ctx, orgName, &github.ListMembersOptions{
		PublicOnly: false,
		Role:       usrRole,
	})

	return nil, fmt.Errorf("either username or team name must be provided")

}

func (g *GitHubProvider) getUserGroupKeys(ctx context.Context, orgName string, options *UsersPublicKeysOptions, usrRole string) ([]string, error) {
	team, ghr, err := g.Teams.GetTeamBySlug(ctx, orgName, options.TeamName)
	if err != nil {
		return nil, fmt.Errorf("failed to get team %s in org %s: %w", options.TeamName, orgName, err)
	}

	slog.Debug("github response", "response", ghr)

	if team == nil {
		return nil, fmt.Errorf("team %s not found in org %s", options.TeamName, orgName)
	}

	users := g.Teams.ListTeamMembersBySlugIter(ctx, orgName, options.TeamName, &github.TeamListTeamMembersOptions{
		Role:        usrRole,
		ListOptions: github.ListOptions{},
	})

	return g.getUsersKeys(ctx, orgName, users, options)
}

func (g *GitHubProvider) getUsersKeys(ctx context.Context, orgName string, users iter.Seq2[*github.User, error], options *UsersPublicKeysOptions) ([]string, error) {
	teamKeys := []string{}
	keys := make(chan string, 100)
	go func() {
		for key := range keys {
			teamKeys = append(teamKeys, key)
		}
	}()

	eg, ctx := errgroup.WithContext(ctx)
	for user, err := range users {
		if err != nil {
			return nil, fmt.Errorf("failed to list members of team %s in org %s: %w", options.TeamName, orgName, err)
		}

		slog.Debug("Fetched team member", "team", options.TeamName, "username", user.GetLogin())

		eg.Go(func() error {
			uk, err := g.getUserKeys(ctx, &UsersPublicKeysOptions{
				Username: user.GetLogin(),
			})
			if err != nil {
				return fmt.Errorf("failed to get keys for user %s: %w", user.GetLogin(), err)
			}

			for _, key := range uk {
				keys <- key
			}

			return nil
		})
	}

	err := eg.Wait()
	if err != nil {
		return nil, fmt.Errorf("error fetching keys for team members: %w", err)
	}

	close(keys)

	return teamKeys, nil
}

func (g *GitHubProvider) getTeamKeys(ctx context.Context, orgName string, options *UsersPublicKeysOptions, usrRole string) ([]string, error) {
	team, ghr, err := g.Teams.GetTeamBySlug(ctx, orgName, options.TeamName)
	if err != nil {
		return nil, fmt.Errorf("failed to get team %s in org %s: %w", options.TeamName, orgName, err)
	}

	slog.Debug("github response", "response", ghr)

	if team == nil {
		return nil, fmt.Errorf("team %s not found in org %s", options.TeamName, orgName)
	}

	users := g.Teams.ListTeamMembersBySlugIter(ctx, orgName, options.TeamName, &github.TeamListTeamMembersOptions{
		Role:        usrRole,
		ListOptions: github.ListOptions{},
	})

	return g.getUsersKeys(ctx, orgName, users, options)
}

func (g *GitHubProvider) validateOrgMember(ctx context.Context, orgName string, options *UsersPublicKeysOptions) (bool, error) {
	ms, ghr, err := g.Organizations.GetOrgMembership(ctx, options.Username, options.Username)
	if err != nil {
		return false, fmt.Errorf("failed to get org membership for user %s: %w", options.Username, err)
	}

	slog.Debug("github response", "response", ghr)

	if ms == nil {
		return false, fmt.Errorf("user %s is not a member of org %s", options.Username, orgName)
	}

	if ms.GetState() != "active" {
		return false, fmt.Errorf("user %s is not an active member of org %s", options.Username, orgName)
	}

	if strings.EqualFold(ms.GetRole(), options.MinUserRole) || strings.EqualFold(ms.GetRole(), "admin") {
		return false, fmt.Errorf("user %s does not have the required role in org %s", options.Username, orgName)
	}

	slog.Debug("User is an active member of the org with the required role", "username", options.Username, "org", orgName, "user_role", ms.GetRole())
	return true, nil
}

func (g *GitHubProvider) getUserKeys(ctx context.Context, options *UsersPublicKeysOptions) ([]string, error) {
	keys := []string{}
	for key, err := range g.Users.ListKeysIter(ctx, options.Username, nil) {
		if err != nil {
			return nil, fmt.Errorf("failed to list keys for user %s: %w", options.Username, err)
		}
		slog.Debug("Fetched key for user", "username", options.Username, "key_id", key.GetID(), "key_title", key.GetTitle())
		keys = append(keys, key.GetKey())
	}

	return keys, nil
}

var _ Provider = (*GitHubProvider)(nil)

func NewGitHubProvider(token string) *GitHubProvider {

	ghc := github.NewClient(nil).WithAuthToken(token)

	return &GitHubProvider{
		Client: ghc,
	}
}
