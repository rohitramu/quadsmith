package auth

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	pb "quadsmith/shared/gen/quadsmith/api"
	"quadsmith/shared/gen/quadsmith/api/apiconnect"
)

func TestAllowedBannedProcedures(t *testing.T) {
	allowed := []string{
		apiconnect.AuthServiceRefreshTokenProcedure,
		apiconnect.AuthServiceLogoutProcedure,
		apiconnect.AuthServiceUpdateProfileProcedure,
		apiconnect.AuthServiceListActiveSessionsProcedure,
		apiconnect.AuthServiceRevokeSessionProcedure,
		apiconnect.AuthServiceRevokeOtherSessionsProcedure,
		apiconnect.AuthServiceGetCurrentUserProcedure,
		apiconnect.AuthServiceListUserEmailsProcedure,
		apiconnect.AuthServiceSendDataExportVerificationProcedure,
		apiconnect.AuthServiceVerifyDataExportProcedure,
		apiconnect.AuthServiceSendDeleteAccountVerificationProcedure,
		apiconnect.AuthServiceDeleteAccountProcedure,
		apiconnect.AuthServiceCancelAccountDeletionProcedure,
		apiconnect.UserServiceGetUserBanStatusProcedure,
	}

	for _, proc := range allowed {
		if !IsAllowedBannedProcedure(proc) {
			t.Errorf("expected procedure %q to be allowed for banned users", proc)
		}
	}

	blocked := []string{
		apiconnect.AuthServiceChangeUsernameProcedure,
		apiconnect.AuthServiceSendAddEmailOtpProcedure,
		apiconnect.AuthServiceVerifyAddEmailOtpProcedure,
		apiconnect.AuthServiceMergeAccountProcedure,
		apiconnect.AuthServiceSetPrimaryEmailProcedure,
		apiconnect.AuthServiceRemoveUserEmailProcedure,
		apiconnect.SocialServiceCreateReviewProcedure,
		apiconnect.SocialServiceCreateQuestionProcedure,
		apiconnect.SocialServiceCreateAnswerProcedure,
		apiconnect.BuildServiceCreateBuildProcedure,
		apiconnect.BuildServiceDeleteBuildProcedure,
		apiconnect.UserServiceBanUserProcedure,
		apiconnect.UserServiceUnbanUserProcedure,
		apiconnect.UserServiceListBansProcedure,
	}

	for _, proc := range blocked {
		if IsAllowedBannedProcedure(proc) {
			t.Errorf("expected procedure %q to be BLOCKED for banned users", proc)
		}
	}
}

func TestUserServiceModeration_Unauthenticated(t *testing.T) {
	svc := &UserService{}
	ctx := context.Background()

	// 1. BanUser
	_, err := svc.BanUser(ctx, connect.NewRequest(&pb.BanUserRequest{
		UserId: "test-user",
		Reason: "Violation",
	}))
	if err == nil {
		t.Fatal("expected unauthenticated error for BanUser, got nil")
	}
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("expected CodeUnauthenticated, got %v", connect.CodeOf(err))
	}

	// 2. UnbanUser
	_, err = svc.UnbanUser(ctx, connect.NewRequest(&pb.UnbanUserRequest{
		UserId: "test-user",
	}))
	if err == nil {
		t.Fatal("expected unauthenticated error for UnbanUser, got nil")
	}
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("expected CodeUnauthenticated, got %v", connect.CodeOf(err))
	}

	// 3. ListBans
	_, err = svc.ListBans(ctx, connect.NewRequest(&pb.ListBansRequest{}))
	if err == nil {
		t.Fatal("expected unauthenticated error for ListBans, got nil")
	}
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("expected CodeUnauthenticated, got %v", connect.CodeOf(err))
	}

	// 4. GetUserBanStatus without identifier and unauthenticated
	_, err = svc.GetUserBanStatus(ctx, connect.NewRequest(&pb.GetUserBanStatusRequest{}))
	if err == nil {
		t.Fatal("expected unauthenticated error for GetUserBanStatus without args, got nil")
	}
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("expected CodeUnauthenticated, got %v", connect.CodeOf(err))
	}
}

func TestUserServiceModeration_Validation(t *testing.T) {
	svc := &UserService{}
	userCtx := &UserContext{
		UserUUID: "admin-uuid",
		UserID:   "admin_user",
		Role:     "ADMIN",
	}
	ctx := WithUserContext(context.Background(), userCtx)

	// BanUser: missing user identifier
	_, err := svc.BanUser(ctx, connect.NewRequest(&pb.BanUserRequest{
		UserId: "   ",
		Reason: "Spam",
	}))
	if err == nil || connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("expected CodeInvalidArgument for empty user_id, got %v", err)
	}

	// BanUser: missing reason
	_, err = svc.BanUser(ctx, connect.NewRequest(&pb.BanUserRequest{
		UserId: "spammer",
		Reason: "   ",
	}))
	if err == nil || connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("expected CodeInvalidArgument for empty reason, got %v", err)
	}

	// UnbanUser: missing user identifier
	_, err = svc.UnbanUser(ctx, connect.NewRequest(&pb.UnbanUserRequest{
		UserId: "   ",
	}))
	if err == nil || connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("expected CodeInvalidArgument for empty user_id in UnbanUser, got %v", err)
	}
}
