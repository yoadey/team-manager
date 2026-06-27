// Package server wires together the feature handlers and presents a single
// implementation of gen.StrictServerInterface.
package server

import (
	"context"
	"errors"

	"github.com/yoadey/team-manager/backend/internal/absences"
	"github.com/yoadey/team-manager/backend/internal/auth"
	"github.com/yoadey/team-manager/backend/internal/finances"
	"github.com/yoadey/team-manager/backend/internal/gen"
	"github.com/yoadey/team-manager/backend/internal/members"
	"github.com/yoadey/team-manager/backend/internal/news"
	"github.com/yoadey/team-manager/backend/internal/notifications"
	"github.com/yoadey/team-manager/backend/internal/polls"
	"github.com/yoadey/team-manager/backend/internal/roles"
	"github.com/yoadey/team-manager/backend/internal/stats"
	"github.com/yoadey/team-manager/backend/internal/teams"
)

// errNotImplemented is returned by stub methods that have not been implemented yet.
var errNotImplemented = errors.New("not implemented")

// StrictUnimplemented satisfies every method of gen.StrictServerInterface by
// returning a 501 Not Implemented error. Feature handlers override the methods
// they handle by embedding this struct and shadowing the relevant methods.
type StrictUnimplemented struct{}

func (StrictUnimplemented) Login(_ context.Context, _ gen.LoginRequestObject) (gen.LoginResponseObject, error) {
	return nil, errNotImplemented
}

func (StrictUnimplemented) Logout(_ context.Context, _ gen.LogoutRequestObject) (gen.LogoutResponseObject, error) {
	return nil, errNotImplemented
}

func (StrictUnimplemented) GetCurrentUser(_ context.Context, _ gen.GetCurrentUserRequestObject) (gen.GetCurrentUserResponseObject, error) {
	return nil, errNotImplemented
}

func (StrictUnimplemented) GetMyPhoto(_ context.Context, _ gen.GetMyPhotoRequestObject) (gen.GetMyPhotoResponseObject, error) {
	return nil, errNotImplemented
}

func (StrictUnimplemented) UploadMyPhoto(_ context.Context, _ gen.UploadMyPhotoRequestObject) (gen.UploadMyPhotoResponseObject, error) {
	return nil, errNotImplemented
}

func (StrictUnimplemented) ListProviders(_ context.Context, _ gen.ListProvidersRequestObject) (gen.ListProvidersResponseObject, error) {
	return nil, errNotImplemented
}

func (StrictUnimplemented) ListTeams(_ context.Context, _ gen.ListTeamsRequestObject) (gen.ListTeamsResponseObject, error) {
	return nil, errNotImplemented
}

func (StrictUnimplemented) CreateTeam(_ context.Context, _ gen.CreateTeamRequestObject) (gen.CreateTeamResponseObject, error) {
	return nil, errNotImplemented
}

func (StrictUnimplemented) GetTeam(_ context.Context, _ gen.GetTeamRequestObject) (gen.GetTeamResponseObject, error) {
	return nil, errNotImplemented
}

func (StrictUnimplemented) UpdateTeam(_ context.Context, _ gen.UpdateTeamRequestObject) (gen.UpdateTeamResponseObject, error) {
	return nil, errNotImplemented
}

func (StrictUnimplemented) CreateInvite(_ context.Context, _ gen.CreateInviteRequestObject) (gen.CreateInviteResponseObject, error) {
	return nil, errNotImplemented
}

func (StrictUnimplemented) GetTeamLogo(_ context.Context, _ gen.GetTeamLogoRequestObject) (gen.GetTeamLogoResponseObject, error) {
	return nil, errNotImplemented
}

func (StrictUnimplemented) UploadTeamLogo(_ context.Context, _ gen.UploadTeamLogoRequestObject) (gen.UploadTeamLogoResponseObject, error) {
	return nil, errNotImplemented
}

func (StrictUnimplemented) GetTeamPhoto(_ context.Context, _ gen.GetTeamPhotoRequestObject) (gen.GetTeamPhotoResponseObject, error) {
	return nil, errNotImplemented
}

func (StrictUnimplemented) UploadTeamPhoto(_ context.Context, _ gen.UploadTeamPhotoRequestObject) (gen.UploadTeamPhotoResponseObject, error) {
	return nil, errNotImplemented
}

func (StrictUnimplemented) ListMembers(_ context.Context, _ gen.ListMembersRequestObject) (gen.ListMembersResponseObject, error) {
	return nil, errNotImplemented
}

func (StrictUnimplemented) AddMember(_ context.Context, _ gen.AddMemberRequestObject) (gen.AddMemberResponseObject, error) {
	return nil, errNotImplemented
}

func (StrictUnimplemented) RemoveMember(_ context.Context, _ gen.RemoveMemberRequestObject) (gen.RemoveMemberResponseObject, error) {
	return nil, errNotImplemented
}

func (StrictUnimplemented) UpdateMember(_ context.Context, _ gen.UpdateMemberRequestObject) (gen.UpdateMemberResponseObject, error) {
	return nil, errNotImplemented
}

func (StrictUnimplemented) SetMemberRoles(_ context.Context, _ gen.SetMemberRolesRequestObject) (gen.SetMemberRolesResponseObject, error) {
	return nil, errNotImplemented
}

func (StrictUnimplemented) ListRoles(_ context.Context, _ gen.ListRolesRequestObject) (gen.ListRolesResponseObject, error) {
	return nil, errNotImplemented
}

func (StrictUnimplemented) CreateRole(_ context.Context, _ gen.CreateRoleRequestObject) (gen.CreateRoleResponseObject, error) {
	return nil, errNotImplemented
}

func (StrictUnimplemented) DeleteRole(_ context.Context, _ gen.DeleteRoleRequestObject) (gen.DeleteRoleResponseObject, error) {
	return nil, errNotImplemented
}

func (StrictUnimplemented) UpdateRole(_ context.Context, _ gen.UpdateRoleRequestObject) (gen.UpdateRoleResponseObject, error) {
	return nil, errNotImplemented
}

func (StrictUnimplemented) ListEvents(_ context.Context, _ gen.ListEventsRequestObject) (gen.ListEventsResponseObject, error) {
	return nil, errNotImplemented
}

func (StrictUnimplemented) CreateEvent(_ context.Context, _ gen.CreateEventRequestObject) (gen.CreateEventResponseObject, error) {
	return nil, errNotImplemented
}

func (StrictUnimplemented) DeleteEvent(_ context.Context, _ gen.DeleteEventRequestObject) (gen.DeleteEventResponseObject, error) {
	return nil, errNotImplemented
}

func (StrictUnimplemented) GetEvent(_ context.Context, _ gen.GetEventRequestObject) (gen.GetEventResponseObject, error) {
	return nil, errNotImplemented
}

func (StrictUnimplemented) UpdateEvent(_ context.Context, _ gen.UpdateEventRequestObject) (gen.UpdateEventResponseObject, error) {
	return nil, errNotImplemented
}

func (StrictUnimplemented) ListAttendance(_ context.Context, _ gen.ListAttendanceRequestObject) (gen.ListAttendanceResponseObject, error) {
	return nil, errNotImplemented
}

func (StrictUnimplemented) SetAttendance(_ context.Context, _ gen.SetAttendanceRequestObject) (gen.SetAttendanceResponseObject, error) {
	return nil, errNotImplemented
}

func (StrictUnimplemented) SetNomination(_ context.Context, _ gen.SetNominationRequestObject) (gen.SetNominationResponseObject, error) {
	return nil, errNotImplemented
}

func (StrictUnimplemented) ListEventComments(_ context.Context, _ gen.ListEventCommentsRequestObject) (gen.ListEventCommentsResponseObject, error) {
	return nil, errNotImplemented
}

func (StrictUnimplemented) AddEventComment(_ context.Context, _ gen.AddEventCommentRequestObject) (gen.AddEventCommentResponseObject, error) {
	return nil, errNotImplemented
}

func (StrictUnimplemented) DeleteEventComment(_ context.Context, _ gen.DeleteEventCommentRequestObject) (gen.DeleteEventCommentResponseObject, error) {
	return nil, errNotImplemented
}

func (StrictUnimplemented) SetEventStatus(_ context.Context, _ gen.SetEventStatusRequestObject) (gen.SetEventStatusResponseObject, error) {
	return nil, errNotImplemented
}

func (StrictUnimplemented) ListAbsences(_ context.Context, _ gen.ListAbsencesRequestObject) (gen.ListAbsencesResponseObject, error) {
	return nil, errNotImplemented
}

func (StrictUnimplemented) CreateAbsence(_ context.Context, _ gen.CreateAbsenceRequestObject) (gen.CreateAbsenceResponseObject, error) {
	return nil, errNotImplemented
}

func (StrictUnimplemented) ListMyAbsences(_ context.Context, _ gen.ListMyAbsencesRequestObject) (gen.ListMyAbsencesResponseObject, error) {
	return nil, errNotImplemented
}

func (StrictUnimplemented) DeleteAbsence(_ context.Context, _ gen.DeleteAbsenceRequestObject) (gen.DeleteAbsenceResponseObject, error) {
	return nil, errNotImplemented
}

func (StrictUnimplemented) UpdateAbsence(_ context.Context, _ gen.UpdateAbsenceRequestObject) (gen.UpdateAbsenceResponseObject, error) {
	return nil, errNotImplemented
}

func (StrictUnimplemented) GetFinanceOverview(_ context.Context, _ gen.GetFinanceOverviewRequestObject) (gen.GetFinanceOverviewResponseObject, error) {
	return nil, errNotImplemented
}

func (StrictUnimplemented) UpdateContribution(_ context.Context, _ gen.UpdateContributionRequestObject) (gen.UpdateContributionResponseObject, error) {
	return nil, errNotImplemented
}

func (StrictUnimplemented) ToggleContribution(_ context.Context, _ gen.ToggleContributionRequestObject) (gen.ToggleContributionResponseObject, error) {
	return nil, errNotImplemented
}

func (StrictUnimplemented) CreatePenalty(_ context.Context, _ gen.CreatePenaltyRequestObject) (gen.CreatePenaltyResponseObject, error) {
	return nil, errNotImplemented
}

func (StrictUnimplemented) DeletePenalty(_ context.Context, _ gen.DeletePenaltyRequestObject) (gen.DeletePenaltyResponseObject, error) {
	return nil, errNotImplemented
}

func (StrictUnimplemented) UpdatePenalty(_ context.Context, _ gen.UpdatePenaltyRequestObject) (gen.UpdatePenaltyResponseObject, error) {
	return nil, errNotImplemented
}

func (StrictUnimplemented) CreatePenaltyAssignment(_ context.Context, _ gen.CreatePenaltyAssignmentRequestObject) (gen.CreatePenaltyAssignmentResponseObject, error) {
	return nil, errNotImplemented
}

func (StrictUnimplemented) DeletePenaltyAssignment(_ context.Context, _ gen.DeletePenaltyAssignmentRequestObject) (gen.DeletePenaltyAssignmentResponseObject, error) {
	return nil, errNotImplemented
}

func (StrictUnimplemented) TogglePenaltyPaid(_ context.Context, _ gen.TogglePenaltyPaidRequestObject) (gen.TogglePenaltyPaidResponseObject, error) {
	return nil, errNotImplemented
}

func (StrictUnimplemented) CreateTransaction(_ context.Context, _ gen.CreateTransactionRequestObject) (gen.CreateTransactionResponseObject, error) {
	return nil, errNotImplemented
}

func (StrictUnimplemented) DeleteTransaction(_ context.Context, _ gen.DeleteTransactionRequestObject) (gen.DeleteTransactionResponseObject, error) {
	return nil, errNotImplemented
}

func (StrictUnimplemented) UpdateTransaction(_ context.Context, _ gen.UpdateTransactionRequestObject) (gen.UpdateTransactionResponseObject, error) {
	return nil, errNotImplemented
}

func (StrictUnimplemented) ListNews(_ context.Context, _ gen.ListNewsRequestObject) (gen.ListNewsResponseObject, error) {
	return nil, errNotImplemented
}

func (StrictUnimplemented) CreateNews(_ context.Context, _ gen.CreateNewsRequestObject) (gen.CreateNewsResponseObject, error) {
	return nil, errNotImplemented
}

func (StrictUnimplemented) DeleteNews(_ context.Context, _ gen.DeleteNewsRequestObject) (gen.DeleteNewsResponseObject, error) {
	return nil, errNotImplemented
}

func (StrictUnimplemented) UpdateNews(_ context.Context, _ gen.UpdateNewsRequestObject) (gen.UpdateNewsResponseObject, error) {
	return nil, errNotImplemented
}

func (StrictUnimplemented) ListNotifications(_ context.Context, _ gen.ListNotificationsRequestObject) (gen.ListNotificationsResponseObject, error) {
	return nil, errNotImplemented
}

func (StrictUnimplemented) MarkNotificationsSeen(_ context.Context, _ gen.MarkNotificationsSeenRequestObject) (gen.MarkNotificationsSeenResponseObject, error) {
	return nil, errNotImplemented
}

func (StrictUnimplemented) ListPolls(_ context.Context, _ gen.ListPollsRequestObject) (gen.ListPollsResponseObject, error) {
	return nil, errNotImplemented
}

func (StrictUnimplemented) CreatePoll(_ context.Context, _ gen.CreatePollRequestObject) (gen.CreatePollResponseObject, error) {
	return nil, errNotImplemented
}

func (StrictUnimplemented) DeletePoll(_ context.Context, _ gen.DeletePollRequestObject) (gen.DeletePollResponseObject, error) {
	return nil, errNotImplemented
}

func (StrictUnimplemented) VotePoll(_ context.Context, _ gen.VotePollRequestObject) (gen.VotePollResponseObject, error) {
	return nil, errNotImplemented
}

func (StrictUnimplemented) GetStatsOverview(_ context.Context, _ gen.GetStatsOverviewRequestObject) (gen.GetStatsOverviewResponseObject, error) {
	return nil, errNotImplemented
}

func (StrictUnimplemented) GetMemberStats(_ context.Context, _ gen.GetMemberStatsRequestObject) (gen.GetMemberStatsResponseObject, error) {
	return nil, errNotImplemented
}

// ─── Server ──────────────────────────────────────────────────────────────────

// Server implements gen.StrictServerInterface by composing feature handlers.
type Server struct {
	StrictUnimplemented
	Auth    *auth.Handler
	Teams   *teams.Handler
	Members *members.Handler
	Roles   *roles.Handler
	Events  interface {
		ListEvents(ctx context.Context, request gen.ListEventsRequestObject) (gen.ListEventsResponseObject, error)
		CreateEvent(ctx context.Context, request gen.CreateEventRequestObject) (gen.CreateEventResponseObject, error)
		GetEvent(ctx context.Context, request gen.GetEventRequestObject) (gen.GetEventResponseObject, error)
		UpdateEvent(ctx context.Context, request gen.UpdateEventRequestObject) (gen.UpdateEventResponseObject, error)
		DeleteEvent(ctx context.Context, request gen.DeleteEventRequestObject) (gen.DeleteEventResponseObject, error)
		SetEventStatus(ctx context.Context, request gen.SetEventStatusRequestObject) (gen.SetEventStatusResponseObject, error)
		ListEventComments(ctx context.Context, request gen.ListEventCommentsRequestObject) (gen.ListEventCommentsResponseObject, error)
		AddEventComment(ctx context.Context, request gen.AddEventCommentRequestObject) (gen.AddEventCommentResponseObject, error)
		DeleteEventComment(ctx context.Context, request gen.DeleteEventCommentRequestObject) (gen.DeleteEventCommentResponseObject, error)
		ListAttendance(ctx context.Context, request gen.ListAttendanceRequestObject) (gen.ListAttendanceResponseObject, error)
		SetAttendance(ctx context.Context, request gen.SetAttendanceRequestObject) (gen.SetAttendanceResponseObject, error)
		SetNomination(ctx context.Context, request gen.SetNominationRequestObject) (gen.SetNominationResponseObject, error)
	}
	Absences      *absences.Handler
	News          *news.Handler
	Polls         *polls.Handler
	Notifications *notifications.Handler
	Finances      *finances.Handler
	Stats         *stats.Handler
}

// New creates a Server wired to the provided feature handlers.
func New(
	authHandler *auth.Handler,
	teamsHandler *teams.Handler,
	membersHandler *members.Handler,
	rolesHandler *roles.Handler,
	eventsHandler interface {
		ListEvents(ctx context.Context, request gen.ListEventsRequestObject) (gen.ListEventsResponseObject, error)
		CreateEvent(ctx context.Context, request gen.CreateEventRequestObject) (gen.CreateEventResponseObject, error)
		GetEvent(ctx context.Context, request gen.GetEventRequestObject) (gen.GetEventResponseObject, error)
		UpdateEvent(ctx context.Context, request gen.UpdateEventRequestObject) (gen.UpdateEventResponseObject, error)
		DeleteEvent(ctx context.Context, request gen.DeleteEventRequestObject) (gen.DeleteEventResponseObject, error)
		SetEventStatus(ctx context.Context, request gen.SetEventStatusRequestObject) (gen.SetEventStatusResponseObject, error)
		ListEventComments(ctx context.Context, request gen.ListEventCommentsRequestObject) (gen.ListEventCommentsResponseObject, error)
		AddEventComment(ctx context.Context, request gen.AddEventCommentRequestObject) (gen.AddEventCommentResponseObject, error)
		DeleteEventComment(ctx context.Context, request gen.DeleteEventCommentRequestObject) (gen.DeleteEventCommentResponseObject, error)
		ListAttendance(ctx context.Context, request gen.ListAttendanceRequestObject) (gen.ListAttendanceResponseObject, error)
		SetAttendance(ctx context.Context, request gen.SetAttendanceRequestObject) (gen.SetAttendanceResponseObject, error)
		SetNomination(ctx context.Context, request gen.SetNominationRequestObject) (gen.SetNominationResponseObject, error)
	},
	absencesHandler *absences.Handler,
	newsHandler *news.Handler,
	pollsHandler *polls.Handler,
	notificationsHandler *notifications.Handler,
	financesHandler *finances.Handler,
	statsHandler *stats.Handler,
) *Server {
	return &Server{
		Auth:          authHandler,
		Teams:         teamsHandler,
		Members:       membersHandler,
		Roles:         rolesHandler,
		Events:        eventsHandler,
		Absences:      absencesHandler,
		News:          newsHandler,
		Polls:         pollsHandler,
		Notifications: notificationsHandler,
		Finances:      financesHandler,
		Stats:         statsHandler,
	}
}

// ─── Auth delegations ─────────────────────────────────────────────────────────

func (s *Server) Login(ctx context.Context, req gen.LoginRequestObject) (gen.LoginResponseObject, error) {
	return s.Auth.Login(ctx, req)
}

func (s *Server) Logout(ctx context.Context, req gen.LogoutRequestObject) (gen.LogoutResponseObject, error) {
	return s.Auth.Logout(ctx, req)
}

func (s *Server) GetCurrentUser(ctx context.Context, req gen.GetCurrentUserRequestObject) (gen.GetCurrentUserResponseObject, error) {
	return s.Auth.GetCurrentUser(ctx, req)
}

func (s *Server) DeleteCurrentUser(ctx context.Context, req gen.DeleteCurrentUserRequestObject) (gen.DeleteCurrentUserResponseObject, error) {
	return s.Auth.DeleteCurrentUser(ctx, req)
}

func (s *Server) GetMyDataExport(ctx context.Context, req gen.GetMyDataExportRequestObject) (gen.GetMyDataExportResponseObject, error) {
	return s.Auth.GetMyDataExport(ctx, req)
}

func (s *Server) GetMyPhoto(ctx context.Context, req gen.GetMyPhotoRequestObject) (gen.GetMyPhotoResponseObject, error) {
	return s.Auth.GetMyPhoto(ctx, req)
}

func (s *Server) UploadMyPhoto(ctx context.Context, req gen.UploadMyPhotoRequestObject) (gen.UploadMyPhotoResponseObject, error) {
	return s.Auth.UploadMyPhoto(ctx, req)
}

func (s *Server) ListProviders(ctx context.Context, req gen.ListProvidersRequestObject) (gen.ListProvidersResponseObject, error) {
	return s.Auth.ListProviders(ctx, req)
}

// ─── Teams delegations ────────────────────────────────────────────────────────

func (s *Server) ListTeams(ctx context.Context, req gen.ListTeamsRequestObject) (gen.ListTeamsResponseObject, error) {
	return s.Teams.ListTeams(ctx, req)
}

func (s *Server) CreateTeam(ctx context.Context, req gen.CreateTeamRequestObject) (gen.CreateTeamResponseObject, error) {
	return s.Teams.CreateTeam(ctx, req)
}

func (s *Server) GetTeam(ctx context.Context, req gen.GetTeamRequestObject) (gen.GetTeamResponseObject, error) {
	return s.Teams.GetTeam(ctx, req)
}

func (s *Server) UpdateTeam(ctx context.Context, req gen.UpdateTeamRequestObject) (gen.UpdateTeamResponseObject, error) {
	return s.Teams.UpdateTeam(ctx, req)
}

func (s *Server) CreateInvite(ctx context.Context, req gen.CreateInviteRequestObject) (gen.CreateInviteResponseObject, error) {
	return s.Teams.CreateInvite(ctx, req)
}

func (s *Server) GetTeamLogo(ctx context.Context, req gen.GetTeamLogoRequestObject) (gen.GetTeamLogoResponseObject, error) {
	return s.Teams.GetTeamLogo(ctx, req)
}

func (s *Server) UploadTeamLogo(ctx context.Context, req gen.UploadTeamLogoRequestObject) (gen.UploadTeamLogoResponseObject, error) {
	return s.Teams.UploadTeamLogo(ctx, req)
}

func (s *Server) GetTeamPhoto(ctx context.Context, req gen.GetTeamPhotoRequestObject) (gen.GetTeamPhotoResponseObject, error) {
	return s.Teams.GetTeamPhoto(ctx, req)
}

func (s *Server) UploadTeamPhoto(ctx context.Context, req gen.UploadTeamPhotoRequestObject) (gen.UploadTeamPhotoResponseObject, error) {
	return s.Teams.UploadTeamPhoto(ctx, req)
}

// ─── Members delegations ──────────────────────────────────────────────────────

func (s *Server) ListMembers(ctx context.Context, req gen.ListMembersRequestObject) (gen.ListMembersResponseObject, error) {
	return s.Members.ListMembers(ctx, req)
}

func (s *Server) AddMember(ctx context.Context, req gen.AddMemberRequestObject) (gen.AddMemberResponseObject, error) {
	return s.Members.AddMember(ctx, req)
}

func (s *Server) RemoveMember(ctx context.Context, req gen.RemoveMemberRequestObject) (gen.RemoveMemberResponseObject, error) {
	return s.Members.RemoveMember(ctx, req)
}

func (s *Server) UpdateMember(ctx context.Context, req gen.UpdateMemberRequestObject) (gen.UpdateMemberResponseObject, error) {
	return s.Members.UpdateMember(ctx, req)
}

func (s *Server) SetMemberRoles(ctx context.Context, req gen.SetMemberRolesRequestObject) (gen.SetMemberRolesResponseObject, error) {
	return s.Members.SetMemberRoles(ctx, req)
}

// ─── Roles delegations ────────────────────────────────────────────────────────

func (s *Server) ListRoles(ctx context.Context, req gen.ListRolesRequestObject) (gen.ListRolesResponseObject, error) {
	return s.Roles.ListRoles(ctx, req)
}

func (s *Server) CreateRole(ctx context.Context, req gen.CreateRoleRequestObject) (gen.CreateRoleResponseObject, error) {
	return s.Roles.CreateRole(ctx, req)
}

func (s *Server) DeleteRole(ctx context.Context, req gen.DeleteRoleRequestObject) (gen.DeleteRoleResponseObject, error) {
	return s.Roles.DeleteRole(ctx, req)
}

func (s *Server) UpdateRole(ctx context.Context, req gen.UpdateRoleRequestObject) (gen.UpdateRoleResponseObject, error) {
	return s.Roles.UpdateRole(ctx, req)
}

// ─── Events delegations ───────────────────────────────────────────────────────

func (s *Server) ListEvents(ctx context.Context, req gen.ListEventsRequestObject) (gen.ListEventsResponseObject, error) {
	return s.Events.ListEvents(ctx, req)
}

func (s *Server) CreateEvent(ctx context.Context, req gen.CreateEventRequestObject) (gen.CreateEventResponseObject, error) {
	return s.Events.CreateEvent(ctx, req)
}

func (s *Server) GetEvent(ctx context.Context, req gen.GetEventRequestObject) (gen.GetEventResponseObject, error) {
	return s.Events.GetEvent(ctx, req)
}

func (s *Server) UpdateEvent(ctx context.Context, req gen.UpdateEventRequestObject) (gen.UpdateEventResponseObject, error) {
	return s.Events.UpdateEvent(ctx, req)
}

func (s *Server) DeleteEvent(ctx context.Context, req gen.DeleteEventRequestObject) (gen.DeleteEventResponseObject, error) {
	return s.Events.DeleteEvent(ctx, req)
}

func (s *Server) SetEventStatus(ctx context.Context, req gen.SetEventStatusRequestObject) (gen.SetEventStatusResponseObject, error) {
	return s.Events.SetEventStatus(ctx, req)
}

func (s *Server) ListEventComments(ctx context.Context, req gen.ListEventCommentsRequestObject) (gen.ListEventCommentsResponseObject, error) {
	return s.Events.ListEventComments(ctx, req)
}

func (s *Server) AddEventComment(ctx context.Context, req gen.AddEventCommentRequestObject) (gen.AddEventCommentResponseObject, error) {
	return s.Events.AddEventComment(ctx, req)
}

func (s *Server) DeleteEventComment(ctx context.Context, req gen.DeleteEventCommentRequestObject) (gen.DeleteEventCommentResponseObject, error) {
	return s.Events.DeleteEventComment(ctx, req)
}

func (s *Server) ListAttendance(ctx context.Context, req gen.ListAttendanceRequestObject) (gen.ListAttendanceResponseObject, error) {
	return s.Events.ListAttendance(ctx, req)
}

func (s *Server) SetAttendance(ctx context.Context, req gen.SetAttendanceRequestObject) (gen.SetAttendanceResponseObject, error) {
	return s.Events.SetAttendance(ctx, req)
}

func (s *Server) SetNomination(ctx context.Context, req gen.SetNominationRequestObject) (gen.SetNominationResponseObject, error) {
	return s.Events.SetNomination(ctx, req)
}

// ─── Absences delegations ─────────────────────────────────────────────────────

func (s *Server) ListAbsences(ctx context.Context, req gen.ListAbsencesRequestObject) (gen.ListAbsencesResponseObject, error) {
	return s.Absences.ListAbsences(ctx, req)
}

func (s *Server) CreateAbsence(ctx context.Context, req gen.CreateAbsenceRequestObject) (gen.CreateAbsenceResponseObject, error) {
	return s.Absences.CreateAbsence(ctx, req)
}

func (s *Server) ListMyAbsences(ctx context.Context, req gen.ListMyAbsencesRequestObject) (gen.ListMyAbsencesResponseObject, error) {
	return s.Absences.ListMyAbsences(ctx, req)
}

func (s *Server) DeleteAbsence(ctx context.Context, req gen.DeleteAbsenceRequestObject) (gen.DeleteAbsenceResponseObject, error) {
	return s.Absences.DeleteAbsence(ctx, req)
}

func (s *Server) UpdateAbsence(ctx context.Context, req gen.UpdateAbsenceRequestObject) (gen.UpdateAbsenceResponseObject, error) {
	return s.Absences.UpdateAbsence(ctx, req)
}

// ─── News delegations ─────────────────────────────────────────────────────────

func (s *Server) ListNews(ctx context.Context, req gen.ListNewsRequestObject) (gen.ListNewsResponseObject, error) {
	return s.News.ListNews(ctx, req)
}

func (s *Server) CreateNews(ctx context.Context, req gen.CreateNewsRequestObject) (gen.CreateNewsResponseObject, error) {
	return s.News.CreateNews(ctx, req)
}

func (s *Server) UpdateNews(ctx context.Context, req gen.UpdateNewsRequestObject) (gen.UpdateNewsResponseObject, error) {
	return s.News.UpdateNews(ctx, req)
}

func (s *Server) DeleteNews(ctx context.Context, req gen.DeleteNewsRequestObject) (gen.DeleteNewsResponseObject, error) {
	return s.News.DeleteNews(ctx, req)
}

// ─── Polls delegations ────────────────────────────────────────────────────────

func (s *Server) ListPolls(ctx context.Context, req gen.ListPollsRequestObject) (gen.ListPollsResponseObject, error) {
	return s.Polls.ListPolls(ctx, req)
}

func (s *Server) CreatePoll(ctx context.Context, req gen.CreatePollRequestObject) (gen.CreatePollResponseObject, error) {
	return s.Polls.CreatePoll(ctx, req)
}

func (s *Server) VotePoll(ctx context.Context, req gen.VotePollRequestObject) (gen.VotePollResponseObject, error) {
	return s.Polls.VotePoll(ctx, req)
}

func (s *Server) DeletePoll(ctx context.Context, req gen.DeletePollRequestObject) (gen.DeletePollResponseObject, error) {
	return s.Polls.DeletePoll(ctx, req)
}

// ─── Notifications delegations ────────────────────────────────────────────────

func (s *Server) ListNotifications(ctx context.Context, req gen.ListNotificationsRequestObject) (gen.ListNotificationsResponseObject, error) {
	return s.Notifications.ListNotifications(ctx, req)
}

func (s *Server) MarkNotificationsSeen(ctx context.Context, req gen.MarkNotificationsSeenRequestObject) (gen.MarkNotificationsSeenResponseObject, error) {
	return s.Notifications.MarkNotificationsSeen(ctx, req)
}

// ─── Finances delegations ─────────────────────────────────────────────────────

func (s *Server) GetFinanceOverview(ctx context.Context, req gen.GetFinanceOverviewRequestObject) (gen.GetFinanceOverviewResponseObject, error) {
	return s.Finances.GetFinanceOverview(ctx, req)
}

func (s *Server) CreateTransaction(ctx context.Context, req gen.CreateTransactionRequestObject) (gen.CreateTransactionResponseObject, error) {
	return s.Finances.CreateTransaction(ctx, req)
}

func (s *Server) UpdateTransaction(ctx context.Context, req gen.UpdateTransactionRequestObject) (gen.UpdateTransactionResponseObject, error) {
	return s.Finances.UpdateTransaction(ctx, req)
}

func (s *Server) DeleteTransaction(ctx context.Context, req gen.DeleteTransactionRequestObject) (gen.DeleteTransactionResponseObject, error) {
	return s.Finances.DeleteTransaction(ctx, req)
}

func (s *Server) CreatePenalty(ctx context.Context, req gen.CreatePenaltyRequestObject) (gen.CreatePenaltyResponseObject, error) {
	return s.Finances.CreatePenalty(ctx, req)
}

func (s *Server) UpdatePenalty(ctx context.Context, req gen.UpdatePenaltyRequestObject) (gen.UpdatePenaltyResponseObject, error) {
	return s.Finances.UpdatePenalty(ctx, req)
}

func (s *Server) DeletePenalty(ctx context.Context, req gen.DeletePenaltyRequestObject) (gen.DeletePenaltyResponseObject, error) {
	return s.Finances.DeletePenalty(ctx, req)
}

func (s *Server) CreatePenaltyAssignment(ctx context.Context, req gen.CreatePenaltyAssignmentRequestObject) (gen.CreatePenaltyAssignmentResponseObject, error) {
	return s.Finances.CreatePenaltyAssignment(ctx, req)
}

func (s *Server) DeletePenaltyAssignment(ctx context.Context, req gen.DeletePenaltyAssignmentRequestObject) (gen.DeletePenaltyAssignmentResponseObject, error) {
	return s.Finances.DeletePenaltyAssignment(ctx, req)
}

func (s *Server) TogglePenaltyPaid(ctx context.Context, req gen.TogglePenaltyPaidRequestObject) (gen.TogglePenaltyPaidResponseObject, error) {
	return s.Finances.TogglePenaltyPaid(ctx, req)
}

func (s *Server) UpdateContribution(ctx context.Context, req gen.UpdateContributionRequestObject) (gen.UpdateContributionResponseObject, error) {
	return s.Finances.UpdateContribution(ctx, req)
}

func (s *Server) ToggleContribution(ctx context.Context, req gen.ToggleContributionRequestObject) (gen.ToggleContributionResponseObject, error) {
	return s.Finances.ToggleContribution(ctx, req)
}

// ─── Stats delegations ────────────────────────────────────────────────────────

func (s *Server) GetStatsOverview(ctx context.Context, req gen.GetStatsOverviewRequestObject) (gen.GetStatsOverviewResponseObject, error) {
	return s.Stats.GetStatsOverview(ctx, req)
}

func (s *Server) GetMemberStats(ctx context.Context, req gen.GetMemberStatsRequestObject) (gen.GetMemberStatsResponseObject, error) {
	return s.Stats.GetMemberStats(ctx, req)
}
