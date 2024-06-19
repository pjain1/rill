package server

import (
	"context"
	"errors"
	"math"
	"strconv"
	"strings"

	"github.com/rilldata/rill/admin/billing"
	"github.com/rilldata/rill/admin/database"
	"github.com/rilldata/rill/admin/pkg/publicemail"
	"github.com/rilldata/rill/admin/server/auth"
	adminv1 "github.com/rilldata/rill/proto/gen/rill/admin/v1"
	"github.com/rilldata/rill/runtime/pkg/email"
	"github.com/rilldata/rill/runtime/pkg/observability"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *Server) ListOrganizations(ctx context.Context, req *adminv1.ListOrganizationsRequest) (*adminv1.ListOrganizationsResponse, error) {
	// Check the request is made by an authenticated user
	claims := auth.GetClaims(ctx)
	if claims.OwnerType() != auth.OwnerTypeUser {
		return nil, status.Error(codes.Unauthenticated, "not authenticated as a user")
	}

	token, err := unmarshalPageToken(req.PageToken)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	pageSize := validPageSize(req.PageSize)

	orgs, err := s.admin.DB.FindOrganizationsForUser(ctx, claims.OwnerID(), token.Val, pageSize)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	nextToken := ""
	if len(orgs) >= pageSize {
		nextToken = marshalPageToken(orgs[len(orgs)-1].Name)
	}

	pbs := make([]*adminv1.Organization, len(orgs))
	for i, org := range orgs {
		pbs[i] = organizationToDTO(org)
	}

	return &adminv1.ListOrganizationsResponse{Organizations: pbs, NextPageToken: nextToken}, nil
}

func (s *Server) GetOrganization(ctx context.Context, req *adminv1.GetOrganizationRequest) (*adminv1.GetOrganizationResponse, error) {
	observability.AddRequestAttributes(ctx, attribute.String("args.org", req.Name))

	org, err := s.admin.DB.FindOrganizationByName(ctx, req.Name)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "org not found")
		}
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	claims := auth.GetClaims(ctx)
	if !claims.OrganizationPermissions(ctx, org.ID).ReadOrg && !claims.Superuser(ctx) {
		// check if the org has any public projects, this works for anonymous users as well
		hasPublicProject, err := s.admin.DB.CheckOrganizationHasPublicProjects(ctx, org.ID)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		// these are the permissions for public and for outside members
		publicPermissions := &adminv1.OrganizationPermissions{ReadOrg: true, ReadProjects: true}
		if hasPublicProject {
			return &adminv1.GetOrganizationResponse{
				Organization: organizationToDTO(org),
				Permissions:  publicPermissions,
			}, nil
		}
		// check if the user is outside members of a project in the org
		if claims.OwnerType() == auth.OwnerTypeUser {
			exists, err := s.admin.DB.CheckOrganizationHasOutsideUser(ctx, org.ID, claims.OwnerID())
			if err != nil {
				return nil, status.Error(codes.Internal, err.Error())
			}
			if exists {
				return &adminv1.GetOrganizationResponse{
					Organization: organizationToDTO(org),
					Permissions:  publicPermissions,
				}, nil
			}
		}
		return nil, status.Error(codes.PermissionDenied, "not allowed to read org")
	}

	return &adminv1.GetOrganizationResponse{
		Organization: organizationToDTO(org),
		Permissions:  claims.OrganizationPermissions(ctx, org.ID),
	}, nil
}

func (s *Server) CreateOrganization(ctx context.Context, req *adminv1.CreateOrganizationRequest) (*adminv1.CreateOrganizationResponse, error) {
	observability.AddRequestAttributes(ctx,
		attribute.String("args.org", req.Name),
		attribute.String("args.description", req.Description),
	)

	// Check the request is made by an authenticated user
	claims := auth.GetClaims(ctx)
	if claims.OwnerType() != auth.OwnerTypeUser {
		return nil, status.Error(codes.Unauthenticated, "not authenticated as a user")
	}

	// check single user org limit for this user
	user, err := s.admin.DB.FindUser(ctx, claims.OwnerID())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	count, err := s.admin.DB.CountSingleuserOrganizationsForMemberUser(ctx, user.ID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if user.QuotaSingleuserOrgs >= 0 && count >= user.QuotaSingleuserOrgs {
		return nil, status.Errorf(codes.FailedPrecondition, "quota exceeded: you can only create %d single-user orgs", user.QuotaSingleuserOrgs)
	}

	org, err := s.admin.CreateOrganizationForUser(ctx, user.ID, req.Name, req.Description)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	return &adminv1.CreateOrganizationResponse{
		Organization: organizationToDTO(org),
	}, nil
}

func (s *Server) DeleteOrganization(ctx context.Context, req *adminv1.DeleteOrganizationRequest) (*adminv1.DeleteOrganizationResponse, error) {
	observability.AddRequestAttributes(ctx, attribute.String("args.org", req.Name))

	org, err := s.admin.DB.FindOrganizationByName(ctx, req.Name)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	claims := auth.GetClaims(ctx)
	if !claims.OrganizationPermissions(ctx, org.ID).ManageOrg {
		return nil, status.Error(codes.PermissionDenied, "not allowed to delete org")
	}

	err = s.admin.DB.DeleteOrganization(ctx, req.Name)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// cancel subscription
	if org.BillingCustomerID != "" {
		err = s.admin.Biller.CancelSubscriptionsForCustomer(ctx, org.BillingCustomerID, billing.SubscriptionCancellationOptionImmediate)
		if err != nil {
			s.logger.Error("failed to cancel subscriptions", zap.String("org", org.Name), zap.Error(err))
		}
	}

	return &adminv1.DeleteOrganizationResponse{}, nil
}

func (s *Server) UpdateOrganization(ctx context.Context, req *adminv1.UpdateOrganizationRequest) (*adminv1.UpdateOrganizationResponse, error) {
	observability.AddRequestAttributes(ctx, attribute.String("args.org", req.Name))
	if req.Description != nil {
		observability.AddRequestAttributes(ctx, attribute.String("args.description", *req.Description))
	}
	if req.NewName != nil {
		observability.AddRequestAttributes(ctx, attribute.String("args.new_name", *req.NewName))
	}

	org, err := s.admin.DB.FindOrganizationByName(ctx, req.Name)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	claims := auth.GetClaims(ctx)
	if !claims.OrganizationPermissions(ctx, org.ID).ManageOrg {
		return nil, status.Error(codes.PermissionDenied, "not allowed to update org")
	}

	nameChanged := req.NewName != nil && *req.NewName != org.Name
	org, err = s.admin.DB.UpdateOrganization(ctx, org.ID, &database.UpdateOrganizationOptions{
		Name:                                valOrDefault(req.NewName, org.Name),
		Description:                         valOrDefault(req.Description, org.Description),
		QuotaProjects:                       org.QuotaProjects,
		QuotaDeployments:                    org.QuotaDeployments,
		QuotaSlotsTotal:                     org.QuotaSlotsTotal,
		QuotaSlotsPerDeployment:             org.QuotaSlotsPerDeployment,
		QuotaOutstandingInvites:             org.QuotaOutstandingInvites,
		QuotaStorageLimitBytesPerDeployment: org.QuotaStorageLimitBytesPerDeployment,
		BillingCustomerID:                   org.BillingCustomerID,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if nameChanged {
		err := s.admin.UpdateOrgDeploymentAnnotations(ctx, org)
		if err != nil {
			return nil, err
		}
	}

	return &adminv1.UpdateOrganizationResponse{
		Organization: organizationToDTO(org),
	}, nil
}

func (s *Server) UpdateOrganizationBillingPlan(ctx context.Context, req *adminv1.UpdateOrganizationBillingPlanRequest) (*adminv1.UpdateOrganizationBillingPlanResponse, error) {
	observability.AddRequestAttributes(ctx, attribute.String("args.org", req.OrgName))
	if req.RillPlanId != nil {
		observability.AddRequestAttributes(ctx, attribute.String("args.rill_plan_id", *req.RillPlanId))
	}
	if req.BillerPlanId != nil {
		observability.AddRequestAttributes(ctx, attribute.String("args.biller_plan_id", *req.BillerPlanId))
	}

	org, err := s.admin.DB.FindOrganizationByName(ctx, req.OrgName)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	claims := auth.GetClaims(ctx)
	if !claims.OrganizationPermissions(ctx, org.ID).ManageOrg && !claims.Superuser(ctx) {
		return nil, status.Error(codes.PermissionDenied, "not allowed to update org billing plan")
	}

	if req.RillPlanId == nil && req.BillerPlanId == nil {
		return nil, status.Error(codes.InvalidArgument, "rill or biller plan id must be provided")
	}

	plan, err := s.admin.Biller.GetPlan(ctx, valOrDefault(req.RillPlanId, ""), valOrDefault(req.BillerPlanId, ""))
	if err != nil {
		if errors.Is(err, billing.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "plan not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	if org.BillingCustomerID == "" {
		// create new customer
		customerID, err := s.admin.Biller.CreateCustomer(ctx, org)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		org.BillingCustomerID = customerID

		// create new subscription
		_, err = s.admin.Biller.CreateSubscription(ctx, org.BillingCustomerID, plan)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
	} else {
		subs, err := s.admin.Biller.GetSubscriptionsForCustomer(ctx, org.BillingCustomerID)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		if len(subs) > 0 {
			for _, sub := range subs {
				if sub.Plan.BillerID == plan.BillerID {
					return nil, status.Error(codes.InvalidArgument, "same plan already assigned to the organization")
				}
			}

			if len(subs) == 1 {
				// schedule plan change
				_, err = s.admin.Biller.ChangeSubscriptionPlan(ctx, subs[0].ID, plan)
				if err != nil {
					return nil, status.Error(codes.Internal, err.Error())
				}
			} else {
				// multiple subscriptions, cancel them first immediately and assign new plan
				// should not happen unless externally assigned multiple subscriptions to the same org in the billing system
				for _, sub := range subs {
					err = s.admin.Biller.CancelSubscription(ctx, sub.ID, billing.SubscriptionCancellationOptionImmediate)
					if err != nil {
						return nil, status.Error(codes.Internal, err.Error())
					}
				}

				// create new subscription
				_, err = s.admin.Biller.CreateSubscription(ctx, org.BillingCustomerID, plan)
				if err != nil {
					return nil, status.Error(codes.Internal, err.Error())
				}
			}
		} else {
			// create new subscription
			_, err = s.admin.Biller.CreateSubscription(ctx, org.BillingCustomerID, plan)
			if err != nil {
				return nil, status.Error(codes.Internal, err.Error())
			}
		}
	}

	quotaProjects := valOrDefault(plan.Quotas.NumProjects, org.QuotaProjects)
	quotaDeployments := valOrDefault(plan.Quotas.NumDeployments, org.QuotaDeployments)
	quotaSlotsTotal := valOrDefault(plan.Quotas.NumSlotsTotal, org.QuotaSlotsTotal)
	quotaSlotsPerDeployment := valOrDefault(plan.Quotas.NumSlotsPerDeployment, org.QuotaSlotsPerDeployment)
	quotaOutstandingInvites := valOrDefault(plan.Quotas.NumOutstandingInvites, org.QuotaOutstandingInvites)
	quotaStorageLimitBytesPerDeployment := valOrDefault(plan.Quotas.StorageLimitBytesPerDeployment, org.QuotaStorageLimitBytesPerDeployment)

	org, err = s.admin.DB.UpdateOrganization(ctx, org.ID, &database.UpdateOrganizationOptions{
		Name:                                org.Name,
		Description:                         org.Description,
		QuotaProjects:                       quotaProjects,
		QuotaDeployments:                    quotaDeployments,
		QuotaSlotsTotal:                     quotaSlotsTotal,
		QuotaSlotsPerDeployment:             quotaSlotsPerDeployment,
		QuotaOutstandingInvites:             quotaOutstandingInvites,
		QuotaStorageLimitBytesPerDeployment: quotaStorageLimitBytesPerDeployment,
		BillingCustomerID:                   org.BillingCustomerID,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	subs, err := s.admin.Biller.GetSubscriptionsForCustomer(ctx, org.BillingCustomerID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	var subscriptions []*adminv1.Subscription
	for _, sub := range subs {
		subscriptions = append(subscriptions, subscriptionToDTO(sub))
	}

	return &adminv1.UpdateOrganizationBillingPlanResponse{
		Organization:  organizationToDTO(org),
		Subscriptions: subscriptions,
	}, nil
}

func (s *Server) ListOrganizationSubscriptions(ctx context.Context, req *adminv1.ListOrganizationSubscriptionsRequest) (*adminv1.ListOrganizationSubscriptionsResponse, error) {
	observability.AddRequestAttributes(ctx, attribute.String("args.org", req.OrgName))

	org, err := s.admin.DB.FindOrganizationByName(ctx, req.OrgName)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	claims := auth.GetClaims(ctx)
	if !claims.OrganizationPermissions(ctx, org.ID).ManageOrg && !claims.Superuser(ctx) {
		return nil, status.Error(codes.PermissionDenied, "not allowed to read org subscriptions")
	}

	if org.BillingCustomerID == "" {
		return &adminv1.ListOrganizationSubscriptionsResponse{Organization: organizationToDTO(org)}, nil
	}

	subs, err := s.admin.Biller.GetSubscriptionsForCustomer(ctx, org.BillingCustomerID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	var subscriptions []*adminv1.Subscription
	for _, sub := range subs {
		subscriptions = append(subscriptions, subscriptionToDTO(sub))
	}

	return &adminv1.ListOrganizationSubscriptionsResponse{
		Organization:  organizationToDTO(org),
		Subscriptions: subscriptions,
	}, nil
}

func (s *Server) DeleteOrganizationSubscription(ctx context.Context, req *adminv1.DeleteOrganizationSubscriptionRequest) (*adminv1.DeleteOrganizationSubscriptionResponse, error) {
	observability.AddRequestAttributes(ctx, attribute.String("args.org", req.OrgName), attribute.String("args.subscription_id", req.SubscriptionId))

	org, err := s.admin.DB.FindOrganizationByName(ctx, req.OrgName)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	claims := auth.GetClaims(ctx)
	if !claims.OrganizationPermissions(ctx, org.ID).ManageOrg && !claims.Superuser(ctx) {
		return nil, status.Error(codes.PermissionDenied, "not allowed to delete org subscription")
	}

	if org.BillingCustomerID == "" {
		return nil, status.Error(codes.FailedPrecondition, "organization has no billing customer")
	}

	subs, err := s.admin.Biller.GetSubscriptionsForCustomer(ctx, org.BillingCustomerID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	for _, sub := range subs {
		if sub.ID == req.SubscriptionId {
			cancelOption := billing.SubscriptionCancellationOptionEndOfSubscriptionTerm
			if req.SubscriptionCancelEffective == adminv1.SubscriptionCancelEffective_SUBSCRIPTION_CANCEL_EFFECTIVE_NOW {
				cancelOption = billing.SubscriptionCancellationOptionImmediate
			}
			err = s.admin.Biller.CancelSubscription(ctx, sub.ID, cancelOption)
			if err != nil {
				return nil, status.Error(codes.Internal, err.Error())
			}
			return &adminv1.DeleteOrganizationSubscriptionResponse{}, nil
		}
	}

	return nil, status.Error(codes.NotFound, "subscription not found")
}

func (s *Server) ListOrganizationMembers(ctx context.Context, req *adminv1.ListOrganizationMembersRequest) (*adminv1.ListOrganizationMembersResponse, error) {
	observability.AddRequestAttributes(ctx,
		attribute.String("args.org", req.Organization),
	)

	org, err := s.admin.DB.FindOrganizationByName(ctx, req.Organization)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	claims := auth.GetClaims(ctx)
	if !claims.Superuser(ctx) && !claims.OrganizationPermissions(ctx, org.ID).ReadOrgMembers {
		return nil, status.Error(codes.PermissionDenied, "not authorized to read org members")
	}

	token, err := unmarshalPageToken(req.PageToken)
	if err != nil {
		return nil, err
	}
	pageSize := validPageSize(req.PageSize)

	members, err := s.admin.DB.FindOrganizationMemberUsers(ctx, org.ID, token.Val, pageSize)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	nextToken := ""
	if len(members) >= pageSize {
		nextToken = marshalPageToken(members[len(members)-1].Email)
	}

	dtos := make([]*adminv1.Member, len(members))
	for i, user := range members {
		dtos[i] = memberToPB(user)
	}

	return &adminv1.ListOrganizationMembersResponse{
		Members:       dtos,
		NextPageToken: nextToken,
	}, nil
}

func (s *Server) ListOrganizationInvites(ctx context.Context, req *adminv1.ListOrganizationInvitesRequest) (*adminv1.ListOrganizationInvitesResponse, error) {
	observability.AddRequestAttributes(ctx,
		attribute.String("args.org", req.Organization),
	)

	org, err := s.admin.DB.FindOrganizationByName(ctx, req.Organization)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	claims := auth.GetClaims(ctx)
	if !claims.OrganizationPermissions(ctx, org.ID).ReadOrgMembers {
		return nil, status.Error(codes.PermissionDenied, "not authorized to read org members")
	}

	token, err := unmarshalPageToken(req.PageToken)
	if err != nil {
		return nil, err
	}
	pageSize := validPageSize(req.PageSize)

	// get pending user invites for this org
	userInvites, err := s.admin.DB.FindOrganizationInvites(ctx, org.ID, token.Val, pageSize)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	nextToken := ""
	if len(userInvites) >= pageSize {
		nextToken = marshalPageToken(userInvites[len(userInvites)-1].Email)
	}

	invitesDtos := make([]*adminv1.UserInvite, len(userInvites))
	for i, invite := range userInvites {
		invitesDtos[i] = inviteToPB(invite)
	}

	return &adminv1.ListOrganizationInvitesResponse{
		Invites:       invitesDtos,
		NextPageToken: nextToken,
	}, nil
}

func (s *Server) AddOrganizationMember(ctx context.Context, req *adminv1.AddOrganizationMemberRequest) (*adminv1.AddOrganizationMemberResponse, error) {
	observability.AddRequestAttributes(ctx,
		attribute.String("args.org", req.Organization),
		attribute.String("args.role", req.Role),
	)

	org, err := s.admin.DB.FindOrganizationByName(ctx, req.Organization)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	claims := auth.GetClaims(ctx)
	if !claims.OrganizationPermissions(ctx, org.ID).ManageOrgMembers {
		return nil, status.Error(codes.PermissionDenied, "not allowed to add org members")
	}

	count, err := s.admin.DB.CountInvitesForOrganization(ctx, org.ID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if org.QuotaOutstandingInvites >= 0 && count >= org.QuotaOutstandingInvites {
		return nil, status.Errorf(codes.FailedPrecondition, "quota exceeded: org can at most have %d outstanding invitations", org.QuotaOutstandingInvites)
	}

	role, err := s.admin.DB.FindOrganizationRole(ctx, req.Role)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
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

		// Invite user to join org
		err := s.admin.DB.InsertOrganizationInvite(ctx, &database.InsertOrganizationInviteOptions{
			Email:     req.Email,
			InviterID: invitedByUserID,
			OrgID:     org.ID,
			RoleID:    role.ID,
		})
		if err != nil {
			if errors.Is(err, database.ErrNotUnique) {
				return nil, status.Error(codes.AlreadyExists, err.Error())
			}
			return nil, status.Error(codes.Internal, err.Error())
		}

		// Send invitation email
		err = s.admin.Email.SendOrganizationInvite(&email.OrganizationInvite{
			ToEmail:       req.Email,
			ToName:        "",
			AdminURL:      s.opts.ExternalURL,
			FrontendURL:   s.opts.FrontendURL,
			OrgName:       org.Name,
			RoleName:      role.Name,
			InvitedByName: invitedByName,
		})
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		return &adminv1.AddOrganizationMemberResponse{
			PendingSignup: true,
		}, nil
	}

	ctx, tx, err := s.admin.DB.NewTx(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	defer func() { _ = tx.Rollback() }()

	err = s.admin.DB.InsertOrganizationMemberUser(ctx, org.ID, user.ID, role.ID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	err = s.admin.DB.InsertUsergroupMember(ctx, *org.AllUsergroupID, user.ID)
	if err != nil {
		if !errors.Is(err, database.ErrNotUnique) {
			return nil, status.Error(codes.Internal, err.Error())
		}
		// If the user is already in the all user group, we can ignore the error
	}

	err = tx.Commit()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	err = s.admin.Email.SendOrganizationAddition(&email.OrganizationAddition{
		ToEmail:       req.Email,
		ToName:        "",
		FrontendURL:   s.opts.FrontendURL,
		OrgName:       org.Name,
		RoleName:      role.Name,
		InvitedByName: invitedByName,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &adminv1.AddOrganizationMemberResponse{
		PendingSignup: false,
	}, nil
}

func (s *Server) RemoveOrganizationMember(ctx context.Context, req *adminv1.RemoveOrganizationMemberRequest) (*adminv1.RemoveOrganizationMemberResponse, error) {
	observability.AddRequestAttributes(ctx,
		attribute.String("args.org", req.Organization),
		attribute.Bool("args.keep_project_roles", req.KeepProjectRoles),
	)

	org, err := s.admin.DB.FindOrganizationByName(ctx, req.Organization)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	user, err := s.admin.DB.FindUserByEmail(ctx, req.Email)
	if err != nil {
		if !errors.Is(err, database.ErrNotFound) {
			return nil, status.Error(codes.Internal, err.Error())
		}

		// Only admins can remove pending invites.
		// NOTE: If we change invites to accept/decline (instead of auto-accept on signup), we need to revisit this.
		claims := auth.GetClaims(ctx)
		if !claims.OrganizationPermissions(ctx, org.ID).ManageOrgMembers {
			return nil, status.Error(codes.PermissionDenied, "not allowed to remove org members")
		}

		// Check if there is a pending invite
		invite, err := s.admin.DB.FindOrganizationInvite(ctx, org.ID, req.Email)
		if err != nil {
			if errors.Is(err, database.ErrNotFound) {
				return nil, status.Error(codes.InvalidArgument, "user not found")
			}
			return nil, status.Error(codes.Internal, err.Error())
		}

		err = s.admin.DB.DeleteOrganizationInvite(ctx, invite.ID)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		return &adminv1.RemoveOrganizationMemberResponse{}, nil
	}

	// The caller must either have ManageOrgMembers permission or be the user being removed.
	claims := auth.GetClaims(ctx)
	isManager := claims.OrganizationPermissions(ctx, org.ID).ManageOrgMembers
	isSelf := claims.OwnerType() == auth.OwnerTypeUser && claims.OwnerID() == user.ID
	if !isManager && !isSelf {
		return nil, status.Error(codes.PermissionDenied, "not allowed to remove org members")
	}

	// Check that the user is not the last admin
	role, err := s.admin.DB.FindOrganizationRole(ctx, database.OrganizationRoleNameAdmin)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	users, err := s.admin.DB.FindOrganizationMemberUsersByRole(ctx, org.ID, role.ID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if len(users) == 1 && users[0].ID == user.ID {
		return nil, status.Error(codes.InvalidArgument, "cannot remove the last admin member")
	}

	ctx, tx, err := s.admin.DB.NewTx(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	defer func() { _ = tx.Rollback() }()

	err = s.admin.DB.DeleteOrganizationMemberUser(ctx, org.ID, user.ID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// delete from all user group
	err = s.admin.DB.DeleteUsergroupMember(ctx, *org.AllUsergroupID, user.ID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// delete from projects if KeepProjectRoles flag is set
	if !req.KeepProjectRoles {
		err = s.admin.DB.DeleteAllProjectMemberUserForOrganization(ctx, org.ID, user.ID)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	err = tx.Commit()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &adminv1.RemoveOrganizationMemberResponse{}, nil
}

func (s *Server) SetOrganizationMemberRole(ctx context.Context, req *adminv1.SetOrganizationMemberRoleRequest) (*adminv1.SetOrganizationMemberRoleResponse, error) {
	observability.AddRequestAttributes(ctx,
		attribute.String("args.org", req.Organization),
		attribute.String("args.role", req.Role),
	)

	org, err := s.admin.DB.FindOrganizationByName(ctx, req.Organization)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	claims := auth.GetClaims(ctx)
	if !claims.OrganizationPermissions(ctx, org.ID).ManageOrgMembers {
		return nil, status.Error(codes.PermissionDenied, "not allowed to set org members role")
	}

	role, err := s.admin.DB.FindOrganizationRole(ctx, req.Role)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	user, err := s.admin.DB.FindUserByEmail(ctx, req.Email)
	if err != nil {
		if !errors.Is(err, database.ErrNotFound) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		// Check if there is a pending invite for this user
		invite, err := s.admin.DB.FindOrganizationInvite(ctx, org.ID, req.Email)
		if err != nil {
			if errors.Is(err, database.ErrNotFound) {
				return nil, status.Error(codes.InvalidArgument, "user not found")
			}
			return nil, status.Error(codes.Internal, err.Error())
		}
		err = s.admin.DB.UpdateOrganizationInviteRole(ctx, invite.ID, role.ID)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		return &adminv1.SetOrganizationMemberRoleResponse{}, nil
	}

	// Check if the user is the last owner
	if role.Name != database.OrganizationRoleNameAdmin {
		adminRole, err := s.admin.DB.FindOrganizationRole(ctx, database.OrganizationRoleNameAdmin)
		if err != nil {
			panic(err)
		}
		// TODO optimize this, may be extract roles during auth token validation
		//  and store as part of the claims and fetch admins only if the user is an admin
		users, err := s.admin.DB.FindOrganizationMemberUsersByRole(ctx, org.ID, adminRole.ID)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		if len(users) == 1 && users[0].ID == user.ID {
			return nil, status.Error(codes.InvalidArgument, "cannot change role of the last owner")
		}
	}

	err = s.admin.DB.UpdateOrganizationMemberUserRole(ctx, org.ID, user.ID, role.ID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	return &adminv1.SetOrganizationMemberRoleResponse{}, nil
}

func (s *Server) LeaveOrganization(ctx context.Context, req *adminv1.LeaveOrganizationRequest) (*adminv1.LeaveOrganizationResponse, error) {
	observability.AddRequestAttributes(ctx,
		attribute.String("args.org", req.Organization),
	)

	// Check the request is made by an authenticated user
	claims := auth.GetClaims(ctx)
	if claims.OwnerType() != auth.OwnerTypeUser {
		return nil, status.Error(codes.Unauthenticated, "not authenticated as a user")
	}

	org, err := s.admin.DB.FindOrganizationByName(ctx, req.Organization)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if !claims.OrganizationPermissions(ctx, org.ID).ManageOrgMembers {
		return nil, status.Error(codes.PermissionDenied, "not allowed to remove org members")
	}

	role, err := s.admin.DB.FindOrganizationRole(ctx, database.OrganizationRoleNameAdmin)
	if err != nil {
		panic(err)
	}

	// check if the user is the last owner
	// TODO optimize this, may be extract roles during auth token validation
	//  and store as part of the claims and fetch admins only if the user is an admin
	users, err := s.admin.DB.FindOrganizationMemberUsersByRole(ctx, org.ID, role.ID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if len(users) == 1 && users[0].ID == claims.OwnerID() {
		return nil, status.Error(codes.InvalidArgument, "cannot remove the last owner")
	}

	ctx, tx, err := s.admin.DB.NewTx(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	defer func() { _ = tx.Rollback() }()
	err = s.admin.DB.DeleteOrganizationMemberUser(ctx, org.ID, claims.OwnerID())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// delete from all user group
	err = s.admin.DB.DeleteUsergroupMember(ctx, *org.AllUsergroupID, claims.OwnerID())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	err = tx.Commit()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &adminv1.LeaveOrganizationResponse{}, nil
}

func (s *Server) CreateWhitelistedDomain(ctx context.Context, req *adminv1.CreateWhitelistedDomainRequest) (*adminv1.CreateWhitelistedDomainResponse, error) {
	observability.AddRequestAttributes(ctx,
		attribute.String("args.org", req.Organization),
		attribute.String("args.domain", req.Domain),
		attribute.String("args.role", req.Role),
	)

	claims := auth.GetClaims(ctx)
	if claims.OwnerType() != auth.OwnerTypeUser {
		return nil, status.Error(codes.Unauthenticated, "not authenticated as a user")
	}

	org, err := s.admin.DB.FindOrganizationByName(ctx, req.Organization)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "org not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	if !claims.Superuser(ctx) {
		if !claims.OrganizationPermissions(ctx, org.ID).ManageOrg {
			return nil, status.Error(codes.PermissionDenied, "only org admins can add whitelisted domain")
		}
		// check if the user's domain matches the whitelist domain
		user, err := s.admin.DB.FindUser(ctx, claims.OwnerID())
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		if !strings.HasSuffix(user.Email, "@"+req.Domain) {
			return nil, status.Error(codes.PermissionDenied, "Domain name doesn’t match verified email domain. Please contact Rill support.")
		}

		if publicemail.IsPublic(req.Domain) {
			return nil, status.Errorf(codes.InvalidArgument, "Public Domain %s cannot be whitelisted", req.Domain)
		}
	}

	role, err := s.admin.DB.FindOrganizationRole(ctx, req.Role)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "role not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	// find existing users belonging to the whitelisted domain to the org
	users, err := s.admin.DB.FindUsersByEmailPattern(ctx, "%@"+req.Domain, "", math.MaxInt)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// filter out users who are already members of the org
	newUsers := make([]*database.User, 0)
	for _, user := range users {
		// check if user is already a member of the org
		exists, err := s.admin.DB.CheckUserIsAnOrganizationMember(ctx, user.ID, org.ID)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		if !exists {
			newUsers = append(newUsers, user)
		}
	}

	ctx, tx, err := s.admin.DB.NewTx(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	_, err = s.admin.DB.InsertOrganizationWhitelistedDomain(ctx, &database.InsertOrganizationWhitelistedDomainOptions{
		OrgID:     org.ID,
		OrgRoleID: role.ID,
		Domain:    req.Domain,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	for _, user := range newUsers {
		err = s.admin.DB.InsertOrganizationMemberUser(ctx, org.ID, user.ID, role.ID)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		// add to all user group
		err = s.admin.DB.InsertUsergroupMember(ctx, *org.AllUsergroupID, user.ID)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return &adminv1.CreateWhitelistedDomainResponse{}, nil
}

func (s *Server) RemoveWhitelistedDomain(ctx context.Context, req *adminv1.RemoveWhitelistedDomainRequest) (*adminv1.RemoveWhitelistedDomainResponse, error) {
	observability.AddRequestAttributes(ctx,
		attribute.String("args.org", req.Organization),
		attribute.String("args.domain", req.Domain),
	)

	claims := auth.GetClaims(ctx)

	org, err := s.admin.DB.FindOrganizationByName(ctx, req.Organization)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "org not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	if !(claims.OrganizationPermissions(ctx, org.ID).ManageOrg || claims.Superuser(ctx)) {
		return nil, status.Error(codes.PermissionDenied, "only org admins can remove whitelisted domain")
	}

	invite, err := s.admin.DB.FindOrganizationWhitelistedDomain(ctx, org.ID, req.Domain)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "whitelist not found for org %q and domain %q", org.Name, req.Domain)
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	err = s.admin.DB.DeleteOrganizationWhitelistedDomain(ctx, invite.ID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &adminv1.RemoveWhitelistedDomainResponse{}, nil
}

func (s *Server) ListWhitelistedDomains(ctx context.Context, req *adminv1.ListWhitelistedDomainsRequest) (*adminv1.ListWhitelistedDomainsResponse, error) {
	observability.AddRequestAttributes(ctx,
		attribute.String("args.org", req.Organization),
	)

	claims := auth.GetClaims(ctx)

	org, err := s.admin.DB.FindOrganizationByName(ctx, req.Organization)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "org not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	if !(claims.OrganizationPermissions(ctx, org.ID).ManageOrg || claims.Superuser(ctx)) {
		return nil, status.Error(codes.PermissionDenied, "only org admins can list whitelisted domain")
	}

	whitelistedDomains, err := s.admin.DB.FindOrganizationWhitelistedDomainForOrganizationWithJoinedRoleNames(ctx, org.ID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	whitelistedDomainDtos := make([]*adminv1.WhitelistedDomain, len(whitelistedDomains))
	for i, whitelistedDomain := range whitelistedDomains {
		whitelistedDomainDtos[i] = whitelistedDomainToPB(whitelistedDomain)
	}

	return &adminv1.ListWhitelistedDomainsResponse{
		Domains: whitelistedDomainDtos,
	}, nil
}

func (s *Server) SudoUpdateOrganizationQuotas(ctx context.Context, req *adminv1.SudoUpdateOrganizationQuotasRequest) (*adminv1.SudoUpdateOrganizationQuotasResponse, error) {
	observability.AddRequestAttributes(ctx, attribute.String("args.org", req.OrgName))
	if req.Projects != nil {
		observability.AddRequestAttributes(ctx, attribute.Int("args.projects", int(*req.Projects)))
	}
	if req.Deployments != nil {
		observability.AddRequestAttributes(ctx, attribute.Int("args.deployments", int(*req.Deployments)))
	}
	if req.SlotsTotal != nil {
		observability.AddRequestAttributes(ctx, attribute.Int("args.slots_total", int(*req.SlotsTotal)))
	}
	if req.SlotsPerDeployment != nil {
		observability.AddRequestAttributes(ctx, attribute.Int("args.slots_per_deployment", int(*req.SlotsPerDeployment)))
	}
	if req.OutstandingInvites != nil {
		observability.AddRequestAttributes(ctx, attribute.Int("args.outstanding_invites", int(*req.OutstandingInvites)))
	}

	claims := auth.GetClaims(ctx)
	if !claims.Superuser(ctx) {
		return nil, status.Error(codes.PermissionDenied, "only superusers can manage quotas")
	}

	org, err := s.admin.DB.FindOrganizationByName(ctx, req.OrgName)
	if err != nil {
		return nil, err
	}

	opts := &database.UpdateOrganizationOptions{
		Name:                                req.OrgName,
		Description:                         org.Description,
		QuotaProjects:                       int(valOrDefault(req.Projects, uint32(org.QuotaProjects))),
		QuotaDeployments:                    int(valOrDefault(req.Deployments, uint32(org.QuotaDeployments))),
		QuotaSlotsTotal:                     int(valOrDefault(req.SlotsTotal, uint32(org.QuotaSlotsTotal))),
		QuotaSlotsPerDeployment:             int(valOrDefault(req.SlotsPerDeployment, uint32(org.QuotaSlotsPerDeployment))),
		QuotaOutstandingInvites:             int(valOrDefault(req.OutstandingInvites, uint32(org.QuotaOutstandingInvites))),
		QuotaStorageLimitBytesPerDeployment: int64(valOrDefault(req.StorageLimitBytesPerDeployment, uint64(org.QuotaStorageLimitBytesPerDeployment))),
	}

	updatedOrg, err := s.admin.DB.UpdateOrganization(ctx, org.ID, opts)
	if err != nil {
		return nil, err
	}

	return &adminv1.SudoUpdateOrganizationQuotasResponse{
		Organization: organizationToDTO(updatedOrg),
	}, nil
}

// SudoUpdateOrganizationBillingCustomer updates the billing customer id for an organization. May be useful if customer is initialized manually in billing system
func (s *Server) SudoUpdateOrganizationBillingCustomer(ctx context.Context, req *adminv1.SudoUpdateOrganizationBillingCustomerRequest) (*adminv1.SudoUpdateOrganizationBillingCustomerResponse, error) {
	observability.AddRequestAttributes(ctx, attribute.String("args.org", req.OrgName))
	observability.AddRequestAttributes(ctx, attribute.String("args.billing_customer_id", req.BillingCustomerId))

	claims := auth.GetClaims(ctx)
	if !claims.Superuser(ctx) {
		return nil, status.Error(codes.PermissionDenied, "only superusers can manage billing customer")
	}

	if req.BillingCustomerId == "" {
		return nil, status.Error(codes.InvalidArgument, "billing customer id is required")
	}

	org, err := s.admin.DB.FindOrganizationByName(ctx, req.OrgName)
	if err != nil {
		return nil, err
	}

	opts := &database.UpdateOrganizationOptions{
		Name:                                req.OrgName,
		Description:                         org.Description,
		QuotaProjects:                       org.QuotaProjects,
		QuotaDeployments:                    org.QuotaDeployments,
		QuotaSlotsTotal:                     org.QuotaSlotsTotal,
		QuotaSlotsPerDeployment:             org.QuotaSlotsPerDeployment,
		QuotaOutstandingInvites:             org.QuotaOutstandingInvites,
		QuotaStorageLimitBytesPerDeployment: org.QuotaStorageLimitBytesPerDeployment,
		BillingCustomerID:                   req.BillingCustomerId,
	}

	org, err = s.admin.DB.UpdateOrganization(ctx, org.ID, opts)
	if err != nil {
		return nil, err
	}

	// get subscriptions if present
	subs, err := s.admin.Biller.GetSubscriptionsForCustomer(ctx, org.BillingCustomerID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	var subscriptions []*adminv1.Subscription
	for _, sub := range subs {
		subscriptions = append(subscriptions, subscriptionToDTO(sub))
	}

	return &adminv1.SudoUpdateOrganizationBillingCustomerResponse{
		Organization:  organizationToDTO(org),
		Subscriptions: subscriptions,
	}, nil
}

func (s *Server) ListPublicBillingPlans(ctx context.Context, req *adminv1.ListPublicBillingPlansRequest) (*adminv1.ListPublicBillingPlansResponse, error) {
	observability.AddRequestAttributes(ctx)

	// no permissions required to list public billing plans
	plans, err := s.admin.Biller.GetPublicPlans(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	var dtos []*adminv1.BillingPlan
	for _, plan := range plans {
		dtos = append(dtos, billingPlanToDTO(plan))
	}

	return &adminv1.ListPublicBillingPlansResponse{
		Plans: dtos,
	}, nil
}

func organizationToDTO(o *database.Organization) *adminv1.Organization {
	return &adminv1.Organization{
		Id:          o.ID,
		Name:        o.Name,
		Description: o.Description,
		Quotas: &adminv1.OrganizationQuotas{
			Projects:                       uint32(o.QuotaProjects),
			Deployments:                    uint32(o.QuotaDeployments),
			SlotsTotal:                     uint32(o.QuotaSlotsTotal),
			SlotsPerDeployment:             uint32(o.QuotaSlotsPerDeployment),
			OutstandingInvites:             uint32(o.QuotaOutstandingInvites),
			StorageLimitBytesPerDeployment: uint64(o.QuotaStorageLimitBytesPerDeployment),
		},
		BillingCustomerId: o.BillingCustomerID,
		CreatedOn:         timestamppb.New(o.CreatedOn),
		UpdatedOn:         timestamppb.New(o.UpdatedOn),
	}
}

func subscriptionToDTO(sub *billing.Subscription) *adminv1.Subscription {
	return &adminv1.Subscription{
		Id:                           sub.ID,
		PlanId:                       sub.Plan.BillerID,
		PlanName:                     sub.Plan.Name,
		StartDate:                    timestamppb.New(sub.StartDate),
		EndDate:                      timestamppb.New(sub.EndDate),
		CurrentBillingCycleStartDate: timestamppb.New(sub.CurrentBillingCycleStartDate),
		CurrentBillingCycleEndDate:   timestamppb.New(sub.CurrentBillingCycleEndDate),
		TrialEndDate:                 timestamppb.New(sub.TrialEndDate),
	}
}

func billingPlanToDTO(plan *billing.Plan) *adminv1.BillingPlan {
	return &adminv1.BillingPlan{
		BillerId:        plan.BillerID,
		RillId:          plan.RillID,
		Name:            plan.Name,
		Description:     plan.Description,
		TrialPeriodDays: uint32(plan.TrialPeriodDays),
		Quotas: &adminv1.Quotas{
			Projects:                       valOrEmptyString(plan.Quotas.NumProjects),
			Deployments:                    valOrEmptyString(plan.Quotas.NumDeployments),
			SlotsTotal:                     valOrEmptyString(plan.Quotas.NumSlotsTotal),
			SlotsPerDeployment:             valOrEmptyString(plan.Quotas.NumSlotsPerDeployment),
			OutstandingInvites:             valOrEmptyString(plan.Quotas.NumOutstandingInvites),
			StorageLimitBytesPerDeployment: val64OrEmptyString(plan.Quotas.StorageLimitBytesPerDeployment),
		},
	}
}

func valOrEmptyString(v *int) string {
	if v != nil {
		return strconv.Itoa(*v)
	}
	return ""
}

func val64OrEmptyString(v *int64) string {
	if v != nil {
		return strconv.FormatInt(*v, 10)
	}
	return ""
}

func whitelistedDomainToPB(a *database.OrganizationWhitelistedDomainWithJoinedRoleNames) *adminv1.WhitelistedDomain {
	return &adminv1.WhitelistedDomain{
		Domain: a.Domain,
		Role:   a.RoleName,
	}
}
