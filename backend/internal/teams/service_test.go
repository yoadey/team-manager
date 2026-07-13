package teams_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/color"
	"image/jpeg"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/teams"
)

// fixedJPEG returns a minimal valid 2x2 JPEG for image-processing tests.
func fixedJPEG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	var buf bytes.Buffer
	require.NoError(t, jpeg.Encode(&buf, img, nil))
	return buf.Bytes()
}

// fakeOversizedPNGHeader builds just enough of a PNG (signature + a valid
// IHDR chunk declaring width x height) for image/png.DecodeConfig to
// successfully read the declared dimensions — without actually encoding
// width*height pixels, which would itself be the decompression-bomb-sized
// allocation the resize code is meant to reject before ever happening.
func fakeOversizedPNGHeader(width, height uint32) []byte {
	var buf bytes.Buffer
	buf.Write([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}) // PNG signature

	ihdr := make([]byte, 13)
	binary.BigEndian.PutUint32(ihdr[0:4], width)
	binary.BigEndian.PutUint32(ihdr[4:8], height)
	ihdr[8] = 8  // bit depth
	ihdr[9] = 6  // color type: RGBA
	ihdr[10] = 0 // compression
	ihdr[11] = 0 // filter
	ihdr[12] = 0 // interlace

	chunkType := []byte("IHDR")
	var lenBuf [4]byte
	binary.BigEndian.PutUint32(lenBuf[:], uint32(len(ihdr)))
	buf.Write(lenBuf[:])
	buf.Write(chunkType)
	buf.Write(ihdr)

	crc := crc32.NewIEEE()
	crc.Write(chunkType)
	crc.Write(ihdr)
	var crcBuf [4]byte
	binary.BigEndian.PutUint32(crcBuf[:], crc.Sum32())
	buf.Write(crcBuf[:])

	return buf.Bytes()
}

// ─── mock repository ─────────────────────────────────────────────────────────

type mockTeamRepo struct {
	listTeamsForUser       func(ctx context.Context, userID string) ([]teams.TeamRow, error)
	getTeam                func(ctx context.Context, teamID string) (*teams.TeamRow, error)
	createTeam             func(ctx context.Context, name, creatorUserID string, icon, iconBg, iconFg *string) (*teams.TeamRow, error)
	updateTeam             func(ctx context.Context, teamID string, patch teams.TeamPatch) (*teams.TeamRow, error)
	getMemberCount         func(ctx context.Context, teamID string) (int, error)
	getMembership          func(ctx context.Context, teamID, userID string) (*teams.MembershipRow, error)
	getRolesForMembership  func(ctx context.Context, membershipID, teamID string) ([]teams.RoleRow, error)
	getMemberCounts        func(ctx context.Context, teamIDs []string) (map[string]int, error)
	getMembershipsForUser  func(ctx context.Context, teamIDs []string, userID string) (map[string]teams.MembershipRow, error)
	getRolesForMemberships func(ctx context.Context, membershipIDs []string) (map[string][]teams.RoleRow, error)
	createInvite           func(ctx context.Context, teamID string, ttl time.Duration) (*teams.InviteRow, error)
	acceptInvite           func(ctx context.Context, code, userID string) (*teams.TeamRow, bool, error)
	getTeamPhotoBytes      func(ctx context.Context, teamID string) ([]byte, *string, error)
	getTeamLogoBytes       func(ctx context.Context, teamID string) ([]byte, *string, error)
	updateTeamPhoto        func(ctx context.Context, teamID string, data []byte, mime string) error
	updateTeamLogo         func(ctx context.Context, teamID string, data []byte, mime string) error
	deleteTeamPhoto        func(ctx context.Context, teamID string) error
	deleteTeamLogo         func(ctx context.Context, teamID string) error
}

func (m *mockTeamRepo) ListTeamsForUser(ctx context.Context, userID string) ([]teams.TeamRow, error) {
	return m.listTeamsForUser(ctx, userID)
}

func (m *mockTeamRepo) GetTeam(ctx context.Context, teamID string) (*teams.TeamRow, error) {
	return m.getTeam(ctx, teamID)
}

func (m *mockTeamRepo) CreateTeam(ctx context.Context, name, creatorUserID string, icon, iconBg, iconFg *string) (*teams.TeamRow, error) {
	return m.createTeam(ctx, name, creatorUserID, icon, iconBg, iconFg)
}

func (m *mockTeamRepo) UpdateTeam(ctx context.Context, teamID string, patch teams.TeamPatch) (*teams.TeamRow, error) {
	return m.updateTeam(ctx, teamID, patch)
}

func (m *mockTeamRepo) GetMemberCount(ctx context.Context, teamID string) (int, error) {
	return m.getMemberCount(ctx, teamID)
}

func (m *mockTeamRepo) GetMembership(ctx context.Context, teamID, userID string) (*teams.MembershipRow, error) {
	return m.getMembership(ctx, teamID, userID)
}

func (m *mockTeamRepo) GetRolesForMembership(ctx context.Context, membershipID, teamID string) ([]teams.RoleRow, error) {
	return m.getRolesForMembership(ctx, membershipID, teamID)
}

func (m *mockTeamRepo) GetMemberCounts(ctx context.Context, teamIDs []string) (map[string]int, error) {
	return m.getMemberCounts(ctx, teamIDs)
}

func (m *mockTeamRepo) GetMembershipsForUser(ctx context.Context, teamIDs []string, userID string) (map[string]teams.MembershipRow, error) {
	return m.getMembershipsForUser(ctx, teamIDs, userID)
}

func (m *mockTeamRepo) GetRolesForMemberships(ctx context.Context, membershipIDs []string) (map[string][]teams.RoleRow, error) {
	return m.getRolesForMemberships(ctx, membershipIDs)
}

func (m *mockTeamRepo) CreateInvite(ctx context.Context, teamID string, ttl time.Duration) (*teams.InviteRow, error) {
	return m.createInvite(ctx, teamID, ttl)
}

func (m *mockTeamRepo) AcceptInvite(ctx context.Context, code, userID string) (*teams.TeamRow, bool, error) {
	return m.acceptInvite(ctx, code, userID)
}

func (m *mockTeamRepo) GetTeamPhotoBytes(ctx context.Context, teamID string) (data []byte, mime *string, err error) {
	return m.getTeamPhotoBytes(ctx, teamID)
}

func (m *mockTeamRepo) GetTeamLogoBytes(ctx context.Context, teamID string) (data []byte, mime *string, err error) {
	return m.getTeamLogoBytes(ctx, teamID)
}

func (m *mockTeamRepo) UpdateTeamPhoto(ctx context.Context, teamID string, data []byte, mime string) error {
	return m.updateTeamPhoto(ctx, teamID, data, mime)
}

func (m *mockTeamRepo) UpdateTeamLogo(ctx context.Context, teamID string, data []byte, mime string) error {
	return m.updateTeamLogo(ctx, teamID, data, mime)
}

func (m *mockTeamRepo) DeleteTeamPhoto(ctx context.Context, teamID string) error {
	return m.deleteTeamPhoto(ctx, teamID)
}

func (m *mockTeamRepo) DeleteTeamLogo(ctx context.Context, teamID string) error {
	return m.deleteTeamLogo(ctx, teamID)
}

// ─── helpers ────────────────────────────────────────────────────────────────

func fixedTeamRow(id uuid.UUID) teams.TeamRow {
	return teams.TeamRow{
		Id:                      id,
		Name:                    "Alpha Team",
		CreatedAt:               time.Now(),
		ReasonVisibilityRoleIDs: []uuid.UUID{},
	}
}

func fixedMembershipRow(membershipID, teamID, userID uuid.UUID) *teams.MembershipRow {
	return &teams.MembershipRow{
		Id:       membershipID,
		TeamID:   teamID,
		UserID:   userID,
		JoinedAt: time.Now(),
	}
}

func fixedAdminRole(teamID uuid.UUID) teams.RoleRow {
	return teams.RoleRow{
		Id:     uuid.New(),
		TeamID: teamID,
		Name:   "Admin",
		System: true,
		Permissions: teams.PermissionsJSON{
			Events: "write", Members: "write", Finances: "write",
			News: "write", Polls: "write", Settings: "write",
		},
	}
}

// ─── tests ───────────────────────────────────────────────────────────────────

func TestTeamService_ListForUser(t *testing.T) {
	t.Parallel()

	teamID := uuid.New()
	membershipID := uuid.New()
	userID := uuid.New()

	row := fixedTeamRow(teamID)
	membership := fixedMembershipRow(membershipID, teamID, userID)
	role := fixedAdminRole(teamID)

	repo := &mockTeamRepo{
		listTeamsForUser: func(_ context.Context, uid string) ([]teams.TeamRow, error) {
			assert.Equal(t, userID.String(), uid)
			return []teams.TeamRow{row}, nil
		},
		getMemberCounts: func(_ context.Context, teamIDs []string) (map[string]int, error) {
			assert.Equal(t, []string{teamID.String()}, teamIDs)
			return map[string]int{teamID.String(): 3}, nil
		},
		getMembershipsForUser: func(_ context.Context, teamIDs []string, uid string) (map[string]teams.MembershipRow, error) {
			assert.Equal(t, []string{teamID.String()}, teamIDs)
			assert.Equal(t, userID.String(), uid)
			return map[string]teams.MembershipRow{teamID.String(): *membership}, nil
		},
		getRolesForMemberships: func(_ context.Context, membershipIDs []string) (map[string][]teams.RoleRow, error) {
			assert.Equal(t, []string{membershipID.String()}, membershipIDs)
			return map[string][]teams.RoleRow{membershipID.String(): {role}}, nil
		},
	}

	svc := teams.NewService(repo, "https://app.example.com")
	result, err := svc.ListForUser(context.Background(), userID.String())
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "Alpha Team", result[0].Name)
	assert.Equal(t, 3, result[0].MemberCount)
	assert.Len(t, result[0].MyRoles, 1)
	assert.Equal(t, "write", string(result[0].MyPerms.Events))
}

// Regression test: ListForUser previously called GetMemberCount/GetMembership/
// GetRolesForMembership once per team (3N+1 sequential queries for N teams).
// It must now fetch all three in one batched call each, regardless of how
// many teams the user belongs to.
func TestTeamService_ListForUser_BatchesAcrossMultipleTeams(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	teamA, teamB := uuid.New(), uuid.New()
	membershipA, membershipB := uuid.New(), uuid.New()
	rowA, rowB := fixedTeamRow(teamA), fixedTeamRow(teamB)
	roleA := fixedAdminRole(teamA)

	var memberCountsCalls, membershipsCalls, rolesCalls int
	repo := &mockTeamRepo{
		listTeamsForUser: func(_ context.Context, _ string) ([]teams.TeamRow, error) {
			return []teams.TeamRow{rowA, rowB}, nil
		},
		getMemberCounts: func(_ context.Context, teamIDs []string) (map[string]int, error) {
			memberCountsCalls++
			assert.ElementsMatch(t, []string{teamA.String(), teamB.String()}, teamIDs)
			return map[string]int{teamA.String(): 2, teamB.String(): 5}, nil
		},
		getMembershipsForUser: func(_ context.Context, teamIDs []string, _ string) (map[string]teams.MembershipRow, error) {
			membershipsCalls++
			assert.ElementsMatch(t, []string{teamA.String(), teamB.String()}, teamIDs)
			return map[string]teams.MembershipRow{
				teamA.String(): *fixedMembershipRow(membershipA, teamA, userID),
				teamB.String(): *fixedMembershipRow(membershipB, teamB, userID),
			}, nil
		},
		getRolesForMemberships: func(_ context.Context, membershipIDs []string) (map[string][]teams.RoleRow, error) {
			rolesCalls++
			assert.ElementsMatch(t, []string{membershipA.String(), membershipB.String()}, membershipIDs)
			return map[string][]teams.RoleRow{membershipA.String(): {roleA}}, nil
		},
	}

	svc := teams.NewService(repo, "https://app.example.com")
	result, err := svc.ListForUser(context.Background(), userID.String())
	require.NoError(t, err)
	require.Len(t, result, 2)
	assert.Equal(t, 1, memberCountsCalls)
	assert.Equal(t, 1, membershipsCalls)
	assert.Equal(t, 1, rolesCalls)
}

func TestCreateInvite_BuildsLinkFromPublicBaseURL(t *testing.T) {
	teamID := uuid.New()
	inviteID := uuid.New()
	now := time.Now()

	repo := &mockTeamRepo{
		createInvite: func(_ context.Context, tid string, ttl time.Duration) (*teams.InviteRow, error) {
			assert.Equal(t, teamID.String(), tid)
			assert.Equal(t, 7*24*time.Hour, ttl)
			return &teams.InviteRow{
				Id:        inviteID,
				TeamID:    teamID,
				Code:      "ABC123",
				ExpiresAt: now.Add(ttl),
				CreatedAt: now,
			}, nil
		},
	}

	// A trailing slash on the base URL must not produce a double slash.
	svc := teams.NewService(repo, "https://app.example.com/")
	inv, err := svc.CreateInvite(context.Background(), teamID.String())
	require.NoError(t, err)
	assert.Equal(t, "https://app.example.com/join/"+teamID.String()+"/ABC123", inv.Link)
	assert.Equal(t, "ABC123", inv.Code)
}

func TestTeamService_AcceptInvite_ReturnsEnrichedTeam(t *testing.T) {
	t.Parallel()

	teamID := uuid.New()
	membershipID := uuid.New()
	userID := uuid.New()

	row := fixedTeamRow(teamID)
	membership := fixedMembershipRow(membershipID, teamID, userID)
	role := fixedAdminRole(teamID)

	repo := &mockTeamRepo{
		acceptInvite: func(_ context.Context, code, uid string) (*teams.TeamRow, bool, error) {
			assert.Equal(t, "ABC123", code)
			assert.Equal(t, userID.String(), uid)
			return &row, false, nil
		},
		getMemberCount: func(_ context.Context, _ string) (int, error) { return 4, nil },
		getMembership:  func(_ context.Context, _, _ string) (*teams.MembershipRow, error) { return membership, nil },
		getRolesForMembership: func(_ context.Context, _, _ string) ([]teams.RoleRow, error) {
			return []teams.RoleRow{role}, nil
		},
	}

	svc := teams.NewService(repo, "https://app.example.com")
	tfu, err := svc.AcceptInvite(context.Background(), "ABC123", userID.String())
	require.NoError(t, err)
	assert.Equal(t, teamID, tfu.Id)
	assert.Equal(t, 4, tfu.MemberCount)
	assert.False(t, tfu.AlreadyMember)
}

func TestTeamService_AcceptInvite_PropagatesAlreadyMember(t *testing.T) {
	t.Parallel()

	teamID := uuid.New()
	membershipID := uuid.New()
	userID := uuid.New()

	row := fixedTeamRow(teamID)
	membership := fixedMembershipRow(membershipID, teamID, userID)
	role := fixedAdminRole(teamID)

	repo := &mockTeamRepo{
		acceptInvite: func(_ context.Context, _, _ string) (*teams.TeamRow, bool, error) {
			return &row, true, nil
		},
		getMemberCount: func(_ context.Context, _ string) (int, error) { return 4, nil },
		getMembership:  func(_ context.Context, _, _ string) (*teams.MembershipRow, error) { return membership, nil },
		getRolesForMembership: func(_ context.Context, _, _ string) ([]teams.RoleRow, error) {
			return []teams.RoleRow{role}, nil
		},
	}

	svc := teams.NewService(repo, "https://app.example.com")
	tfu, err := svc.AcceptInvite(context.Background(), "ABC123", userID.String())
	require.NoError(t, err)
	assert.True(t, tfu.AlreadyMember)
}

func TestTeamService_AcceptInvite_PropagatesErrInviteNotFound(t *testing.T) {
	t.Parallel()

	repo := &mockTeamRepo{
		acceptInvite: func(_ context.Context, _, _ string) (*teams.TeamRow, bool, error) {
			return nil, false, teams.ErrInviteNotFound
		},
	}

	svc := teams.NewService(repo, "https://app.example.com")
	_, err := svc.AcceptInvite(context.Background(), "expired-or-unknown", uuid.New().String())
	require.ErrorIs(t, err, teams.ErrInviteNotFound)
}

func TestTeamService_UpdateLogo_StoresResizedJPEGAndReturnsTeam(t *testing.T) {
	teamID := uuid.New()
	row := fixedTeamRow(teamID)
	row.HasLogo = true // stand-in for "stored" bytes on refresh

	var storedData []byte
	var storedMime string
	repo := &mockTeamRepo{
		updateTeamLogo: func(_ context.Context, tid string, data []byte, mime string) error {
			assert.Equal(t, teamID.String(), tid)
			storedData, storedMime = data, mime
			return nil
		},
		getTeam: func(_ context.Context, _ string) (*teams.TeamRow, error) { return &row, nil },
	}

	svc := teams.NewService(repo, "https://app.example.com")
	result, err := svc.UpdateLogo(context.Background(), teamID.String(), fixedJPEG(t), "image/jpeg")
	require.NoError(t, err)
	assert.Equal(t, "image/jpeg", storedMime)
	assert.NotEmpty(t, storedData)
	assert.True(t, *result.HasLogo)
}

func TestTeamService_DeletePhoto_ClearsStoredPhoto(t *testing.T) {
	teamID := uuid.New()
	called := false
	repo := &mockTeamRepo{
		deleteTeamPhoto: func(_ context.Context, tid string) error {
			assert.Equal(t, teamID.String(), tid)
			called = true
			return nil
		},
	}
	svc := teams.NewService(repo, "https://app.example.com")
	err := svc.DeletePhoto(context.Background(), teamID.String())
	require.NoError(t, err)
	assert.True(t, called)
}

func TestTeamService_DeletePhoto_WrongTeam_PropagatesNoRows(t *testing.T) {
	repo := &mockTeamRepo{
		deleteTeamPhoto: func(context.Context, string) error { return pgx.ErrNoRows },
	}
	svc := teams.NewService(repo, "https://app.example.com")
	err := svc.DeletePhoto(context.Background(), uuid.New().String())
	require.ErrorIs(t, err, pgx.ErrNoRows)
}

func TestTeamService_DeleteLogo_ClearsStoredLogo(t *testing.T) {
	teamID := uuid.New()
	called := false
	repo := &mockTeamRepo{
		deleteTeamLogo: func(_ context.Context, tid string) error {
			assert.Equal(t, teamID.String(), tid)
			called = true
			return nil
		},
	}
	svc := teams.NewService(repo, "https://app.example.com")
	err := svc.DeleteLogo(context.Background(), teamID.String())
	require.NoError(t, err)
	assert.True(t, called)
}

func TestTeamService_DeleteLogo_WrongTeam_PropagatesNoRows(t *testing.T) {
	repo := &mockTeamRepo{
		deleteTeamLogo: func(context.Context, string) error { return pgx.ErrNoRows },
	}
	svc := teams.NewService(repo, "https://app.example.com")
	err := svc.DeleteLogo(context.Background(), uuid.New().String())
	require.ErrorIs(t, err, pgx.ErrNoRows)
}

// A PNG that declares dimensions exceeding the decompression-bomb guard must
// be rejected before decode — full decode of e.g. 20000x20000 would allocate
// ~1.6 GB for a single upload.
func TestTeamService_UpdateLogo_RejectsOversizedImage(t *testing.T) {
	repo := &mockTeamRepo{
		updateTeamLogo: func(context.Context, string, []byte, string) error {
			t.Fatal("must not store an oversized image")
			return nil
		},
	}
	svc := teams.NewService(repo, "https://app.example.com")

	oversized := fakeOversizedPNGHeader(20000, 20000)
	_, err := svc.UpdateLogo(context.Background(), uuid.New().String(), oversized, "image/png")

	require.Error(t, err)
	assert.ErrorIs(t, err, teams.ErrImageTooLarge)
}

func TestTeamService_UpdatePhoto_RejectsOversizedImage(t *testing.T) {
	repo := &mockTeamRepo{
		updateTeamPhoto: func(context.Context, string, []byte, string) error {
			t.Fatal("must not store an oversized image")
			return nil
		},
	}
	svc := teams.NewService(repo, "https://app.example.com")

	oversized := fakeOversizedPNGHeader(20000, 20000)
	_, err := svc.UpdatePhoto(context.Background(), uuid.New().String(), oversized, "image/png")

	require.Error(t, err)
	assert.ErrorIs(t, err, teams.ErrImageTooLarge)
}

func TestTeamService_GetTeamLogoData_ReturnsStoredBytes(t *testing.T) {
	teamID := uuid.New()
	mime := "image/jpeg"

	repo := &mockTeamRepo{
		getTeamLogoBytes: func(_ context.Context, _ string) ([]byte, *string, error) {
			return []byte{1, 2, 3}, &mime, nil
		},
	}

	svc := teams.NewService(repo, "https://app.example.com")
	data, gotMime, err := svc.GetTeamLogoData(context.Background(), teamID.String())
	require.NoError(t, err)
	assert.Equal(t, []byte{1, 2, 3}, data)
	assert.Equal(t, "image/jpeg", gotMime)
}

func TestTeamService_GetTeamLogoData_NoLogoReturnsErrNoRows(t *testing.T) {
	teamID := uuid.New()

	repo := &mockTeamRepo{
		getTeamLogoBytes: func(_ context.Context, _ string) ([]byte, *string, error) {
			return nil, nil, nil
		},
	}

	svc := teams.NewService(repo, "https://app.example.com")
	_, _, err := svc.GetTeamLogoData(context.Background(), teamID.String())
	require.ErrorIs(t, err, pgx.ErrNoRows)
}
