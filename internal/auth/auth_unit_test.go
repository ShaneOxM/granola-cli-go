package auth

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	fn()
	_ = w.Close()
	os.Stdout = old
	buf := new(bytes.Buffer)
	_, _ = io.Copy(buf, r)
	return buf.String()
}

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	fn()
	_ = w.Close()
	os.Stderr = old
	buf := new(bytes.Buffer)
	_, _ = io.Copy(buf, r)
	return buf.String()
}

func withAuthFns(t *testing.T, get func() (*Credentials, error), load func() (*Credentials, error), save func(*Credentials) error, del func() error) {
	t.Helper()
	oldGet, oldLoad, oldSave, oldDelete := getCredentialsFn, loadCredentialsFromFileFn, saveCredentialsFn, deleteCredentialsFn
	t.Cleanup(func() {
		getCredentialsFn = oldGet
		loadCredentialsFromFileFn = oldLoad
		saveCredentialsFn = oldSave
		deleteCredentialsFn = oldDelete
		defaultAuth = nil
	})
	if get != nil {
		getCredentialsFn = get
	}
	if load != nil {
		loadCredentialsFromFileFn = load
	}
	if save != nil {
		saveCredentialsFn = save
	}
	if del != nil {
		deleteCredentialsFn = del
	}
}

func TestAuthGetAuthEnvVars(t *testing.T) {
	withAuthFns(t,
		func() (*Credentials, error) {
			return &Credentials{AccessToken: "a", RefreshToken: "r", ClientID: "c"}, nil
		}, nil, nil, nil,
	)
	a := &Auth{cache: map[string]string{}}
	env, err := a.GetAuthEnvVars()
	if err != nil {
		t.Fatalf("GetAuthEnvVars error: %v", err)
	}
	if env["GRANOLA_ACCESS_TOKEN"] != "a" || env["GRANOLA_CLIENT_ID"] != "c" {
		t.Fatalf("unexpected env: %+v", env)
	}
}

func TestAuthGetAuthEnvVarsError(t *testing.T) {
	withAuthFns(t,
		func() (*Credentials, error) { return nil, errors.New("missing") }, nil, nil, nil,
	)
	a := &Auth{cache: map[string]string{}}
	_, err := a.GetAuthEnvVars()
	if err == nil || !strings.Contains(err.Error(), "no authentication available") {
		t.Fatalf("expected auth env error, got %v", err)
	}
}

func TestAuthStatus(t *testing.T) {
	withAuthFns(t,
		func() (*Credentials, error) {
			return &Credentials{EmailAddress: "e@example.com", ClientID: "c", AccessToken: "a", RefreshToken: "r"}, nil
		}, nil, nil, nil,
	)
	a := &Auth{}
	out := captureStdout(t, func() { _ = a.Status() })
	if !strings.Contains(out, "Authenticated account") {
		t.Fatalf("unexpected status output: %s", out)
	}
}

func TestAuthStatusMissingCreds(t *testing.T) {
	withAuthFns(t,
		func() (*Credentials, error) { return nil, errors.New("missing") }, nil, nil, nil,
	)
	a := &Auth{}
	out := captureStdout(t, func() { _ = a.Status() })
	if !strings.Contains(out, "granola auth login") {
		t.Fatalf("unexpected missing creds output: %s", out)
	}
}

func TestAuthLogin(t *testing.T) {
	saved := false
	withAuthFns(t,
		nil,
		func() (*Credentials, error) {
			return &Credentials{EmailAddress: "e@example.com", ClientID: "c", AccessToken: "a", RefreshToken: "r"}, nil
		},
		func(*Credentials) error { saved = true; return nil },
		nil,
	)
	a := &Auth{}
	out := captureStdout(t, func() {
		if err := a.Login(nil); err != nil {
			t.Fatalf("login error: %v", err)
		}
	})
	if !saved || !strings.Contains(out, "Credentials imported successfully") {
		t.Fatalf("unexpected login output: %s", out)
	}
}

func TestAuthLoginLoadError(t *testing.T) {
	withAuthFns(t, nil, func() (*Credentials, error) { return nil, errors.New("no file") }, nil, nil)
	a := &Auth{}
	errOut := captureStderr(t, func() { _ = a.Login(nil) })
	if !strings.Contains(errOut, "Make sure the Granola desktop app is installed") {
		t.Fatalf("unexpected stderr: %s", errOut)
	}
}

func TestAuthLogout(t *testing.T) {
	called := false
	withAuthFns(t, nil, nil, nil, func() error { called = true; return nil })
	a := &Auth{}
	out := captureStdout(t, func() { _ = a.Logout(nil) })
	if !called || !strings.Contains(out, "Logged out successfully") {
		t.Fatalf("unexpected logout output: %s", out)
	}
}

func TestAuthLogoutError(t *testing.T) {
	withAuthFns(t, nil, nil, nil, func() error { return errors.New("boom") })
	a := &Auth{}
	errOut := captureStderr(t, func() { _ = a.Logout(nil) })
	if !strings.Contains(errOut, "Failed to logout") {
		t.Fatalf("unexpected stderr: %s", errOut)
	}
}

func TestGlobalWrappersRequireInit(t *testing.T) {
	defaultAuth = nil
	if _, err := GetAuthEnvVars(); err == nil {
		t.Fatal("expected GetAuthEnvVars init error")
	}
	if err := Status(); err == nil {
		t.Fatal("expected Status init error")
	}
	if err := Login(nil); err == nil {
		t.Fatal("expected Login init error")
	}
	if err := Logout(nil); err == nil {
		t.Fatal("expected Logout init error")
	}
}

func TestGlobalWrappersUseDefaultAuth(t *testing.T) {
	called := false
	withAuthFns(t,
		func() (*Credentials, error) {
			called = true
			return &Credentials{AccessToken: "a", RefreshToken: "r", ClientID: "c"}, nil
		},
		nil, nil, nil,
	)
	defaultAuth = &Auth{cache: map[string]string{}}
	_, err := GetAuthEnvVars()
	if err != nil || !called {
		t.Fatalf("expected wrapper to use default auth err=%v called=%v", err, called)
	}
}
