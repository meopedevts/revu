package profiles

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/meopedevts/revu/internal/auth"
)

// failingKeyring devolve erros configuráveis em qualquer dos 3 métodos
// da auth.Keyring. Set/Get/Delete sem erro caem num map interno pra
// permitir cenários parciais (ex: Set OK, Delete falha durante
// rollback).
type failingKeyring struct {
	set       error
	get       error
	deleteErr error
	store     map[string]string
}

func (f *failingKeyring) Set(ref, secret string) error {
	if f.set != nil {
		return f.set
	}
	if f.store == nil {
		f.store = map[string]string{}
	}
	f.store[ref] = secret
	return nil
}

func (f *failingKeyring) Get(ref string) (string, error) {
	if f.get != nil {
		return "", f.get
	}
	if v, ok := f.store[ref]; ok {
		return v, nil
	}
	return "", auth.ErrNotFound
}

func (f *failingKeyring) Delete(ref string) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	delete(f.store, ref)
	return nil
}

// fakeRepo é um stub do profileRepo com hooks injetáveis pros métodos
// exercitados pelos error-path tests. Métodos sem hook fazem t.Fatalf —
// chamada inesperada indica teste mal-configurado e fica visível ao
// invés de virar zero-value silencioso.
type fakeRepo struct {
	t *testing.T

	list         func(context.Context) ([]Profile, error)
	get          func(context.Context, string) (Profile, error)
	getActive    func(context.Context) (Profile, error)
	create       func(context.Context, Profile) error
	updateFields func(context.Context, Profile) error
	deleteFn     func(context.Context, string) error
	setActive    func(context.Context, string) error
	count        func(context.Context) (int, error)
}

func (f *fakeRepo) List(ctx context.Context) ([]Profile, error) {
	if f.list == nil {
		f.t.Helper()
		f.t.Fatalf("unexpected fakeRepo.List call")
	}
	return f.list(ctx)
}

func (f *fakeRepo) Get(ctx context.Context, id string) (Profile, error) {
	if f.get == nil {
		f.t.Helper()
		f.t.Fatalf("unexpected fakeRepo.Get call (id=%q)", id)
	}
	return f.get(ctx, id)
}

func (f *fakeRepo) GetActive(ctx context.Context) (Profile, error) {
	if f.getActive == nil {
		f.t.Helper()
		f.t.Fatalf("unexpected fakeRepo.GetActive call")
	}
	return f.getActive(ctx)
}

func (f *fakeRepo) Create(ctx context.Context, p Profile) error {
	if f.create == nil {
		f.t.Helper()
		f.t.Fatalf("unexpected fakeRepo.Create call")
	}
	return f.create(ctx, p)
}

func (f *fakeRepo) UpdateFields(ctx context.Context, p Profile) error {
	if f.updateFields == nil {
		f.t.Helper()
		f.t.Fatalf("unexpected fakeRepo.UpdateFields call")
	}
	return f.updateFields(ctx, p)
}

func (f *fakeRepo) Delete(ctx context.Context, id string) error {
	if f.deleteFn == nil {
		f.t.Helper()
		f.t.Fatalf("unexpected fakeRepo.Delete call (id=%q)", id)
	}
	return f.deleteFn(ctx, id)
}

func (f *fakeRepo) SetActive(ctx context.Context, id string) error {
	if f.setActive == nil {
		f.t.Helper()
		f.t.Fatalf("unexpected fakeRepo.SetActive call (id=%q)", id)
	}
	return f.setActive(ctx, id)
}

func (f *fakeRepo) Count(ctx context.Context) (int, error) {
	if f.count == nil {
		f.t.Helper()
		f.t.Fatalf("unexpected fakeRepo.Count call")
	}
	return f.count(ctx)
}

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
		t: t,
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
		t: t,
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
		t: t,
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

// TestService_Update_RepoUpdateFails_RollsBackKeyring exercita o caminho
// de erro em Service.Update: token novo é gravado no keyring,
// repo.UpdateFields falha, rollback do keyring é invocado e a entrada
// nova não fica órfã. Mata mutantes que pulam keyringRollback().
func TestService_Update_RepoUpdateFails_RollsBackKeyring(t *testing.T) {
	cur := Profile{
		ID:             "abc",
		Name:           "trabalho",
		AuthMethod:     AuthKeyring,
		KeyringRef:     "profile-abc",
		GitHubUsername: "old",
		IsActive:       false,
		CreatedAt:      time.Now().UTC(),
	}
	keys := auth.NewFake()
	if err := keys.Set("profile-abc", "ghp_old_token"); err != nil {
		t.Fatalf("seed keyring: %v", err)
	}
	baselineLen := keys.Len()

	sentinel := errors.New("update failed")
	repo := &fakeRepo{
		t: t,
		get: func(_ context.Context, id string) (Profile, error) {
			if id != "abc" {
				t.Fatalf("get id: want abc, got %q", id)
			}
			return cur, nil
		},
		updateFields: func(_ context.Context, _ Profile) error {
			return sentinel
		},
	}
	svc := newService(repo, keys, fakeOKRunner{})

	newToken := "ghp_new_token"
	_, err := svc.Update(context.Background(), "abc", Update{Token: &newToken})
	if !errors.Is(err, sentinel) {
		t.Fatalf("err: want sentinel, got %v", err)
	}
	if keys.Len() != baselineLen {
		t.Errorf("keyring entries: want %d (rollback restored old), got %d", baselineLen, keys.Len())
	}
	got, gerr := keys.Get("profile-abc")
	if gerr != nil {
		t.Fatalf("get rolled-back token: %v", gerr)
	}
	if got != "ghp_old_token" {
		t.Errorf("rolled-back token: want old, got %q", got)
	}
}

// TestService_Update_KeyringSetFails_PropagatesError exercita falha do
// keys.Set em Update (token novo): erro é wrappado "store token:" e
// repo.UpdateFields nunca é chamado (sem rollback necessário). Mata
// mutantes que invertem `if err != nil` em Set.
func TestService_Update_KeyringSetFails_PropagatesError(t *testing.T) {
	cur := Profile{
		ID:         "abc",
		Name:       "trabalho",
		AuthMethod: AuthKeyring,
		KeyringRef: "profile-abc",
		CreatedAt:  time.Now().UTC(),
	}
	setErr := errors.New("keyring locked")
	repo := &fakeRepo{
		t: t,
		get: func(_ context.Context, _ string) (Profile, error) {
			return cur, nil
		},
	}
	keys := &failingKeyring{set: setErr}
	svc := newService(repo, keys, fakeOKRunner{})

	newToken := "ghp_new"
	_, err := svc.Update(context.Background(), "abc", Update{Token: &newToken})
	if err == nil {
		t.Fatal("err: want failure, got nil")
	}
	if !strings.Contains(err.Error(), "store token:") {
		t.Errorf("err: want wrap 'store token:', got %v", err)
	}
}

// TestService_Update_KeyringRollbackLogsDeleteFailure verifica que se o
// rollback do keyring (após repo.UpdateFields falhar) também falha, o
// erro é apenas logado — Update propaga o erro original do repo. Mata
// mutantes que invertem o defer/recover do log.
func TestService_Update_KeyringRollbackLogsDeleteFailure(t *testing.T) {
	cur := Profile{
		ID:         "abc",
		Name:       "trabalho",
		AuthMethod: AuthKeyring,
		KeyringRef: "profile-abc",
		CreatedAt:  time.Now().UTC(),
	}
	repoErr := errors.New("repo down")
	repo := &fakeRepo{
		t: t,
		get: func(_ context.Context, _ string) (Profile, error) {
			return cur, nil
		},
		updateFields: func(_ context.Context, _ Profile) error {
			return repoErr
		},
	}
	keys := &failingKeyring{
		// Set sucede; Delete falha (durante rollback).
		deleteErr: errors.New("can't delete"),
	}
	buf := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(buf, nil))
	svc := newService(repo, keys, fakeOKRunner{}, WithLogger(logger))

	newToken := "ghp_new"
	_, err := svc.Update(context.Background(), "abc", Update{Token: &newToken})
	if !errors.Is(err, repoErr) {
		t.Fatalf("err: want repo sentinel, got %v", err)
	}
	if !strings.Contains(buf.String(), "rollback keyring entry") {
		t.Errorf("log: want 'rollback keyring entry', got %q", buf.String())
	}
}

// TestService_TokenFor_KeyringGetError exercita o caminho de erro do
// keys.Get em TokenFor: erro é wrappado com "read keyring:" e o token
// devolvido é vazio. Mata mutantes que invertem `if err != nil` em
// keys.Get ou trocam wrap.
func TestService_TokenFor_KeyringGetError(t *testing.T) {
	getErr := errors.New("keyring offline")
	keys := &failingKeyring{get: getErr}
	repo := &fakeRepo{t: t}
	svc := newService(repo, keys, fakeOKRunner{})

	prof := Profile{
		ID:         "abc",
		AuthMethod: AuthKeyring,
		KeyringRef: "profile-abc",
	}
	tok, err := svc.TokenFor(context.Background(), prof)
	if err == nil {
		t.Fatal("err: want failure, got nil")
	}
	if !strings.Contains(err.Error(), "read keyring:") {
		t.Errorf("err: want wrap 'read keyring:', got %v", err)
	}
	if tok != "" {
		t.Errorf("token: want empty on error, got %q", tok)
	}
}

// TestService_Update_GHCLI_DeleteKeyringError_LogsAndContinues garante
// que falha do keys.Delete na transição keyring→gh-cli não bloqueia o
// Update — só loga warning. Mata mutantes que removem o defer/log ou
// transformam o erro em fatal.
func TestService_Update_GHCLI_DeleteKeyringError_LogsAndContinues(t *testing.T) {
	cur := Profile{
		ID:         "abc",
		Name:       "trabalho",
		AuthMethod: AuthKeyring,
		KeyringRef: "profile-abc",
		CreatedAt:  time.Now().UTC(),
	}
	repo := &fakeRepo{
		t: t,
		get: func(_ context.Context, _ string) (Profile, error) {
			return cur, nil
		},
		updateFields: func(_ context.Context, _ Profile) error {
			return nil
		},
	}
	keys := &failingKeyring{deleteErr: errors.New("can't delete")}
	buf := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(buf, nil))
	svc := newService(repo, keys, fakeOKRunner{}, WithLogger(logger))

	ghcli := AuthGHCLI
	updated, err := svc.Update(context.Background(), "abc", Update{Method: &ghcli})
	if err != nil {
		t.Fatalf("Update should succeed despite delete err, got %v", err)
	}
	if updated.AuthMethod != AuthGHCLI {
		t.Errorf("method: want gh-cli, got %s", updated.AuthMethod)
	}
	if updated.KeyringRef != string(AuthGHCLI) {
		t.Errorf("keyring_ref: want gh-cli, got %q", updated.KeyringRef)
	}
	if !strings.Contains(buf.String(), "delete old keyring entry") {
		t.Errorf("log: want 'delete old keyring entry', got %q", buf.String())
	}
}

// TestService_FanoutActive_RecoversAndLogsSubscriberPanic garante que
// quando um subscriber panica, fanoutActive recupera, loga, e continua
// chamando os outros subscribers. Mata mutantes que removem o recover
// ou interrompem o loop.
func TestService_FanoutActive_RecoversAndLogsSubscriberPanic(t *testing.T) {
	repo := &fakeRepo{t: t}
	buf := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(buf, nil))
	svc := newService(repo, &failingKeyring{}, fakeOKRunner{}, WithLogger(logger))

	var goodCalls int
	var mu sync.Mutex
	svc.SubscribeActive(func(_ Profile) {
		panic("boom")
	})
	svc.SubscribeActive(func(_ Profile) {
		mu.Lock()
		goodCalls++
		mu.Unlock()
	})

	svc.fanoutActive(Profile{ID: "x"})

	mu.Lock()
	defer mu.Unlock()
	if goodCalls != 1 {
		t.Errorf("good subscriber: want 1 call (after panic recover), got %d", goodCalls)
	}
	if !strings.Contains(buf.String(), "subscriber panic in fanoutActive") {
		t.Errorf("log: want panic log, got %q", buf.String())
	}
}

// TestService_SubscribeActive_ConcurrentFanout dispara N goroutines
// fazendo Subscribe/Unsubscribe enquanto outra ráfaga roda
// fanoutActive. Garante invariantes do subsMu — corre com -race pra
// pegar mutações que removam o lock.
func TestService_SubscribeActive_ConcurrentFanout(t *testing.T) {
	repo := &fakeRepo{t: t}
	svc := newService(repo, &failingKeyring{}, fakeOKRunner{})

	const workers = 50
	const iterations = 20

	var wg sync.WaitGroup
	wg.Add(workers * 2)

	for range workers {
		go func() {
			defer wg.Done()
			for range iterations {
				unsub := svc.SubscribeActive(func(_ Profile) {})
				unsub()
			}
		}()
	}
	for range workers {
		go func() {
			defer wg.Done()
			for range iterations {
				svc.fanoutActive(Profile{ID: "x"})
			}
		}()
	}
	wg.Wait()
}
