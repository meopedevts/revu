package github

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

type fakeExec struct {
	stdout  []byte
	stderr  []byte
	err     error
	gotName string
	gotArgs []string
}

func (f *fakeExec) Run(_ context.Context, name string, args ...string) ([]byte, []byte, error) {
	f.gotName = name
	f.gotArgs = append([]string(nil), args...)
	return f.stdout, f.stderr, f.err
}

func loadFixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return b
}

func TestAuthStatus(t *testing.T) {
	tests := []struct {
		name    string
		stderr  []byte
		runErr  error
		wantErr error
	}{
		{"ok", nil, nil, nil},
		{"auth expired", []byte("You are not logged into any GitHub hosts"), errors.New("exit 1"), ErrAuthExpired},
		{"rate limit", []byte("API rate limit exceeded for user"), errors.New("exit 1"), ErrRateLimited},
		{"transient net", []byte("Could not resolve host: api.github.com"), errors.New("exit 1"), ErrTransient},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fe := &fakeExec{stderr: tt.stderr, err: tt.runErr}
			c := NewClient(fe)
			err := c.AuthStatus(context.Background())
			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("want nil err, got %v", err)
				}
			} else {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("want errors.Is(%v), got %v", tt.wantErr, err)
				}
			}
			if fe.gotName != "gh" {
				t.Fatalf("want cmd gh, got %s", fe.gotName)
			}
			wantArgs := []string{"auth", "status"}
			if !reflect.DeepEqual(fe.gotArgs, wantArgs) {
				t.Fatalf("want args %v, got %v", wantArgs, fe.gotArgs)
			}
		})
	}
}

func TestListReviewRequested_Happy(t *testing.T) {
	fe := &fakeExec{stdout: loadFixture(t, "search_prs.json")}
	c := NewClient(fe)
	got, err := c.ListReviewRequested(context.Background())
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 prs, got %d", len(got))
	}
	want0 := PRSummary{
		ID:        "octocat/hello-world#142",
		Number:    142,
		Repo:      "octocat/hello-world",
		Title:     "feat: add foo",
		URL:       "https://github.com/octocat/hello-world/pull/142",
		Author:    "alice",
		IsDraft:   false,
		UpdatedAt: time.Date(2026, 4, 23, 14, 30, 0, 0, time.UTC),
	}
	if !reflect.DeepEqual(got[0], want0) {
		t.Fatalf("pr[0] mismatch:\nwant %+v\ngot  %+v", want0, got[0])
	}
	if got[1].ID != "acme/widgets#7" || !got[1].IsDraft {
		t.Fatalf("pr[1] mismatch: %+v", got[1])
	}
	wantArgs := []string{
		"search", "prs",
		"--review-requested=@me",
		"--state=open",
		"--json", "number,title,url,repository,author,isDraft,updatedAt",
		"--limit", "100",
	}
	if !reflect.DeepEqual(fe.gotArgs, wantArgs) {
		t.Fatalf("args mismatch:\nwant %v\ngot  %v", wantArgs, fe.gotArgs)
	}
}

func TestListReviewRequested_Errors(t *testing.T) {
	tests := []struct {
		name    string
		stdout  []byte
		stderr  []byte
		runErr  error
		wantErr error
	}{
		{"auth expired", nil, []byte("gh: authentication required"), errors.New("exit 1"), ErrAuthExpired},
		{"rate limit", nil, []byte("API rate limit exceeded"), errors.New("exit 1"), ErrRateLimited},
		{"transient", nil, []byte("Could not resolve host"), errors.New("exit 1"), ErrTransient},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fe := &fakeExec{stdout: tt.stdout, stderr: tt.stderr, err: tt.runErr}
			c := NewClient(fe)
			_, err := c.ListReviewRequested(context.Background())
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("want errors.Is(%v), got %v", tt.wantErr, err)
			}
		})
	}
}

func TestListReviewRequested_MalformedJSON(t *testing.T) {
	fe := &fakeExec{stdout: []byte(`{"not an array`)}
	c := NewClient(fe)
	_, err := c.ListReviewRequested(context.Background())
	if err == nil {
		t.Fatal("want decode error")
	}
	if errors.Is(err, ErrAuthExpired) || errors.Is(err, ErrRateLimited) || errors.Is(err, ErrTransient) {
		t.Fatalf("should not be sentinel, got %v", err)
	}
}

func TestGetPRDetails_Happy(t *testing.T) {
	fe := &fakeExec{stdout: loadFixture(t, "pr_view.json")}
	c := NewClient(fe)
	url := "https://github.com/octocat/hello-world/pull/142"
	got, err := c.GetPRDetails(context.Background(), url)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	want := &PRDetails{Additions: 142, Deletions: 37, State: "OPEN", MergedAt: nil, IsDraft: false}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("want %+v, got %+v", want, got)
	}
	wantArgs := []string{"pr", "view", url, "--json", "additions,deletions,state,mergedAt,isDraft"}
	if !reflect.DeepEqual(fe.gotArgs, wantArgs) {
		t.Fatalf("args mismatch:\nwant %v\ngot  %v", wantArgs, fe.gotArgs)
	}
}

func TestGetPRDetails_Errors(t *testing.T) {
	tests := []struct {
		name    string
		stderr  []byte
		wantErr error
	}{
		{"auth expired", []byte("You are not logged in"), ErrAuthExpired},
		{"rate limit", []byte("API rate limit exceeded"), ErrRateLimited},
		{"transient", []byte("connection refused"), ErrTransient},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fe := &fakeExec{stderr: tt.stderr, err: errors.New("exit 1")}
			c := NewClient(fe)
			_, err := c.GetPRDetails(context.Background(), "x")
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("want errors.Is(%v), got %v", tt.wantErr, err)
			}
		})
	}
}

func TestGetPRDetails_MalformedJSON(t *testing.T) {
	fe := &fakeExec{stdout: []byte(`not json`)}
	c := NewClient(fe)
	_, err := c.GetPRDetails(context.Background(), "x")
	if err == nil {
		t.Fatal("want decode error")
	}
}

func TestClassify_UnknownStderrWrapsRunErr(t *testing.T) {
	stderr := []byte("something weird happened\nline 2")
	runErr := errors.New("exit 2")
	err := classify(stderr, runErr)
	if !errors.Is(err, runErr) {
		t.Fatalf("want wrap runErr, got %v", err)
	}
	// Sanity: must include first line of stderr for debuggability.
	if got := err.Error(); !contains(got, "something weird happened") {
		t.Fatalf("want first stderr line in msg, got %q", got)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
