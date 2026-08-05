package services

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"clawreef/internal/models"
)

func TestParseRuntimeAgentSkillsReport(t *testing.T) {
	payload := map[string]any{
		"mode": "full",
		"instances": []map[string]any{
			{
				"instance_id": 12,
				"skills": []map[string]any{
					{
						"identifier":  "weather",
						"content_md5": "abc123",
						"source":      "runtime",
					},
				},
			},
		},
	}
	reports, mode, _, err := parseRuntimeAgentSkillsReport(payload)
	if err != nil {
		t.Fatalf("parseRuntimeAgentSkillsReport() error = %v", err)
	}
	if mode != "full" {
		t.Fatalf("mode = %q, want full", mode)
	}
	if len(reports) != 1 || reports[0].InstanceID != 12 {
		t.Fatalf("unexpected reports: %#v", reports)
	}
	if len(reports[0].Skills) != 1 || reports[0].Skills[0].Identifier != "weather" {
		t.Fatalf("unexpected skills: %#v", reports[0].Skills)
	}
}

func TestNormalizeRuntimeSkillSource(t *testing.T) {
	if got := normalizeRuntimeSkillSource("runtime"); got != "discovered_in_instance" {
		t.Fatalf("normalizeRuntimeSkillSource(runtime) = %q", got)
	}
	if got := normalizeRuntimeSkillSource("injected_by_clawmanager"); got != "injected_by_clawmanager" {
		t.Fatalf("normalizeRuntimeSkillSource(injected) = %q", got)
	}
}

func TestCollectLiteSkillDirectoryFiles(t *testing.T) {
	root := t.TempDir()
	skillRoot := filepath.Join(root, "weather")
	if err := os.MkdirAll(filepath.Join(skillRoot, "src"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillRoot, "SKILL.md"), []byte("# weather\n"), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillRoot, "src", "main.py"), []byte("print('ok')\n"), 0o640); err != nil {
		t.Fatal(err)
	}

	files, err := collectLiteSkillDirectoryFiles(skillRoot)
	if err != nil {
		t.Fatalf("collectLiteSkillDirectoryFiles() error = %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("files = %#v, want 2 entries", files)
	}
	if got := hashDirectory(files); got == "" {
		t.Fatal("expected non-empty content md5")
	}
}

func TestCollectLiteSkillDirectoryFilesSkipsWithoutManifest(t *testing.T) {
	root := t.TempDir()
	skillRoot := filepath.Join(root, "orphan")
	if err := os.MkdirAll(skillRoot, 0o750); err != nil {
		t.Fatal(err)
	}
	files, err := collectLiteSkillDirectoryFiles(skillRoot)
	if err != nil {
		t.Fatalf("collectLiteSkillDirectoryFiles() error = %v", err)
	}
	if files != nil {
		t.Fatalf("files = %#v, want nil", files)
	}
}

func TestDiscoverRuntimeSkillDirectoriesNestedCategory(t *testing.T) {
	root := t.TempDir()
	categoryRoot := filepath.Join(root, "productivity")
	skillRoot := filepath.Join(categoryRoot, "my-skill")
	for _, dir := range []string{skillRoot, filepath.Join(skillRoot, "src")} {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(skillRoot, "SKILL.md"), []byte("# my skill\n"), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillRoot, "src", "main.py"), []byte("print('ok')\n"), 0o640); err != nil {
		t.Fatal(err)
	}

	discoveries, err := discoverRuntimeSkillDirectories(root, runtimeSkillDiscoveryMaxDepth)
	if err != nil {
		t.Fatalf("discoverRuntimeSkillDirectories() error = %v", err)
	}
	if len(discoveries) != 1 {
		t.Fatalf("discoveries = %#v, want 1 nested skill", discoveries)
	}
	if discoveries[0].RelativePath != "productivity/my-skill" {
		t.Fatalf("RelativePath = %q, want productivity/my-skill", discoveries[0].RelativePath)
	}
}

func TestDiscoverRuntimeSkillDirectoriesFlatAndNested(t *testing.T) {
	root := t.TempDir()
	flatRoot := filepath.Join(root, "weather")
	if err := os.MkdirAll(flatRoot, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(flatRoot, "SKILL.md"), []byte("# weather\n"), 0o640); err != nil {
		t.Fatal(err)
	}

	discoveries, err := discoverRuntimeSkillDirectories(root, runtimeSkillDiscoveryMaxDepth)
	if err != nil {
		t.Fatalf("discoverRuntimeSkillDirectories() error = %v", err)
	}
	if len(discoveries) != 1 || discoveries[0].RelativePath != "weather" {
		t.Fatalf("discoveries = %#v, want flat weather skill", discoveries)
	}
}

func TestRuntimeSkillInstallRootOpenClawAndHermes(t *testing.T) {
	workspace := "/workspaces/demo/instance-1"
	hermes := &models.Instance{Type: RuntimeTypeHermes, WorkspacePath: &workspace}
	openclaw := &models.Instance{Type: RuntimeTypeOpenClaw, WorkspacePath: &workspace}

	hermesRoot := runtimeSkillInstallRoot(hermes)
	openclawRoot := runtimeSkillInstallRoot(openclaw)
	if hermesRoot != filepath.Join(workspace, ".hermes", "skills") {
		t.Fatalf("hermes root = %q", hermesRoot)
	}
	if openclawRoot != filepath.Join(workspace, "home", ".openclaw", "workspace", "skills") {
		t.Fatalf("openclaw root = %q", openclawRoot)
	}
}

func TestResolveInstanceSkillSourceTypePreservesInjected(t *testing.T) {
	existing := &models.InstanceSkill{SourceType: "injected_by_clawmanager"}
	skill := &models.Skill{SourceType: skillSourceUploaded}
	got := resolveInstanceSkillSourceType(existing, "discovered_in_instance", skill)
	if got != "injected_by_clawmanager" {
		t.Fatalf("resolveInstanceSkillSourceType() = %q, want injected_by_clawmanager", got)
	}
}

func TestSanitizeWorkspaceRelativePath(t *testing.T) {
	if got := sanitizeWorkspaceRelativePath("productivity/my-skill"); got != "productivity/my-skill" {
		t.Fatalf("sanitizeWorkspaceRelativePath() = %q", got)
	}
	if got := sanitizeWorkspaceRelativePath("../escape"); got != "" {
		t.Fatalf("sanitizeWorkspaceRelativePath(../escape) = %q, want empty", got)
	}
}

func TestIsLiteRuntimeInstanceOpenClawVariants(t *testing.T) {
	liteGateway := &models.Instance{InstanceMode: InstanceModeLite, RuntimeType: RuntimeBackendGateway, Type: RuntimeTypeOpenClaw}
	if !IsLiteRuntimeInstance(liteGateway) {
		t.Fatal("expected gateway lite instance")
	}
	proDesktop := &models.Instance{InstanceMode: InstanceModePro, RuntimeType: RuntimeBackendDesktop, Type: RuntimeTypeOpenClaw}
	if IsLiteRuntimeInstance(proDesktop) {
		t.Fatal("expected pro desktop instance to be non-lite")
	}
	shellPod := &models.Instance{InstanceMode: InstanceModePro, RuntimeType: RuntimeBackendShell, Type: RuntimeTypeOpenClaw}
	if IsLiteRuntimeInstance(shellPod) {
		t.Fatal("expected shell pod instance to use pro agent path")
	}
}

type provenanceCaptureRepoStub struct {
	capturingSkillRepoStub
	upserted []*models.InstanceSkill
}

func (s *provenanceCaptureRepoStub) GetBlobByContentHash(hash string) (*models.SkillBlob, error) {
	for _, blob := range s.blobs {
		if strings.EqualFold(strings.TrimSpace(blob.ContentHash), strings.TrimSpace(hash)) {
			copy := *blob
			return &copy, nil
		}
	}
	return nil, nil
}

func (s *provenanceCaptureRepoStub) GetVersionBySkillAndBlob(skillID, blobID int) (*models.SkillVersion, error) {
	for _, version := range s.versions {
		if version.SkillID == skillID && version.BlobID == blobID {
			copy := *version
			return &copy, nil
		}
	}
	return nil, nil
}

func (s *provenanceCaptureRepoStub) UpsertInstanceSkill(item *models.InstanceSkill) error {
	copy := *item
	s.upserted = append(s.upserted, &copy)
	updated := false
	for i, existing := range s.instanceSkills {
		if existing.InstanceID == item.InstanceID && existing.SkillID == item.SkillID {
			s.instanceSkills[i] = copy
			updated = true
			break
		}
	}
	if !updated {
		s.instanceSkills = append(s.instanceSkills, copy)
	}
	return nil
}

func TestSyncAgentSkillsPreservesInjectedProvenanceAfterWorkspaceScan(t *testing.T) {
	contentHash := "abc123def456789012345678901234"
	versionID := 1
	stub := &provenanceCaptureRepoStub{
		capturingSkillRepoStub: capturingSkillRepoStub{
			skillRepoStub: skillRepoStub{
			skills: map[int]*models.Skill{
				10: {
					ID: 10, UserID: 1, SkillKey: "ppt-1-0-0", Name: "ppt-1.0.0",
					SourceType: skillSourceUploaded, Status: skillStatusActive,
					Visibility: skillVisibilityPublic, CurrentVersionID: &versionID,
				},
			},
			blobs: map[int]*models.SkillBlob{
				1: {ID: 1, ContentHash: contentHash, ObjectKey: "hub/ppt.zip", ScanStatus: "completed"},
			},
			versions: map[int]*models.SkillVersion{
				1: {ID: 1, SkillID: 10, BlobID: 1, VersionNo: 1},
			},
			instanceSkills: []models.InstanceSkill{
				{InstanceID: 1, SkillID: 10, SourceType: "injected_by_clawmanager", Status: "active"},
			},
			},
		},
	}
	instRepo := &importTestInstanceRepo{instances: map[int]*models.Instance{
		1: {ID: 1, UserID: 1, Type: RuntimeTypeHermes, InstanceMode: InstanceModePro, RuntimeType: RuntimeBackendDesktop},
	}}
	svc := &skillService{repo: stub, instanceRepo: instRepo, commandService: &noopInstanceCommandService{}}
	err := svc.SyncAgentSkills(1, AgentSkillInventoryReportRequest{
		Mode: "full",
		Skills: []AgentSkillRecord{{
			Identifier:  "ppt-1-0-0",
			ContentMD5:  contentHash,
			Source:      "discovered_in_instance",
			InstallPath: "home/.hermes/skills/ppt-1-0-0",
		}},
	})
	if err != nil {
		t.Fatalf("SyncAgentSkills() error = %v", err)
	}
	if len(stub.upserted) != 1 {
		t.Fatalf("upserted %d instance skills, want 1", len(stub.upserted))
	}
	if stub.upserted[0].SourceType != "injected_by_clawmanager" {
		t.Fatalf("SourceType = %q, want injected_by_clawmanager", stub.upserted[0].SourceType)
	}
}

func TestSyncAgentSkillsReusesUploadedSkillOnWorkspaceScan(t *testing.T) {
	contentHash := "abc123def456789012345678901234"
	versionID := 1
	stub := &provenanceCaptureRepoStub{
		capturingSkillRepoStub: capturingSkillRepoStub{
			skillRepoStub: skillRepoStub{
			skills: map[int]*models.Skill{
				10: {
					ID: 10, UserID: 1, SkillKey: "ppt-1-0-0", Name: "ppt-1.0.0",
					SourceType: skillSourceUploaded, Status: skillStatusActive,
					Visibility: skillVisibilityPublic, CurrentVersionID: &versionID,
				},
			},
			blobs: map[int]*models.SkillBlob{
				1: {ID: 1, ContentHash: contentHash, ObjectKey: "hub/ppt.zip", ScanStatus: "completed"},
			},
			versions: map[int]*models.SkillVersion{
				1: {ID: 1, SkillID: 10, BlobID: 1, VersionNo: 1},
			},
			},
		},
	}
	instRepo := &importTestInstanceRepo{instances: map[int]*models.Instance{
		1: {ID: 1, UserID: 1, Type: RuntimeTypeHermes, InstanceMode: InstanceModeLite, RuntimeType: RuntimeBackendGateway},
	}}
	svc := &skillService{repo: stub, instanceRepo: instRepo, commandService: &noopInstanceCommandService{}}
	err := svc.SyncAgentSkills(1, AgentSkillInventoryReportRequest{
		Mode: "full",
		Skills: []AgentSkillRecord{{
			Identifier:  "ppt-1-0-0",
			ContentMD5:  contentHash,
			Source:      "discovered_in_instance",
			InstallPath: "home/.hermes/skills/ppt-1-0-0",
		}},
	})
	if err != nil {
		t.Fatalf("SyncAgentSkills() error = %v", err)
	}
	if len(stub.createdSkills) != 0 {
		t.Fatalf("created %d discovered skills, want 0 reuse of uploaded skill", len(stub.createdSkills))
	}
	if len(stub.upserted) != 1 || stub.upserted[0].SkillID != 10 {
		t.Fatalf("upserted = %#v, want instance skill for uploaded skill id 10", stub.upserted)
	}
}

func TestSyncAgentSkillsReconcilesDiscoveredSkillToExistingContentHash(t *testing.T) {
	reportedHash := "55836cb933c884d1964443421aad714a"
	legacyHash := "34020e1053ea1ae5aced654c1ef7aff1"
	versionID := 20
	stub := &provenanceCaptureRepoStub{
		capturingSkillRepoStub: capturingSkillRepoStub{
			skillRepoStub: skillRepoStub{
				skills: map[int]*models.Skill{
					20: {
						ID: 20, UserID: 7, SkillKey: "finproc-tender-monitor", Name: "finproc-tender-monitor",
						SourceType: skillSourceDiscovered, Status: skillStatusActive,
						Visibility: skillVisibilityPrivate, CurrentVersionID: &versionID,
					},
				},
				blobs: map[int]*models.SkillBlob{
					75: {ID: 75, ContentHash: legacyHash, ObjectKey: "discovered/legacy.zip", ScanStatus: "completed"},
					76: {ID: 76, ContentHash: reportedHash, ObjectKey: "hub/finproc.zip", ScanStatus: "completed"},
				},
				versions: map[int]*models.SkillVersion{
					20: {ID: 20, SkillID: 20, BlobID: 75, VersionNo: 1},
				},
				instanceSkills: []models.InstanceSkill{},
			},
		},
	}
	instRepo := &importTestInstanceRepo{instances: map[int]*models.Instance{
		1: {ID: 1, UserID: 7, Type: RuntimeTypeHermes, InstanceMode: InstanceModeLite, RuntimeType: RuntimeBackendGateway},
	}}
	svc := &skillService{repo: stub, instanceRepo: instRepo, commandService: &noopInstanceCommandService{}}
	err := svc.SyncAgentSkills(1, AgentSkillInventoryReportRequest{
		Mode: "full",
		Skills: []AgentSkillRecord{{
			Identifier:  "finproc-tender-monitor",
			ContentMD5:  reportedHash,
			Source:      "discovered_in_instance",
			InstallPath: "home/.hermes/skills/finproc-tender-monitor",
		}},
	})
	if err != nil {
		t.Fatalf("SyncAgentSkills() error = %v", err)
	}
	blob75 := stub.blobs[75]
	if blob75 == nil || blob75.ContentHash != legacyHash {
		t.Fatalf("blob 75 content hash = %q, want unchanged %q (no duplicate-hash mutation)", blob75.ContentHash, legacyHash)
	}
	if stub.versions[20].BlobID != 75 {
		t.Fatalf("version 20 BlobID = %d, want 75 (existing version must not be repointed away)", stub.versions[20].BlobID)
	}
	if len(stub.createdVersions) != 1 {
		t.Fatalf("created %d versions, want 1 new version for reported content", len(stub.createdVersions))
	}
	created := stub.createdVersions[0]
	if created.SkillID != 20 || created.BlobID != 76 {
		t.Fatalf("created version = skill %d blob %d, want skill 20 blob 76", created.SkillID, created.BlobID)
	}
	if stub.skills[20].CurrentVersionID == nil || *stub.skills[20].CurrentVersionID != created.ID {
		t.Fatalf("skill 20 CurrentVersionID = %v, want new version %d", stub.skills[20].CurrentVersionID, created.ID)
	}
	if len(stub.createdSkills) != 0 {
		t.Fatalf("created %d discovered skills, want 0", len(stub.createdSkills))
	}
	if len(stub.upserted) != 1 || stub.upserted[0].SkillID != 20 {
		t.Fatalf("upserted = %#v, want instance skill for skill 20", stub.upserted)
	}
	if stub.upserted[0].SkillVersionID == nil || *stub.upserted[0].SkillVersionID != created.ID {
		t.Fatalf("SkillVersionID = %v, want new version %d", stub.upserted[0].SkillVersionID, created.ID)
	}
	if stub.upserted[0].ObservedHash == nil || *stub.upserted[0].ObservedHash != reportedHash {
		t.Fatalf("ObservedHash = %v, want %q", stub.upserted[0].ObservedHash, reportedHash)
	}
}

func TestSyncAgentSkillsStaleInjectedReportKeepsCurrentVersionAndExistingVersion(t *testing.T) {
	reportedHash := "55836cb933c884d1964443421aad714a"
	currentHash := "a05fc8e284e0967d1213d25bdc452b8d"
	version81 := 81
	stub := &provenanceCaptureRepoStub{
		capturingSkillRepoStub: capturingSkillRepoStub{
			skillRepoStub: skillRepoStub{
				skills: map[int]*models.Skill{
					75: {
						ID: 75, UserID: 1, SkillKey: "finproc-tender-monitor", Name: "finproc-tender-monitor",
						SourceType: skillSourceUploaded, Status: skillStatusActive,
						Visibility: skillVisibilityPublic, CurrentVersionID: &version81,
					},
				},
				blobs: map[int]*models.SkillBlob{
					76: {ID: 76, ContentHash: reportedHash, ObjectKey: "hub/finproc-v2.zip", ScanStatus: "completed"},
					77: {ID: 77, ContentHash: currentHash, ObjectKey: "hub/finproc-v3.zip", ScanStatus: "completed"},
				},
				versions: map[int]*models.SkillVersion{
					80: {ID: 80, SkillID: 75, BlobID: 76, VersionNo: 2},
					81: {ID: 81, SkillID: 75, BlobID: 77, VersionNo: 3},
				},
				instanceSkills: []models.InstanceSkill{},
			},
		},
	}
	instRepo := &importTestInstanceRepo{instances: map[int]*models.Instance{
		1: {ID: 1, UserID: 1, Type: RuntimeTypeHermes, InstanceMode: InstanceModeLite, RuntimeType: RuntimeBackendGateway},
	}}
	svc := &skillService{repo: stub, instanceRepo: instRepo, commandService: &noopInstanceCommandService{}}
	err := svc.SyncAgentSkills(1, AgentSkillInventoryReportRequest{
		Mode: "full",
		Skills: []AgentSkillRecord{{
			SkillID:     "skill_75",
			Identifier:  "finproc-tender-monitor",
			ContentMD5:  reportedHash,
			Source:      "injected_by_clawmanager",
			InstallPath: "home/.hermes/skills/finproc-tender-monitor",
		}},
	})
	if err != nil {
		t.Fatalf("SyncAgentSkills() error = %v (duplicate version key must not trigger)", err)
	}
	if stub.versions[81].BlobID != 77 {
		t.Fatalf("current version 81 BlobID = %d, want untouched 77 (published v3 must not be repointed to stale report)", stub.versions[81].BlobID)
	}
	if stub.versions[80].BlobID != 76 {
		t.Fatalf("version 80 BlobID = %d, want 76 (existing version must be reused, not mutated)", stub.versions[80].BlobID)
	}
	if len(stub.createdVersions) != 0 {
		t.Fatalf("created %d versions, want 0 (no duplicate version for skill 75 + blob 76)", len(stub.createdVersions))
	}
	if len(stub.upserted) != 1 || stub.upserted[0].SkillID != 75 {
		t.Fatalf("upserted = %#v, want instance skill for skill 75", stub.upserted)
	}
	if stub.upserted[0].SkillVersionID == nil || *stub.upserted[0].SkillVersionID != 80 {
		t.Fatalf("SkillVersionID = %v, want 80 (the existing version matching the stale reported content)", stub.upserted[0].SkillVersionID)
	}
	if stub.upserted[0].ObservedHash == nil || *stub.upserted[0].ObservedHash != reportedHash {
		t.Fatalf("ObservedHash = %v, want %q", stub.upserted[0].ObservedHash, reportedHash)
	}
	if stub.skills[75].CurrentVersionID == nil || *stub.skills[75].CurrentVersionID != 81 {
		t.Fatalf("skill 75 CurrentVersionID = %v, want 81 (uploaded skill current version must stay on published v3)", stub.skills[75].CurrentVersionID)
	}
}

type recordingSkillResyncAgentClient struct {
	fakeRuntimeAgentClient
	calls []struct {
		instanceID int
		mode       string
	}
	err error
}

func (c *recordingSkillResyncAgentClient) ResyncInstanceSkills(_ context.Context, _ string, instanceID int, mode string) error {
	c.calls = append(c.calls, struct {
		instanceID int
		mode       string
	}{instanceID: instanceID, mode: mode})
	return c.err
}

type inventorySyncCommandRepo struct {
	commands []models.InstanceCommand
}

func (r *inventorySyncCommandRepo) Create(*models.InstanceCommand) error { return nil }
func (r *inventorySyncCommandRepo) Update(command *models.InstanceCommand) error {
	for i := range r.commands {
		if r.commands[i].ID == command.ID {
			r.commands[i] = *command
			return nil
		}
	}
	r.commands = append(r.commands, *command)
	return nil
}
func (r *inventorySyncCommandRepo) GetByID(int) (*models.InstanceCommand, error) { return nil, nil }
func (r *inventorySyncCommandRepo) GetByInstanceIdempotencyKey(int, string) (*models.InstanceCommand, error) {
	return nil, nil
}
func (r *inventorySyncCommandRepo) GetNextPendingByInstance(int) (*models.InstanceCommand, error) {
	return nil, nil
}
func (r *inventorySyncCommandRepo) ListByInstanceID(int, int) ([]models.InstanceCommand, error) {
	return append([]models.InstanceCommand(nil), r.commands...), nil
}
func (r *inventorySyncCommandRepo) FindLatestFailedCollectSkillPackage(string) (*models.InstanceCommand, error) {
	return nil, nil
}

func TestRequestLiteSkillInventorySyncUsesIncrementalWhenAgentResyncFollows(t *testing.T) {
	workspace := t.TempDir()
	skillsRoot := filepath.Join(workspace, "home", ".hermes", "skills")
	if err := os.MkdirAll(skillsRoot, 0o750); err != nil {
		t.Fatal(err)
	}

	stub := &capturingSkillRepoStub{
		skillRepoStub: skillRepoStub{
			skills: map[int]*models.Skill{
				10: {ID: 10, UserID: 1, SkillKey: "weather", Name: "weather", SourceType: skillSourceDiscovered, Status: skillStatusActive},
			},
			blobs:    map[int]*models.SkillBlob{},
			versions: map[int]*models.SkillVersion{},
			instanceSkills: []models.InstanceSkill{
				{InstanceID: 1, SkillID: 10, Status: "active", SourceType: "discovered_in_instance"},
			},
		},
	}
	instRepo := &importTestInstanceRepo{instances: map[int]*models.Instance{
		1: {
			ID: 1, UserID: 1, Type: RuntimeTypeHermes, InstanceMode: InstanceModeLite,
			RuntimeType: RuntimeBackendGateway, WorkspacePath: &workspace, RuntimeGeneration: 1,
		},
	}}
	cmdRepo := &inventorySyncCommandRepo{commands: []models.InstanceCommand{{
		ID: 44, InstanceID: 1, CommandType: InstanceCommandTypeSyncSkillInventory, Status: instanceCommandStatusPending,
	}}}
	agent := &recordingSkillResyncAgentClient{}
	endpoint := "http://runtime-agent"
	bindingRepo := newFakeRuntimeBindingRepo()
	bindingRepo.bindings[1] = &models.InstanceRuntimeBinding{
		InstanceID: 1, RuntimePodID: 9, State: "running", Generation: 1,
	}
	podRepo := &fakeRuntimePodRepo{pods: map[int64]*models.RuntimePod{
		9: {ID: 9, AgentEndpoint: &endpoint},
	}}

	svc := &skillService{
		repo:         stub,
		instanceRepo: instRepo,
		commandRepo:  cmdRepo,
		commandService: &noopInstanceCommandService{},
	}
	svc.ConfigureRuntimeSkillSync(bindingRepo, podRepo, agent)

	if err := svc.RequestLiteSkillInventorySync(1); err != nil {
		t.Fatalf("RequestLiteSkillInventorySync() error = %v", err)
	}
	if stub.markMissingCalls != 0 {
		t.Fatalf("markMissingCalls = %d, want 0 when workspace sync is incremental", stub.markMissingCalls)
	}
	if stub.instanceSkills[0].Status != "active" {
		t.Fatalf("status = %q, want active", stub.instanceSkills[0].Status)
	}
	if len(agent.calls) != 1 || agent.calls[0].mode != "full" {
		t.Fatalf("resync calls = %#v, want one full resync", agent.calls)
	}
	if cmdRepo.commands[0].Status != instanceCommandStatusPending {
		t.Fatalf("command status = %q, want pending until agent inventory arrives", cmdRepo.commands[0].Status)
	}
}

func TestRequestLiteSkillInventorySyncPropagatesResyncError(t *testing.T) {
	workspace := t.TempDir()
	skillsRoot := filepath.Join(workspace, "home", ".hermes", "skills")
	if err := os.MkdirAll(skillsRoot, 0o750); err != nil {
		t.Fatal(err)
	}

	stub := &capturingSkillRepoStub{
		skillRepoStub: skillRepoStub{
			skills:         map[int]*models.Skill{},
			blobs:          map[int]*models.SkillBlob{},
			versions:       map[int]*models.SkillVersion{},
			instanceSkills: nil,
		},
	}
	instRepo := &importTestInstanceRepo{instances: map[int]*models.Instance{
		1: {
			ID: 1, UserID: 1, Type: RuntimeTypeHermes, InstanceMode: InstanceModeLite,
			RuntimeType: RuntimeBackendGateway, WorkspacePath: &workspace, RuntimeGeneration: 1,
		},
	}}
	agent := &recordingSkillResyncAgentClient{err: errors.New("resync failed")}
	endpoint := "http://runtime-agent"
	bindingRepo := newFakeRuntimeBindingRepo()
	bindingRepo.bindings[1] = &models.InstanceRuntimeBinding{
		InstanceID: 1, RuntimePodID: 9, State: "running", Generation: 1,
	}
	podRepo := &fakeRuntimePodRepo{pods: map[int64]*models.RuntimePod{
		9: {ID: 9, AgentEndpoint: &endpoint},
	}}
	svc := &skillService{repo: stub, instanceRepo: instRepo, commandService: &noopInstanceCommandService{}}
	svc.ConfigureRuntimeSkillSync(bindingRepo, podRepo, agent)

	err := svc.RequestLiteSkillInventorySync(1)
	if err == nil || !strings.Contains(err.Error(), "resync") {
		t.Fatalf("error = %v, want resync failure", err)
	}
}
