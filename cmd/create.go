package cmd

import (
	"github.com/spf13/cobra"

	"github.com/agynio/gh-pr-review/internal/comments"
	"github.com/agynio/gh-pr-review/internal/resolver"
)

type createOptions struct {
	Repo      string
	Pull      int
	Selector  string
	Path      string
	Line      int
	Side      string
	StartLine int
	StartSide string
	Body      string
}

func newCreateCommand() *cobra.Command {
	opts := &createOptions{Side: "RIGHT"}

	cmd := &cobra.Command{
		Use:   "create [<number> | <url>]",
		Short: "Create an inline pull request review comment",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.Selector = args[0]
			}
			return runCreate(cmd, opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Repo, "repo", "R", "", "Repository in 'owner/repo' format")
	cmd.Flags().IntVar(&opts.Pull, "pr", 0, "Pull request number")
	cmd.Flags().StringVar(&opts.Path, "path", "", "File path for inline comment")
	cmd.Flags().IntVar(&opts.Line, "line", 0, "Line number for inline comment")
	cmd.Flags().StringVar(&opts.Side, "side", opts.Side, "Diff side (LEFT or RIGHT)")
	cmd.Flags().IntVar(&opts.StartLine, "start-line", 0, "Start line for multi-line comments")
	cmd.Flags().StringVar(&opts.StartSide, "start-side", "", "Start side for multi-line comments (LEFT or RIGHT)")
	cmd.Flags().StringVar(&opts.Body, "body", "", "Comment body")
	_ = cmd.MarkFlagRequired("path")
	_ = cmd.MarkFlagRequired("line")
	_ = cmd.MarkFlagRequired("body")

	return cmd
}

func runCreate(cmd *cobra.Command, opts *createOptions) error {
	identity, err := resolver.Resolve(opts.Selector, opts.Pull, opts.Repo)
	if err != nil {
		return err
	}

	service := comments.NewService(apiClientFactory(identity.Host))

	var startLine *int
	if opts.StartLine > 0 {
		startLine = &opts.StartLine
	}
	var startSide *string
	if opts.StartSide != "" {
		startSide = &opts.StartSide
	}

	created, err := service.Create(identity, comments.CreateInput{
		Path:      opts.Path,
		Line:      opts.Line,
		Side:      opts.Side,
		StartLine: startLine,
		StartSide: startSide,
		Body:      opts.Body,
	})
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
		"comment": created,
	})
}
