package tray

import (
	"testing"

	"github.com/meopedevts/revu/assets"
)

func TestDeriveState_Priority(t *testing.T) {
	cases := []struct {
		name                         string
		pending, attention, hasError bool
		want                         State
	}{
		{"all-false → idle", false, false, false, StateIdle},
		{"only pending → pending", true, false, false, StatePending},
		{"only attention → attention", false, true, false, StateAttention},
		{"only error → error", false, false, true, StateError},
		{"pending+attention → attention", true, true, false, StateAttention},
		{"pending+error → error", true, false, true, StateError},
		{"attention+error → error", false, true, true, StateError},
		{"all true → error", true, true, true, StateError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := deriveState(tc.pending, tc.attention, tc.hasError)
			if got != tc.want {
				t.Fatalf("deriveState(p=%v, a=%v, e=%v) = %s, want %s",
					tc.pending, tc.attention, tc.hasError, got, tc.want)
			}
		})
	}
}

func TestIconFor_AllStates(t *testing.T) {
	cases := []struct {
		state State
		want  []byte
	}{
		{StateIdle, assets.TrayIdle},
		{StatePending, assets.TrayPending},
		{StateAttention, assets.TrayAttention},
		{StateError, assets.TrayError},
		{State(99), assets.TrayIdle}, // unknown → fallback idle
	}
	for _, tc := range cases {
		t.Run(tc.state.String(), func(t *testing.T) {
			got := iconFor(tc.state)
			if len(got) == 0 {
				t.Fatal("icon bytes are empty — embed missing?")
			}
			// Comparação por slice header é suficiente: assets.* são
			// pacotes-globais imutáveis.
			if &got[0] != &tc.want[0] {
				t.Fatalf("iconFor(%s): pointer mismatch — wrong asset wired", tc.state)
			}
		})
	}
}

func TestState_String(t *testing.T) {
	cases := map[State]string{
		StateIdle:      "idle",
		StatePending:   "pending",
		StateAttention: "attention",
		StateError:     "error",
		State(42):      "idle",
	}
	for s, want := range cases {
		if got := s.String(); got != want {
			t.Fatalf("State(%d).String() = %q, want %q", s, got, want)
		}
	}
}

// TestTray_FlagSetters_PreReady garante que os setters atualizam o current
// derivado mesmo antes de Start (sem chamar SetIcon — `ready` ainda é
// false). É a prova de que onReady vai pegar o estado correto no seed.
func TestTray_FlagSetters_PreReady(t *testing.T) {
	tr := New(nil, nil, nil, nil)

	if got := tr.State(); got != StateIdle {
		t.Fatalf("initial state: got %s, want idle", got)
	}

	tr.SetPending(true)
	if got := tr.State(); got != StatePending {
		t.Fatalf("after SetPending: got %s, want pending", got)
	}

	tr.SetAttention(true)
	if got := tr.State(); got != StateAttention {
		t.Fatalf("after SetAttention: got %s, want attention (overrides pending)", got)
	}

	tr.SetError(true)
	if got := tr.State(); got != StateError {
		t.Fatalf("after SetError: got %s, want error (overrides all)", got)
	}

	tr.SetError(false)
	if got := tr.State(); got != StateAttention {
		t.Fatalf("after clear error: got %s, want attention (next priority)", got)
	}

	tr.SetAttention(false)
	if got := tr.State(); got != StatePending {
		t.Fatalf("after clear attention: got %s, want pending", got)
	}

	tr.SetPending(false)
	if got := tr.State(); got != StateIdle {
		t.Fatalf("after clear pending: got %s, want idle", got)
	}
}

// TestTray_FlagSetters_Idempotent confirma que setar o mesmo valor não
// altera State — proxy razoável para "não chama SetIcon redundante".
func TestTray_FlagSetters_Idempotent(t *testing.T) {
	tr := New(nil, nil, nil, nil)
	tr.SetPending(true)
	first := tr.State()
	tr.SetPending(true)
	if tr.State() != first {
		t.Fatal("idempotent SetPending mudou State")
	}
}
