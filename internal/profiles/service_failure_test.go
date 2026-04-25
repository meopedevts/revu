package profiles

import (
	"context"
	"errors"
	"testing"

	"github.com/meopedevts/revu/internal/auth"
)

// fakeRepo é um stub do profileRepo com hooks injetáveis pros métodos
// exercitados pelos error-path tests. Métodos sem hook retornam
// zero-value/nil — adequado pros caminhos cobertos aqui (nenhum teste
// failure exercita List/Get/UpdateFields/Delete/SetActive/Count).
type fakeRepo struct {
	getActive func(context.Context) (Profile, error)
	create    func(context.Context, Profile) error
}

func (f *fakeRepo) List(_ context.Context) ([]Profile, error)        { return nil, nil }
func (f *fakeRepo) Get(_ context.Context, _ string) (Profile, error) { return Profile{}, nil }
func (f *fakeRepo) GetActive(ctx context.Context) (Profile, error) {
	if f.getActive != nil {
		return f.getActive(ctx)
	}
	return Profile{}, nil
}

func (f *fakeRepo) Create(ctx context.Context, p Profile) error {
	if f.create != nil {
		return f.create(ctx, p)
	}
	return nil
}
func (f *fakeRepo) UpdateFields(_ context.Context, _ Profile) error { return nil }
func (f *fakeRepo) Delete(_ context.Context, _ string) error        { return nil }
func (f *fakeRepo) SetActive(_ context.Context, _ string) error     { return nil }
func (f *fakeRepo) Count(_ context.Context) (int, error)            { return 0, nil }

// fakeOKRunner devolve sempre um login GitHub válido — usado pelo
// Create_Rollback test pra passar pela validação de token antes de o
// repo.Create falhar.
type fakeOKRunner struct{}

func (fakeOKRunner) Run(_ context.Context, _ string, _ ...string) ([]byte, []byte, error) {
	return []byte(`{"login":"octocat"}`), nil, nil
}

func (fakeOKRunner) RunEnv(_ context.Context, _ []string, _ string, _ ...string) ([]byte, []byte, error) {
	return []byte(`{"login":"octocat"}`), nil, nil
}

// TestService_ActiveProfileID_RepoError garante que ActiveProfileID
// propaga o erro de GetActive sem mascarar — mata mutantes que invertem
// `if err != nil` ou retornam o ID mesmo com erro.
func TestService_ActiveProfileID_RepoError(t *testing.T) {
	sentinel := errors.New("boom")
	repo := &fakeRepo{
		getActive: func(_ context.Context) (Profile, error) {
			return Profile{}, sentinel
		},
	}
	svc := newService(repo, auth.NewFake(), fakeOKRunner{})

	id, err := svc.ActiveProfileID(context.Background())
	if !errors.Is(err, sentinel) {
		t.Fatalf("err: want sentinel, got %v", err)
	}
	if id != "" {
		t.Errorf("id: want empty on error, got %q", id)
	}
}

// TestService_Create_KeyringRollback_OnRepoCreateError verifica que
// quando repo.Create falha após o token ter sido salvo no keyring, o
// rollback (keys.Delete) é chamado — mata mutantes que pulam o rollback
// ou invertem a checagem `prof.KeyringRef != ""`.
func TestService_Create_KeyringRollback_OnRepoCreateError(t *testing.T) {
	sentinel := errors.New("repo down")
	repo := &fakeRepo{
		create: func(_ context.Context, _ Profile) error {
			return sentinel
		},
	}
	keys := auth.NewFake()
	svc := newService(repo, keys, fakeOKRunner{})

	_, err := svc.Create(context.Background(), CreateParams{
		Name:   "trabalho",
		Method: AuthKeyring,
		Token:  "ghp_rollback_test",
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("err: want sentinel, got %v", err)
	}
	if n := keys.Len(); n != 0 {
		t.Errorf("keyring: want 0 entries after rollback, got %d", n)
	}
}

// TestService_TokenForActive_RepoError garante que TokenForActive
// propaga o erro do GetActive subjacente — mata mutantes que retornam
// token vazio sem erro.
func TestService_TokenForActive_RepoError(t *testing.T) {
	sentinel := errors.New("no active")
	repo := &fakeRepo{
		getActive: func(_ context.Context) (Profile, error) {
			return Profile{}, sentinel
		},
	}
	svc := newService(repo, auth.NewFake(), fakeOKRunner{})

	tok, err := svc.TokenForActive(context.Background())
	if !errors.Is(err, sentinel) {
		t.Fatalf("err: want sentinel, got %v", err)
	}
	if tok != "" {
		t.Errorf("token: want empty on error, got %q", tok)
	}
}
