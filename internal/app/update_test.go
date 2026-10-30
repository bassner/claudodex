package app

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/bassner/claudodex/internal/selfupdate"
)

type recordingUpdater struct {
	requests []selfupdate.Request
	result   selfupdate.Result
	err      error
}

func (u *recordingUpdater) BeforeLaunch(_ context.Context, r selfupdate.Request) (selfupdate.Result, error) {
	u.requests = append(u.requests, r)
	return u.result, u.err
}

func TestUpdatePreflightKeepsRawModelOverrides(t *testing.T) {
	u := &recordingUpdater{result: selfupdate.Result{StatusLineExecutable: "/private/helper"}}
	l := &fakeLauncher{}
	args := []string{"--claudodex-opus-model", "custom", "--model", "opus"}
	code := Run(context.Background(), Config{Version: "1.0.0", Updater: u, Launcher: l}, args)
	if code != 0 || len(u.requests) != 1 || !reflect.DeepEqual(u.requests[0].Args, args) {
		t.Fatalf("code=%d requests=%+v", code, u.requests)
	}
	if !reflect.DeepEqual(l.args, []string{"--model", "opus"}) || l.cfg.Models.Opus != "custom" || l.cfg.StatusLineExecutable != "/private/helper" {
		t.Fatalf("launch=%+v", l)
	}
}

func TestUpdateFailureContinuesLaunchWithPinnedHelper(t *testing.T) {
	u := &recordingUpdater{result: selfupdate.Result{StatusLineExecutable: "/private/helper"}, err: errors.New("cancelled")}
	l := &fakeLauncher{}
	var out bytes.Buffer
	if code := Run(context.Background(), Config{Updater: u, Launcher: l, Stderr: &out}, []string{"--model", "opus"}); code != 0 || len(l.args) == 0 || l.cfg.StatusLineExecutable == "" {
		t.Fatalf("code=%d launch=%+v", code, l)
	}
}

func TestManualUpdateStopsBeforeClaude(t *testing.T) {
	for _, args := range [][]string{{"clx:update"}, {"clx:update", "--yes"}} {
		u := &recordingUpdater{}
		l := &fakeLauncher{}
		code := Run(context.Background(), Config{Updater: u, Launcher: l}, args)
		if code != 0 || len(l.args) > 0 || len(u.requests) != 1 || !u.requests[0].Manual || u.requests[0].Yes != (len(args) == 2) {
			t.Fatalf("args=%q code=%d update=%+v", args, code, u)
		}
	}
}

func TestUpdateStatuslineAndInvalidInputDoNotCheck(t *testing.T) {
	for _, args := range [][]string{{"__statusline"}, {"--claudodex-opus-model"}, {"clx:update", "--surprise"}} {
		u := &recordingUpdater{}
		Run(context.Background(), Config{Updater: u}, args)
		if len(u.requests) > 0 {
			t.Fatalf("unexpected check for %q", args)
		}
	}
}
