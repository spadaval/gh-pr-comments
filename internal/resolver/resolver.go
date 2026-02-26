package resolver

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

var pullURLRE = regexp.MustCompile(`^/([^/]+)/([^/]+)/pull/([0-9]+)(?:/.*)?$`)

// Identity represents a fully-resolved pull request reference.
type Identity struct {
	Owner  string
	Repo   string
	Host   string
	Number int
	URL    string
}

var runGh = func(args ...string) ([]byte, error) {
	cmd := exec.Command("gh", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(output))
		if msg == "" {
			msg = err.Error()
		}
		return nil, errors.New(msg)
	}
	return output, nil
}

// Resolve infers the pull request identity similarly to `gh pr view`.
//
// Priority:
//  1. explicit selector arg
//  2. --pr flag
//  3. current branch PR via gh default behavior
func Resolve(selector string, prFlag int, repoFlag string) (Identity, error) {
	resolvedSelector, err := normalizeSelector(selector, prFlag)
	if err != nil {
		return Identity{}, err
	}

	args := []string{"pr", "view"}
	if resolvedSelector != "" {
		args = append(args, resolvedSelector)
	}
	if repo := strings.TrimSpace(repoFlag); repo != "" {
		args = append(args, "--repo", repo)
	}
	args = append(args, "--json", "url")

	output, err := runGh(args...)
	if err != nil {
		return Identity{}, fmt.Errorf("resolve pull request via gh pr view: %w", err)
	}

	var payload struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(output, &payload); err != nil {
		return Identity{}, fmt.Errorf("parse gh pr view output: %w", err)
	}
	if strings.TrimSpace(payload.URL) == "" {
		return Identity{}, errors.New("gh pr view did not return a pull request URL")
	}

	identity, err := parsePullURL(payload.URL)
	if err != nil {
		return Identity{}, fmt.Errorf("parse pull request URL from gh pr view: %w", err)
	}
	identity.URL = payload.URL
	return identity, nil
}

func normalizeSelector(selector string, prFlag int) (string, error) {
	selector = strings.TrimSpace(selector)

	if prFlag < 0 {
		return "", errors.New("--pr must be a positive integer")
	}

	if prFlag > 0 {
		if selector == "" {
			return strconv.Itoa(prFlag), nil
		}
		if n, err := strconv.Atoi(selector); err == nil {
			if n != prFlag {
				return "", fmt.Errorf("pull request argument %q does not match --pr=%d", selector, prFlag)
			}
			return selector, nil
		}
		if id, err := parsePullURL(selector); err == nil {
			if id.Number != prFlag {
				return "", fmt.Errorf("pull request argument %q does not match --pr=%d", selector, prFlag)
			}
			return selector, nil
		}
	}

	return selector, nil
}

func parsePullURL(raw string) (Identity, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return Identity{}, err
	}
	if u.Host == "" {
		return Identity{}, errors.New("missing host")
	}
	matches := pullURLRE.FindStringSubmatch(u.Path)
	if matches == nil {
		return Identity{}, errors.New("not a pull request url")
	}
	number, err := strconv.Atoi(matches[3])
	if err != nil || number <= 0 {
		return Identity{}, errors.New("invalid pull request number")
	}
	return Identity{
		Owner:  matches[1],
		Repo:   matches[2],
		Host:   strings.ToLower(u.Hostname()),
		Number: number,
	}, nil
}
