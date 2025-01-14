/*
Copyright 2019 The Tekton Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-github/github"
)

func TestFakeGitHubPullRequest(t *testing.T) {
	ctx := context.Background()
	gh := NewFakeGitHub()
	client, close := githubClient(t, gh)
	defer close()

	if _, resp, err := client.PullRequests.Get(ctx, owner, repo, prNum); err == nil || resp.StatusCode != http.StatusNotFound {
		t.Fatalf("Get PullRequest: wanted not found, got %+v, %v", resp, err)
	}
	gh.AddPullRequest(pr)

	got, resp, err := client.PullRequests.Get(ctx, owner, repo, prNum)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("Get PullRequest: wanted OK, got %+v, %v", resp, err)
	}
	if diff := cmp.Diff(pr, got); diff != "" {
		t.Errorf("Get PullRequest: -want +got: %s", diff)
	}
}

func TestFakeGitHubComments(t *testing.T) {
	ctx := context.Background()
	gh := NewFakeGitHub()
	client, close := githubClient(t, gh)
	defer close()

	if got, resp, err := client.Issues.ListComments(ctx, owner, repo, prNum, nil); err != nil || resp.StatusCode != http.StatusOK || len(got) != 0 {
		t.Fatalf("List Issues: wanted [], got %+v, %+v, %v", got, resp, err)
	}

	if _, _, err := client.Issues.CreateComment(ctx, owner, repo, prNum, comment); err != nil {
		t.Fatalf("CreateComment: %v", err)
	}

	got, resp, err := client.Issues.ListComments(ctx, owner, repo, prNum, nil)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("List Issues: wanted OK, got %+v, %v", resp, err)
	}
	want := []*github.IssueComment{comment}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("List Issues: -want +got: %s", diff)
	}
}

func TestFakeGitHubBadKey(t *testing.T) {
	gh := NewFakeGitHub()
	s := httptest.NewServer(gh)
	defer s.Close()

	if resp, err := http.Get(fmt.Sprintf("%s/repos/1/2/pulls/foo", s.URL)); err != nil || resp.StatusCode != http.StatusBadRequest {
		t.Errorf("want BadRequest, got %+v, %v", resp, err)
	}
}

func TestFakeGitHubStatus(t *testing.T) {
	ctx := context.Background()
	gh := NewFakeGitHub()
	client, close := githubClient(t, gh)
	defer close()

	sha := "tacocat"

	if got, resp, err := client.Repositories.GetCombinedStatus(ctx, owner, repo, sha, nil); err != nil || resp.StatusCode != http.StatusOK || len(got.Statuses) != 0 {
		t.Fatalf("GetCombinedStatus: wanted [], got %+v, %+v, %v", got, resp, err)
	}

	rs := &github.RepoStatus{
		Context:     github.String("Tekton"),
		Description: github.String("Test all the things!"),
		State:       github.String("success"),
		TargetURL:   github.String("https://tekton.dev"),
	}
	if _, _, err := client.Repositories.CreateStatus(ctx, owner, repo, sha, rs); err != nil {
		t.Fatalf("CreateStatus: %v", err)
	}

	got, resp, err := client.Repositories.GetCombinedStatus(ctx, owner, repo, sha, nil)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("GetCombinedStatus: wanted OK, got %+v, %v", resp, err)
	}
	want := &github.CombinedStatus{
		TotalCount: github.Int(1),
		Statuses:   []github.RepoStatus{*rs},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("GetCombinedStatus: -want +got: %s", diff)
	}
}

func TestFakeGitHubLabels(t *testing.T) {
	ctx := context.Background()
	gh := NewFakeGitHub()
	client, close := githubClient(t, gh)
	defer close()
	gh.AddPullRequest(pr)

	if got, resp, err := client.Issues.ListLabelsByIssue(ctx, owner, repo, prNum, nil); err != nil || len(got) != 0 {
		t.Fatalf("List Issues: wanted [], got %+v, %+v, %v", got, resp, err)
	}

	check := func(labels ...string) {
		got, resp, err := client.Issues.ListLabelsByIssue(ctx, owner, repo, prNum, nil)
		if err != nil {
			t.Fatalf("List Issues: wanted OK, got %+v, %v", resp, err)
		}
		want := make([]*github.Label, 0, len(labels))
		for _, l := range labels {
			want = append(want, &github.Label{
				Name: github.String(l),
			})
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("List Issues: -want +got: %s", diff)
		}
	}

	if _, _, err := client.Issues.AddLabelsToIssue(ctx, owner, repo, prNum, []string{"a"}); err != nil {
		t.Fatalf("AddLabelsToIssue(a): %v", err)
	}
	check("a")

	if _, _, err := client.Issues.AddLabelsToIssue(ctx, owner, repo, prNum, []string{"b"}); err != nil {
		t.Fatalf("AddLabelsToIssue(b): %v", err)
	}
	check("a", "b")

	if _, _, err := client.Issues.ReplaceLabelsForIssue(ctx, owner, repo, prNum, []string{"c"}); err != nil {
		t.Fatalf("ReplaceLabelsForIssue: %v", err)
	}
	check("c")
}
