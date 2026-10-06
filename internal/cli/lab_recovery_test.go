package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/gustavfredrikson/cka-dojo/internal/config"
	"github.com/gustavfredrikson/cka-dojo/internal/content"
	"github.com/gustavfredrikson/cka-dojo/internal/progress"
	"github.com/gustavfredrikson/cka-dojo/internal/provider"
	"github.com/gustavfredrikson/cka-dojo/internal/provider/fake"
	"github.com/spf13/cobra"
)

// These command tests operate entirely on a temporary state directory and an
// in-memory provider. Stub key material and scripts keep warm provisioning
// independent of external binaries and learner state.
func recoveryApp(t *testing.T) (*App, *fake.Provider) {
	t.Helper()
	t.Setenv("DOJO_HOME", t.TempDir())
	t.Setenv("DOJO_DEV", "0")
	files := fstest.MapFS{}
	add := func(path, body string) { files[path] = &fstest.MapFile{Data: []byte(body)} }
	add("curriculum/test/curriculum.yaml", "id: test\nmodules: [practice]\n")
	add("environments/_common/provisioning/common.sh", "fixture-provision-common\n")
	add("environments/_common/provisioning/terminal.sh", "fixture-provision-terminal\n")
	add("environments/_common/provisioning/shell-defaults.bash", "# fixture shell defaults\n")
	add("curriculum/test/modules/practice/module.yaml", "id: practice\nname: Practice\nlabs: [fixture, other]\n")
	for _, id := range []string{"fixture", "other"} {
		profile := "raw"
		if id == "other" {
			profile = "standard"
		}
		base := "curriculum/test/modules/practice/labs/" + id + "/"
		add(base+"lab.yaml", fmt.Sprintf(`schemaVersion: 1
id: %s
title: Recovery fixture
learningStage: follow
environment: {profile: %s}
setup:
  faults:
    - type: nodeExec
      node: cp1
      script: fixture-prepare
      undo: fixture-cleanup-prepare
    - type: nodeExec
      node: cp1
      script: fixture-inject
      undo: fixture-cleanup-inject
variants:
  options:
    - id: working
    - id: broken-plan
      faults:
        - type: unsupported-fixture-fault
grading:
  all:
    - type: command
      command: fixture-grade
      description: the fixture is repaired
checkpoints:
  - id: inspect
    task: Answer ready.
    answer: {accepted: [ready]}
`, id, profile))
		add(base+"task.md", "# Recovery fixture\nRepair the fixture.\n")
		add(base+"solution.md", "Repair the fixture.\n")
	}
	prov := fake.New()
	for _, profile := range []string{"standard", "raw"} {
		add("environments/"+profile+"/environment.yaml", fmt.Sprintf(`schemaVersion: 1
id: %s
provisioning: none
nodes:
  - {name: terminal, role: workstation, cpus: 1, memory: 1GiB, disk: 1GiB}
  - {name: cp1, role: control-plane, cpus: 1, memory: 1GiB, disk: 1GiB}
`, profile))
		for _, node := range []string{"terminal", "cp1"} {
			prov.Nodes["cka-dojo-"+profile+"-"+node] = provider.StatusRunning
		}
		dir, err := config.EnvDir(profile)
		if err != nil {
			t.Fatal(err)
		}
		for _, file := range []string{"id_ed25519", "id_ed25519.pub"} {
			if err := os.WriteFile(filepath.Join(dir, file), []byte("fixture key\n"), 0o600); err != nil {
				t.Fatal(err)
			}
		}
	}
	return &App{
		Cfg: &config.Config{Curriculum: "test", Profile: "standard"},
		Src: &content.Source{FS: files, Origin: "test"}, Provider: prov,
	}, prov
}

func recoveryExecResult(script string) provider.ExecResult {
	if strings.Contains(script, "ip -4 -o addr show") {
		return provider.ExecResult{Stdout: "192.168.104.10\n"}
	}
	return provider.ExecResult{}
}

func runRecoveryCommand(cmd *cobra.Command, args ...string) error {
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetArgs(args)
	return cmd.Execute()
}

func recoveryState(t *testing.T) *config.State {
	t.Helper()
	st, err := config.LoadState()
	if err != nil {
		t.Fatal(err)
	}
	return st
}

func recoveryHistory(t *testing.T) *progress.File {
	t.Helper()
	history, err := progress.Load()
	if err != nil {
		t.Fatal(err)
	}
	return history
}

func startRecoveryLab(t *testing.T, app *App) {
	t.Helper()
	if err := runRecoveryCommand(newStartCmd(app), "fixture", "--variant", "working", "--seed", "1234"); err != nil {
		t.Fatalf("start: %v", err)
	}
}

func TestFailedStartRetainsPlanAndStopRepairsPartialScenario(t *testing.T) {
	app, prov := recoveryApp(t)
	prov.Responder = func(node, script string) (provider.ExecResult, error) {
		if strings.Contains(script, "fixture-inject") {
			st := recoveryState(t)
			if !st.SetupPending || st.ActiveLab != "fixture" || st.Profile != "raw" || st.Variant != "working" || st.Seed != 1234 {
				t.Fatalf("setup recovery metadata was not persisted before mutation: %+v", st)
			}
			return provider.ExecResult{}, fmt.Errorf("injected setup failure")
		}
		return recoveryExecResult(script), nil
	}
	err := runRecoveryCommand(newStartCmd(app), "fixture", "--variant", "working", "--seed", "1234")
	if err == nil || !strings.Contains(err.Error(), "dojo reset") {
		t.Fatalf("start error = %v, want recovery advice", err)
	}
	if !prov.Ran("fixture-prepare") || !recoveryState(t).SetupPending {
		t.Fatal("partial scenario did not retain its recovery state")
	}
	if len(recoveryHistory(t).Labs) != 0 {
		t.Fatal("a failed initial setup recorded study progress")
	}
	for _, cmd := range []*cobra.Command{newGradeCmd(app), newCheckCmd(app)} {
		if err := runRecoveryCommand(cmd); err == nil || !strings.Contains(err.Error(), "setup is incomplete") {
			t.Fatalf("pending command error = %v", err)
		}
	}
	if err := runRecoveryCommand(newStopCmd(app)); err != nil {
		t.Fatalf("stop partial scenario: %v", err)
	}
	if !prov.Ran("fixture-cleanup-prepare") || !prov.Ran("fixture-cleanup-inject") || recoveryState(t).Active() {
		t.Fatal("stop did not repair the partial scenario and clear its state")
	}
	if len(recoveryHistory(t).Labs) != 0 {
		t.Fatal("cleaning up failed setup recorded study progress")
	}
}

func TestResetRecoversFailedFirstStartAndRecordsOneAttempt(t *testing.T) {
	app, prov := recoveryApp(t)
	fail := true
	prov.Responder = func(node, script string) (provider.ExecResult, error) {
		if fail && strings.Contains(script, "fixture-inject") {
			return provider.ExecResult{}, fmt.Errorf("injected setup failure")
		}
		return recoveryExecResult(script), nil
	}
	if err := runRecoveryCommand(newStartCmd(app), "fixture", "--variant", "working"); err == nil {
		t.Fatal("start unexpectedly succeeded")
	}
	fail = false
	if err := runRecoveryCommand(newResetCmd(app)); err != nil {
		t.Fatalf("recover initial setup: %v", err)
	}
	st := recoveryState(t)
	att := recoveryHistory(t).Labs["fixture"]
	if st.SetupPending || st.Attempt != 1 || st.StartedAt.IsZero() || att == nil || att.Attempts != 1 || att.Passes != 0 {
		t.Fatalf("recovered state/history = %+v / %+v", st, att)
	}
	if err := runRecoveryCommand(newCheckCmd(app), "ready"); err != nil {
		t.Fatalf("complete recovered attempt: %v", err)
	}
	if err := runRecoveryCommand(newGradeCmd(app)); err != nil {
		t.Fatalf("regrade recovered attempt: %v", err)
	}
	att = recoveryHistory(t).Labs["fixture"]
	if att.Attempts != 1 || att.Passes != 1 {
		t.Fatalf("recovery or regrading inflated progress: %+v", att)
	}
}

func TestResetResumesIncompleteProvisioningEvenWhenEveryGuestIsRunning(t *testing.T) {
	app, prov := recoveryApp(t)
	delete(prov.Nodes, "cka-dojo-raw-terminal")
	delete(prov.Nodes, "cka-dojo-raw-cp1")
	fail, provisionCalls := true, 0
	prov.Responder = func(node, script string) (provider.ExecResult, error) {
		if strings.Contains(script, "ip -4 -o addr show") {
			return provider.ExecResult{Stdout: "192.168.104.10\n"}, nil
		}
		if strings.Contains(script, "fixture-provision-common") {
			provisionCalls++
			if fail {
				return provider.ExecResult{}, fmt.Errorf("injected provisioning failure")
			}
		}
		if strings.Contains(script, "fixture-cleanup") && recoveryState(t).EnvironmentPending {
			t.Fatal("scenario teardown ran before provisioning completed")
		}
		return provider.ExecResult{}, nil
	}
	if err := runRecoveryCommand(newStartCmd(app), "fixture", "--variant", "working"); err == nil {
		t.Fatal("provisioning unexpectedly succeeded")
	}
	st := recoveryState(t)
	if !st.SetupPending || !st.EnvironmentPending || len(recoveryHistory(t).Labs) != 0 {
		t.Fatalf("failed provisioning lost its phase or recorded progress: %+v", st)
	}
	if prov.Nodes["cka-dojo-raw-terminal"] != provider.StatusRunning || prov.Nodes["cka-dojo-raw-cp1"] != provider.StatusRunning {
		t.Fatal("fixture did not reproduce failure after all guests were started")
	}
	if prov.Ran("fixture-prepare") {
		t.Fatal("scenario mutation ran after failed provisioning")
	}
	fail = false
	if err := runRecoveryCommand(newResetCmd(app)); err != nil {
		t.Fatalf("resume provisioning: %v", err)
	}
	st = recoveryState(t)
	att := recoveryHistory(t).Labs["fixture"]
	if provisionCalls < 3 || st.SetupPending || st.EnvironmentPending || att == nil || att.Attempts != 1 || att.Passes != 0 {
		t.Fatalf("reset skipped provisioning or inflated progress: calls=%d state=%+v history=%+v", provisionCalls, st, att)
	}
}

func TestStartResumesInterruptedStandaloneSetupWithRunningGuests(t *testing.T) {
	app, prov := recoveryApp(t)
	if recoveryState(t).Active() {
		t.Fatal("fixture has an active lab instead of an interrupted standalone setup")
	}
	provisioned, scenarioStarted := false, false
	prov.Responder = func(node, script string) (provider.ExecResult, error) {
		if strings.Contains(script, "fixture-provision-terminal") {
			provisioned = true
		}
		if strings.Contains(script, "fixture-prepare") {
			scenarioStarted = true
			if !provisioned || recoveryState(t).EnvironmentPending {
				t.Fatal("new scenario started before unfinished provisioning completed")
			}
		}
		return recoveryExecResult(script), nil
	}
	startRecoveryLab(t, app)
	if !provisioned || !scenarioStarted {
		t.Fatal("start did not resume standalone provisioning before building the lab")
	}
	st := recoveryState(t)
	att := recoveryHistory(t).Labs["fixture"]
	if st.SetupPending || st.EnvironmentPending || att == nil || att.Attempts != 1 || att.Passes != 0 {
		t.Fatalf("completed warm start state/history = %+v / %+v", st, att)
	}
}

func TestResetStartsStoppedGuestsBeforeRepairingScenarioWithoutReprovisioning(t *testing.T) {
	app, prov := recoveryApp(t)
	startRecoveryLab(t, app)
	previousCalls := len(prov.Calls)
	prov.Nodes["cka-dojo-raw-terminal"] = provider.StatusStopped
	prov.Nodes["cka-dojo-raw-cp1"] = provider.StatusStopped
	prov.Responder = func(node, script string) (provider.ExecResult, error) {
		if strings.Contains(script, "fixture-cleanup") && prov.Nodes[node] != provider.StatusRunning {
			t.Fatal("scenario repair ran before its guest was restarted")
		}
		return recoveryExecResult(script), nil
	}
	if err := runRecoveryCommand(newResetCmd(app)); err != nil {
		t.Fatalf("reset stopped guests: %v", err)
	}
	for _, call := range prov.Calls[previousCalls:] {
		if strings.Contains(call.Script, "fixture-provision-") {
			t.Fatal("reset reran provisioning before repairing scenario faults")
		}
	}
	if recoveryState(t).SetupPending || recoveryState(t).EnvironmentPending {
		t.Fatal("reset did not finish the recovered scenario")
	}
}

func TestFailedResetBlocksGradingWithoutChangingSuccessfulHistory(t *testing.T) {
	app, prov := recoveryApp(t)
	startRecoveryLab(t, app)
	if err := runRecoveryCommand(newCheckCmd(app), "ready"); err != nil {
		t.Fatal(err)
	}
	prov.Responder = func(node, script string) (provider.ExecResult, error) {
		if strings.Contains(script, "fixture-inject") {
			return provider.ExecResult{}, fmt.Errorf("injected reset failure")
		}
		return recoveryExecResult(script), nil
	}
	if err := runRecoveryCommand(newResetCmd(app)); err == nil {
		t.Fatal("reset unexpectedly succeeded")
	}
	if !recoveryState(t).SetupPending {
		t.Fatal("failed reset did not mark setup incomplete")
	}
	for _, cmd := range []*cobra.Command{newGradeCmd(app), newCheckCmd(app)} {
		if err := runRecoveryCommand(cmd, "ready"); err == nil || !strings.Contains(err.Error(), "setup is incomplete") {
			t.Fatalf("pending reset command error = %v", err)
		}
	}
	att := recoveryHistory(t).Labs["fixture"]
	if att.Attempts != 1 || att.Passes != 1 {
		t.Fatalf("failed reset changed previous progress: %+v", att)
	}
	prov.Responder = nil
	if err := runRecoveryCommand(newResetCmd(app)); err != nil {
		t.Fatalf("recover failed reset: %v", err)
	}
	if recoveryState(t).SetupPending || recoveryState(t).Passed || recoveryState(t).Checkpoint != 0 {
		t.Fatal("successful reset did not restore an ungraded scenario")
	}
	att = recoveryHistory(t).Labs["fixture"]
	if att.Attempts != 1 || att.Passes != 1 {
		t.Fatalf("reset recovery changed previous successful history: %+v", att)
	}
}

func TestFailedCleanupRetainsOldLabBeforeStopOrProfileSwitch(t *testing.T) {
	app, prov := recoveryApp(t)
	startRecoveryLab(t, app)
	prov.Responder = func(node, script string) (provider.ExecResult, error) {
		if strings.Contains(script, "fixture-cleanup") {
			return provider.ExecResult{}, fmt.Errorf("injected cleanup failure")
		}
		return recoveryExecResult(script), nil
	}
	if err := runRecoveryCommand(newStopCmd(app)); err == nil {
		t.Fatal("stop ignored failed cleanup")
	}
	if st := recoveryState(t); st.ActiveLab != "fixture" || !st.SetupPending {
		t.Fatalf("stop discarded recovery state: %+v", st)
	}
	if err := runRecoveryCommand(newStartCmd(app), "other", "--force", "--variant", "working"); err == nil {
		t.Fatal("start ignored failed cleanup of the old lab")
	}
	if st := recoveryState(t); st.ActiveLab != "fixture" || !st.SetupPending {
		t.Fatalf("start discarded previous recovery state: %+v", st)
	}
	if prov.Nodes["cka-dojo-raw-cp1"] != provider.StatusRunning || prov.Nodes["cka-dojo-standard-cp1"] != provider.StatusStopped {
		t.Fatal("start switched profiles before previous cleanup succeeded")
	}
	if recoveryHistory(t).Labs["other"] != nil {
		t.Fatal("failed profile switch recorded an attempt for the new lab")
	}
	if err := runRecoveryCommand(newStopCmd(app), "--keep"); err != nil || recoveryState(t).Active() {
		t.Fatalf("explicit keep failed to end the lab: %v", err)
	}
}

func TestInvalidVariantOrPlanDoesNotMutateCurrentLabOrMachines(t *testing.T) {
	for _, variant := range []string{"missing", "broken-plan"} {
		t.Run(variant, func(t *testing.T) {
			app, prov := recoveryApp(t)
			startRecoveryLab(t, app)
			calls := len(prov.Calls)
			before := *recoveryState(t)
			err := runRecoveryCommand(newStartCmd(app), "other", "--force", "--variant", variant)
			if err == nil {
				t.Fatal("invalid scenario unexpectedly started")
			}
			if after := recoveryState(t); *after != before || len(prov.Calls) != calls {
				t.Fatalf("invalid scenario changed existing state or ran commands: %+v -> %+v", before, after)
			}
			if prov.Nodes["cka-dojo-raw-cp1"] != provider.StatusRunning || prov.Nodes["cka-dojo-standard-cp1"] != provider.StatusStopped {
				t.Fatal("invalid scenario switched environments")
			}
		})
	}
}

type shellRecorder struct {
	*fake.Provider
	node string
	user string
}

func (p *shellRecorder) Shell(_ context.Context, node, user string, _ []string) error {
	p.node, p.user = node, user
	return nil
}

func TestShellProfilePrecedence(t *testing.T) {
	for _, tc := range []struct {
		name, explicit, active, configured, want string
	}{
		{"configured default", "", "", "raw", "raw"},
		{"active lab", "", "raw", "standard", "raw"},
		{"explicit profile", "standard", "raw", "raw", "standard"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app, prov := recoveryApp(t)
			app.profileFlag, app.Cfg.Profile = tc.explicit, tc.configured
			if tc.active != "" {
				if err := config.SaveState(&config.State{ActiveLab: "fixture", Profile: tc.active}); err != nil {
					t.Fatal(err)
				}
			}
			recorder := &shellRecorder{Provider: prov}
			app.Provider = recorder
			if err := runRecoveryCommand(newShellCmd(app)); err != nil {
				t.Fatalf("shell: %v", err)
			}
			if recorder.node != "cka-dojo-"+tc.want+"-terminal" || recorder.user != "student" {
				t.Fatalf("shell opened %s as %s", recorder.node, recorder.user)
			}
		})
	}
}

func TestEnvironmentDestroyUsesActiveProfileAndPreservesOtherLab(t *testing.T) {
	for _, explicit := range []string{"", "standard"} {
		t.Run("profile="+explicit, func(t *testing.T) {
			app, prov := recoveryApp(t)
			app.profileFlag = explicit
			if err := config.SaveState(&config.State{ActiveLab: "fixture", Profile: "raw", SetupPending: true}); err != nil {
				t.Fatal(err)
			}
			if err := runRecoveryCommand(newEnvDestroyCmd(app), "--yes"); err != nil {
				t.Fatal(err)
			}
			if explicit == "" {
				if _, exists := prov.Nodes["cka-dojo-raw-cp1"]; exists || recoveryState(t).Active() {
					t.Fatal("default destroy did not remove the pending active environment")
				}
			} else {
				if _, exists := prov.Nodes["cka-dojo-standard-cp1"]; exists || recoveryState(t).ActiveLab != "fixture" {
					t.Fatal("explicit destroy discarded the other profile's active lab")
				}
				if prov.Nodes["cka-dojo-raw-cp1"] != provider.StatusRunning {
					t.Fatal("explicit destroy touched the active profile")
				}
			}
		})
	}
}

func TestHintAndSolutionRefuseStateMutationWhileLabOperationIsLocked(t *testing.T) {
	app, _ := recoveryApp(t)
	if err := config.SaveState(&config.State{ActiveLab: "fixture", Profile: "raw", SetupPending: true}); err != nil {
		t.Fatal(err)
	}
	lock, err := config.Acquire()
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Release()
	for _, cmd := range []*cobra.Command{newHintCmd(app), newSolutionCmd(app)} {
		if err := runRecoveryCommand(cmd); err == nil || !strings.Contains(err.Error(), "another dojo command") {
			t.Fatalf("concurrent %s error = %v", cmd.Name(), err)
		}
	}
	if !recoveryState(t).SetupPending {
		t.Fatal("concurrent hint/solution discarded pending setup state")
	}
}
