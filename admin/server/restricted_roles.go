package server

import (
	"context"
	"errors"
	"github.com/rilldata/rill/admin/database"
	"github.com/rilldata/rill/admin/email"
	"github.com/rilldata/rill/admin/server/auth"
	adminv1 "github.com/rilldata/rill/proto/gen/rill/admin/v1"
	"github.com/rilldata/rill/runtime/pkg/observability"
	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) CreateProjectRestrictedRole(context.Context, *adminv1.CreateProjectRestrictedRoleRequest) (*adminv1.CreateProjectRestrictedRoleResponse, error) {
	panic("not implemented")
}

func (s *Server) AddRestrictedProjectMember(ctx context.Context, req *adminv1.AddRestrictedProjectMemberRequest) (*adminv1.AddProjectMemberResponse, error) {
	observability.AddRequestAttributes(ctx,
		attribute.String("args.org", req.Organization),
		attribute.String("args.project", req.Project),
		attribute.StringSlice("args.role", req.Roles),
	)

	proj, err := s.admin.DB.FindProjectByName(ctx, req.Organization, req.Project)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	claims := auth.GetClaims(ctx)
	if !claims.ProjectPermissions(ctx, proj.OrganizationID, proj.ID).ManageProjectMembers {
		return nil, status.Error(codes.PermissionDenied, "not allowed to add project members")
	}

	// Check outstanding invites quota
	count, err := s.admin.DB.CountInvitesForOrganization(ctx, proj.OrganizationID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	org, err := s.admin.DB.FindOrganization(ctx, proj.OrganizationID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if org.QuotaOutstandingInvites >= 0 && count >= org.QuotaOutstandingInvites {
		return nil, status.Errorf(codes.FailedPrecondition, "quota exceeded: org %q can at most have %d outstanding invitations", org.Name, org.QuotaOutstandingInvites)
	}

	roles := make([]*database.RestrictedProjectRole, 0, len(req.Roles))
	roleIDs := make([]string, 0, len(req.Roles))
	for _, roleName := range req.Roles {
		role, err := s.admin.DB.FindRestrictedProjectRole(ctx, roleName, proj.ID)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		roles = append(roles, role)
		roleIDs = append(roleIDs, role.ID)
	}

	var invitedByUserID, invitedByName string
	if claims.OwnerType() == auth.OwnerTypeUser {
		user, err := s.admin.DB.FindUser(ctx, claims.OwnerID())
		if err != nil && !errors.Is(err, database.ErrNotFound) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		invitedByUserID = user.ID
		invitedByName = user.DisplayName
	}

	user, err := s.admin.DB.FindUserByEmail(ctx, req.Email)
	if err != nil {
		if !errors.Is(err, database.ErrNotFound) {
			return nil, status.Error(codes.Internal, err.Error())
		}

		// Invite user to join the project
		err := s.admin.DB.InsertProjectInvite(ctx, &database.InsertProjectInviteOptions{
			Email:             req.Email,
			InviterID:         invitedByUserID,
			ProjectID:         proj.ID,
			RoleID:            "",
			RestrictedRoleIDs: roleIDs,
		})
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		// Send invitation email
		err = s.admin.Email.SendProjectInvite(&email.ProjectInvite{
			ToEmail:       req.Email,
			ToName:        "",
			OrgName:       org.Name,
			ProjectName:   proj.Name,
			RoleName:      "restricted",
			InvitedByName: invitedByName,
		})
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		return &adminv1.AddProjectMemberResponse{
			PendingSignup: true,
		}, nil
	}

	for _, role := range roles {
		err = s.admin.DB.InsertProjectMemberUser(ctx, proj.ID, user.ID, role.ID)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
	}

	err = s.admin.Email.SendProjectAddition(&email.ProjectAddition{
		ToEmail:       req.Email,
		ToName:        "",
		OrgName:       org.Name,
		ProjectName:   proj.Name,
		RoleName:      "restricted",
		InvitedByName: invitedByName,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &adminv1.AddProjectMemberResponse{
		PendingSignup: false,
	}, nil
}
