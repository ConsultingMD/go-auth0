package management

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ConsultingMD/go-auth0"
)

func TestHookManager_Create(t *testing.T) {
	configureHTTPTestRecordings(t)

	hook := &Hook{
		Name:      auth0.String("testing-hook-creation"),
		Script:    auth0.String("function (user, context, callback) { callback(null, { user }); }"),
		TriggerID: auth0.String("pre-user-registration"),
		Enabled:   auth0.Bool(false),
	}

	err := api.Hook.Create(context.Background(), hook)
	assert.NoError(t, err)
	assert.NotEmpty(t, hook.GetID())

	t.Cleanup(func() {
		cleanupHook(t, hook.GetID())
	})
}

func TestHookManager_Read(t *testing.T) {
	configureHTTPTestRecordings(t)

	expectedHook := givenAHook(t, nil)

	actualHook, err := api.Hook.Read(context.Background(), expectedHook.GetID())

	assert.NoError(t, err)
	assert.Equal(t, expectedHook, actualHook)
}

func TestHookManager_Update(t *testing.T) {
	configureHTTPTestRecordings(t)

	hook := givenAHook(t, nil)
	updatedHook := &Hook{
		Script:  auth0.String("function (user, context, callback) { console.log('hooked!'); callback(null, { user }); }"),
		Enabled: auth0.Bool(true),
	}

	err := api.Hook.Update(context.Background(), hook.GetID(), updatedHook)
	assert.NoError(t, err)

	actualHook, err := api.Hook.Read(context.Background(), hook.GetID())
	assert.NoError(t, err)
	assert.Equal(t, updatedHook.GetScript(), actualHook.GetScript())
	assert.Equal(t, updatedHook.GetEnabled(), actualHook.GetEnabled())
}

func TestHookManager_Delete(t *testing.T) {
	configureHTTPTestRecordings(t)

	hook := givenAHook(t, nil)

	err := api.Hook.Delete(context.Background(), hook.GetID())
	assert.NoError(t, err)

	actualHook, err := api.Hook.Read(context.Background(), hook.GetID())
	assert.Empty(t, actualHook)
	assert.Error(t, err)
	assert.Implements(t, (*Error)(nil), err)
	assert.Equal(t, http.StatusNotFound, err.(Error).Status())
}

func TestHookManager_List(t *testing.T) {
	configureHTTPTestRecordings(t)

	expectedHook := givenAHook(t, nil)

	hookList, err := api.Hook.List(context.Background(), IncludeFields("id"))

	assert.NoError(t, err)
	assert.Len(t, hookList.Hooks, 1)
	assert.Equal(t, expectedHook.GetID(), hookList.Hooks[0].GetID())
}

func TestHookManager_CreateSecrets(t *testing.T) {
	configureHTTPTestRecordings(t)

	hook := givenAHook(t, nil)
	secrets := HookSecrets{
		"SECRET1": "value1",
		"SECRET2": "value2",
	}

	err := api.Hook.CreateSecrets(context.Background(), hook.GetID(), secrets)
	assert.NoError(t, err)
}

func TestHookManager_UpdateSecrets(t *testing.T) {
	configureHTTPTestRecordings(t)

	secrets := HookSecrets{
		"SECRET1": "value1",
		"SECRET2": "value2",
	}
	hook := givenAHook(t, secrets)

	err := api.Hook.UpdateSecrets(context.Background(), hook.GetID(), HookSecrets{"SECRET1": "something else"})
	assert.NoError(t, err)

	actualSecrets, err := api.Hook.Secrets(context.Background(), hook.GetID())
	assert.NoError(t, err)
	assert.Equal(t, actualSecrets["SECRET1"], "_VALUE_NOT_SHOWN_")
	assert.Equal(t, actualSecrets["SECRET2"], "_VALUE_NOT_SHOWN_")
}

func TestHookManager_ReplaceSecrets(t *testing.T) {
	configureHTTPTestRecordings(t)

	secrets := HookSecrets{
		"SECRET1": "value1",
		"SECRET2": "value2",
	}
	hook := givenAHook(t, secrets)

	newSecrets := HookSecrets{
		"SECRET1": "something else",
		"SECRET3": "other value",
	}
	err := api.Hook.ReplaceSecrets(context.Background(), hook.GetID(), newSecrets)
	assert.NoError(t, err)

	actualSecrets, err := api.Hook.Secrets(context.Background(), hook.GetID())
	assert.NoError(t, err)
	assert.Equal(t, actualSecrets["SECRET1"], "_VALUE_NOT_SHOWN_")
	assert.Empty(t, actualSecrets["SECRET2"])
	assert.Equal(t, actualSecrets["SECRET3"], "_VALUE_NOT_SHOWN_")
}

func TestHookManager_Secrets(t *testing.T) {
	configureHTTPTestRecordings(t)

	secrets := HookSecrets{
		"SECRET1": "value1",
		"SECRET2": "value2",
	}
	hook := givenAHook(t, secrets)

	actualSecrets, err := api.Hook.Secrets(context.Background(), hook.GetID())
	assert.NoError(t, err)
	assert.Equal(t, actualSecrets["SECRET1"], "_VALUE_NOT_SHOWN_")
	assert.Equal(t, actualSecrets["SECRET2"], "_VALUE_NOT_SHOWN_")
}

func TestHookManager_RemoveSecrets(t *testing.T) {
	configureHTTPTestRecordings(t)

	secrets := HookSecrets{
		"SECRET1": "value1",
		"SECRET2": "value2",
	}
	hook := givenAHook(t, secrets)

	err := api.Hook.RemoveSecrets(context.Background(), hook.GetID(), []string{"SECRET1"})
	assert.NoError(t, err)

	actualSecrets, err := api.Hook.Secrets(context.Background(), hook.GetID())
	assert.NoError(t, err)
	assert.Empty(t, actualSecrets["SECRET1"])
	assert.Equal(t, actualSecrets["SECRET2"], "_VALUE_NOT_SHOWN_")
}

func TestHookManager_RemoveAllSecrets(t *testing.T) {
	configureHTTPTestRecordings(t)

	secrets := HookSecrets{
		"SECRET1": "value1",
		"SECRET2": "value2",
	}
	hook := givenAHook(t, secrets)

	err := api.Hook.RemoveAllSecrets(context.Background(), hook.GetID())
	assert.NoError(t, err)

	actualSecrets, err := api.Hook.Secrets(context.Background(), hook.GetID())
	assert.NoError(t, err)
	assert.Empty(t, actualSecrets["SECRET1"])
	assert.Empty(t, actualSecrets["SECRET2"])
}

func TestHookSecretsDifference(t *testing.T) {
	for _, testCase := range []struct {
		secrets, other, difference HookSecrets
	}{
		{
			secrets:    HookSecrets{"foo": "", "bar": ""},
			other:      HookSecrets{"bar": ""},
			difference: HookSecrets{"foo": ""},
		},
		{
			secrets:    HookSecrets{"foo": "", "bar": "", "baz": ""},
			other:      HookSecrets{"bar": ""},
			difference: HookSecrets{"foo": "", "baz": ""},
		},
	} {
		assert.Equal(t, testCase.difference, testCase.secrets.difference(testCase.other))
	}
}

func TestHookSecretsIntersection(t *testing.T) {
	for _, testCase := range []struct {
		secrets, other, intersection HookSecrets
	}{
		{
			secrets:      HookSecrets{"foo": "", "bar": ""},
			other:        HookSecrets{"bar": ""},
			intersection: HookSecrets{"bar": ""},
		},
		{
			secrets:      HookSecrets{"foo": "", "bar": "", "baz": ""},
			other:        HookSecrets{"bar": ""},
			intersection: HookSecrets{"bar": ""},
		},
	} {
		assert.Equal(t, testCase.intersection, testCase.secrets.intersection(testCase.other))
	}
}

func givenAHook(t *testing.T, secrets HookSecrets) *Hook {
	t.Helper()

	hook := &Hook{
		Name:      auth0.String(fmt.Sprintf("test-hook%d", rand.Intn(999))),
		Script:    auth0.String("function (user, context, callback) { callback(null, { user }); }"),
		TriggerID: auth0.String("pre-user-registration"),
		Enabled:   auth0.Bool(false),
	}

	err := api.Hook.Create(context.Background(), hook)
	require.NoError(t, err)

	if secrets != nil {
		err := api.Hook.CreateSecrets(context.Background(), hook.GetID(), secrets)
		require.NoError(t, err)
	}

	t.Cleanup(func() {
		cleanupHook(t, hook.GetID())
	})

	return hook
}

func cleanupHook(t *testing.T, hookID string) {
	t.Helper()

	err := api.Hook.Delete(context.Background(), hookID)
	if err != nil {
		if err.(Error).Status() != http.StatusNotFound {
			t.Error(err)
		}
	}
}
