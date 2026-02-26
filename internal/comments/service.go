package comments

import (
	"errors"
	"fmt"
	"strings"

	"github.com/agynio/gh-pr-review/internal/ghcli"
	"github.com/agynio/gh-pr-review/internal/resolver"
)

const listThreadsQuery = `query PullRequestInlineComments($owner: String!, $name: String!, $number: Int!, $firstThreads: Int!, $firstComments: Int!) {
  repository(owner: $owner, name: $name) {
    pullRequest(number: $number) {
      reviewThreads(first: $firstThreads) {
        nodes {
          id
          path
          line
          startLine
          isResolved
          isOutdated
          comments(first: $firstComments) {
            nodes {
              id
              body
              createdAt
              url
              author { login }
            }
          }
        }
      }
    }
  }
}`

const pullRequestNodeQuery = `query PullRequestNode($owner: String!, $name: String!, $number: Int!) {
  repository(owner: $owner, name: $name) {
    pullRequest(number: $number) {
      id
    }
  }
}`

const createThreadMutation = `mutation AddPullRequestReviewThread($input: AddPullRequestReviewThreadInput!) {
  addPullRequestReviewThread(input: $input) {
    thread {
      id
      path
      line
      startLine
      isResolved
      isOutdated
      comments(first: 1) {
        nodes {
          id
          body
          createdAt
          url
          author { login }
        }
      }
    }
  }
}`

const (
	defaultFirstThreads  = 100
	defaultFirstComments = 100
)

// Service provides inline pull request comment operations.
type Service struct {
	API ghcli.API
}

// NewService constructs a comment service with the provided API client.
func NewService(api ghcli.API) *Service {
	return &Service{API: api}
}

// Comment represents one inline PR review comment.
type Comment struct {
	ID        string `json:"id"`
	Body      string `json:"body"`
	Author    string `json:"author"`
	CreatedAt string `json:"created_at"`
	URL       string `json:"url"`
}

// Thread represents an inline review thread on a PR diff.
type Thread struct {
	ID         string    `json:"id"`
	Path       string    `json:"path"`
	Line       *int      `json:"line,omitempty"`
	StartLine  *int      `json:"start_line,omitempty"`
	IsResolved bool      `json:"is_resolved"`
	IsOutdated bool      `json:"is_outdated"`
	Comments   []Comment `json:"comments"`
}

// CreateInput holds parameters for creating an inline comment thread.
type CreateInput struct {
	Path      string
	Line      int
	Side      string
	StartLine *int
	StartSide *string
	Body      string
}

// CreateResult returns normalized details for a newly-created inline comment thread.
type CreateResult struct {
	ThreadID    string `json:"thread_id"`
	CommentID   string `json:"comment_id"`
	Path        string `json:"path"`
	Line        *int   `json:"line,omitempty"`
	StartLine   *int   `json:"start_line,omitempty"`
	Author      string `json:"author"`
	Body        string `json:"body"`
	CreatedAt   string `json:"created_at"`
	URL         string `json:"url"`
	IsResolved  bool   `json:"is_resolved"`
	IsOutdated  bool   `json:"is_outdated"`
	RequestedOn string `json:"requested_side"`
}

// List fetches inline review threads/comments for a pull request.
func (s *Service) List(pr resolver.Identity) ([]Thread, error) {
	variables := map[string]interface{}{
		"owner":         pr.Owner,
		"name":          pr.Repo,
		"number":        pr.Number,
		"firstThreads":  defaultFirstThreads,
		"firstComments": defaultFirstComments,
	}

	var response struct {
		Repository *struct {
			PullRequest *struct {
				ReviewThreads struct {
					Nodes []struct {
						ID         string `json:"id"`
						Path       string `json:"path"`
						Line       *int   `json:"line"`
						StartLine  *int   `json:"startLine"`
						IsResolved bool   `json:"isResolved"`
						IsOutdated bool   `json:"isOutdated"`
						Comments   struct {
							Nodes []struct {
								ID        string `json:"id"`
								Body      string `json:"body"`
								CreatedAt string `json:"createdAt"`
								URL       string `json:"url"`
								Author    *struct {
									Login string `json:"login"`
								} `json:"author"`
							} `json:"nodes"`
						} `json:"comments"`
					} `json:"nodes"`
				} `json:"reviewThreads"`
			} `json:"pullRequest"`
		} `json:"repository"`
	}

	if err := s.API.GraphQL(listThreadsQuery, variables, &response); err != nil {
		return nil, err
	}

	if response.Repository == nil || response.Repository.PullRequest == nil {
		return nil, errors.New("pull request not found or inaccessible")
	}

	nodes := response.Repository.PullRequest.ReviewThreads.Nodes
	threads := make([]Thread, 0, len(nodes))
	for _, node := range nodes {
		thread := Thread{
			ID:         node.ID,
			Path:       node.Path,
			Line:       node.Line,
			StartLine:  node.StartLine,
			IsResolved: node.IsResolved,
			IsOutdated: node.IsOutdated,
			Comments:   make([]Comment, 0, len(node.Comments.Nodes)),
		}

		for _, c := range node.Comments.Nodes {
			if c.Author == nil || strings.TrimSpace(c.Author.Login) == "" {
				return nil, errors.New("comment missing author")
			}
			thread.Comments = append(thread.Comments, Comment{
				ID:        c.ID,
				Body:      c.Body,
				Author:    c.Author.Login,
				CreatedAt: c.CreatedAt,
				URL:       c.URL,
			})
		}

		threads = append(threads, thread)
	}

	return threads, nil
}

// Create opens a new inline review thread with one comment on the given PR.
func (s *Service) Create(pr resolver.Identity, input CreateInput) (CreateResult, error) {
	path := strings.TrimSpace(input.Path)
	body := strings.TrimSpace(input.Body)
	if path == "" {
		return CreateResult{}, errors.New("path is required")
	}
	if input.Line <= 0 {
		return CreateResult{}, errors.New("line must be greater than zero")
	}
	if body == "" {
		return CreateResult{}, errors.New("body is required")
	}

	side, err := normalizeSide(input.Side)
	if err != nil {
		return CreateResult{}, err
	}

	prID, err := s.pullRequestNodeID(pr)
	if err != nil {
		return CreateResult{}, err
	}

	mutationInput := map[string]interface{}{
		"pullRequestId": prID,
		"path":          path,
		"line":          input.Line,
		"side":          side,
		"body":          body,
	}

	if input.StartLine != nil {
		if *input.StartLine <= 0 {
			return CreateResult{}, errors.New("start-line must be greater than zero")
		}
		mutationInput["startLine"] = *input.StartLine
	}
	if input.StartSide != nil {
		normalizedStartSide, err := normalizeSide(*input.StartSide)
		if err != nil {
			return CreateResult{}, fmt.Errorf("invalid start-side: %w", err)
		}
		mutationInput["startSide"] = normalizedStartSide
	}

	var response struct {
		AddPullRequestReviewThread struct {
			Thread *struct {
				ID         string `json:"id"`
				Path       string `json:"path"`
				Line       *int   `json:"line"`
				StartLine  *int   `json:"startLine"`
				IsResolved bool   `json:"isResolved"`
				IsOutdated bool   `json:"isOutdated"`
				Comments   struct {
					Nodes []struct {
						ID        string `json:"id"`
						Body      string `json:"body"`
						CreatedAt string `json:"createdAt"`
						URL       string `json:"url"`
						Author    *struct {
							Login string `json:"login"`
						} `json:"author"`
					} `json:"nodes"`
				} `json:"comments"`
			} `json:"thread"`
		} `json:"addPullRequestReviewThread"`
	}

	if err := s.API.GraphQL(createThreadMutation, map[string]interface{}{"input": mutationInput}, &response); err != nil {
		return CreateResult{}, err
	}

	thread := response.AddPullRequestReviewThread.Thread
	if thread == nil {
		return CreateResult{}, errors.New("create response missing thread")
	}
	if len(thread.Comments.Nodes) == 0 {
		return CreateResult{}, errors.New("create response missing comment")
	}

	comment := thread.Comments.Nodes[0]
	if comment.Author == nil || strings.TrimSpace(comment.Author.Login) == "" {
		return CreateResult{}, errors.New("create response missing comment author")
	}

	return CreateResult{
		ThreadID:    thread.ID,
		CommentID:   comment.ID,
		Path:        thread.Path,
		Line:        thread.Line,
		StartLine:   thread.StartLine,
		Author:      comment.Author.Login,
		Body:        comment.Body,
		CreatedAt:   comment.CreatedAt,
		URL:         comment.URL,
		IsResolved:  thread.IsResolved,
		IsOutdated:  thread.IsOutdated,
		RequestedOn: side,
	}, nil
}

func (s *Service) pullRequestNodeID(pr resolver.Identity) (string, error) {
	variables := map[string]interface{}{
		"owner":  pr.Owner,
		"name":   pr.Repo,
		"number": pr.Number,
	}

	var response struct {
		Repository *struct {
			PullRequest *struct {
				ID string `json:"id"`
			} `json:"pullRequest"`
		} `json:"repository"`
	}

	if err := s.API.GraphQL(pullRequestNodeQuery, variables, &response); err != nil {
		return "", err
	}
	if response.Repository == nil || response.Repository.PullRequest == nil {
		return "", errors.New("pull request not found or inaccessible")
	}
	id := strings.TrimSpace(response.Repository.PullRequest.ID)
	if id == "" {
		return "", errors.New("pull request id missing from response")
	}
	return id, nil
}

func normalizeSide(side string) (string, error) {
	s := strings.ToUpper(strings.TrimSpace(side))
	switch s {
	case "LEFT", "RIGHT":
		return s, nil
	case "":
		return "", errors.New("side is required")
	default:
		return "", fmt.Errorf("invalid side %q: must be LEFT or RIGHT", side)
	}
}
