package cmd

import (
	"github.com/spf13/cobra"

	"github.com/agynio/gh-pr-review/internal/comments"
	"github.com/agynio/gh-pr-review/internal/resolver"
)

type listOptions struct {
	Repo     string
	Pull     int
	Selector string
}

func newListCommand() *cobra.Command {
	opts := &listOptions{}

	cmd := &cobra.Command{
		Use:   "list [<number> | <url>]",
		Short: "List inline pull request review comments",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.Selector = args[0]
			}
			return runList(cmd, opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Repo, "repo", "R", "", "Repository in 'owner/repo' format")
	cmd.Flags().IntVar(&opts.Pull, "pr", 0, "Pull request number")

	return cmd
}

func runList(cmd *cobra.Command, opts *listOptions) error {
	identity, err := resolver.Resolve(opts.Selector, opts.Pull, opts.Repo)
	if err != nil {
		return err
	}

	service := comments.NewService(apiClientFactory(identity.Host))
	threads, err := service.List(identity)
	if err != nil {
		return err
	}

	return encodeJSON(cmd, map[string]interface{}{
		"pull_request": map[string]interface{}{
			"owner":  identity.Owner,
			"repo":   identity.Repo,
			"host":   identity.Host,
			"number": identity.Number,
			"url":    identity.URL,
		},
		"threads": threads,
	})
}
