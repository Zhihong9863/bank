package gapi

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
	mockdb "github.com/techschool/bank/db/mock"
	db "github.com/techschool/bank/db/sqlc"
	"github.com/techschool/bank/pb"
	"github.com/techschool/bank/token"
	"github.com/techschool/bank/util"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestUpdateUserAPI(t *testing.T) {
	user, _ := randomUser(t, util.DepositorRole)
	other, _ := randomUser(t, util.DepositorRole)
	banker, _ := randomUser(t, util.BankerRole)

	newName := util.RandomOwner()
	newEmail := util.RandomEmail()
	invalidEmail := "invalid-email"

	testCases := []struct {
		name          string
		req           *pb.UpdateUserRequest
		buildStubs    func(store *mockdb.MockStore)
		buildContext  func(t *testing.T, tokenMaker token.Maker) context.Context
		checkResponse func(t *testing.T, res *pb.UpdateUserResponse, err error)
	}{
		{
			name: "OK",
			req: &pb.UpdateUserRequest{
				Username: user.Username,
				FullName: &newName,
				Email:    &newEmail,
			},
			buildStubs: func(store *mockdb.MockStore) {
				arg := db.UpdateUserParams{
					Username: user.Username,
					FullName: pgtype.Text{
						String: newName,
						Valid:  true,
					},
					Email: pgtype.Text{
						String: newEmail,
						Valid:  true,
					},
				}
				updatedUser := db.User{
					Username:          user.Username,
					HashedPassword:    user.HashedPassword,
					FullName:          newName,
					Email:             newEmail,
					PasswordChangedAt: user.PasswordChangedAt,
					CreatedAt:         user.CreatedAt,
					IsEmailVerified:   user.IsEmailVerified,
				}
				store.EXPECT().
					UpdateUser(gomock.Any(), gomock.Eq(arg)).
					Times(1).
					Return(updatedUser, nil)
			},
			buildContext: func(t *testing.T, tokenMaker token.Maker) context.Context {
				return newContextWithBearerToken(t, tokenMaker, user.Username, user.Role, time.Minute)
			},
			checkResponse: func(t *testing.T, res *pb.UpdateUserResponse, err error) {
				require.NoError(t, err)
				require.NotNil(t, res)
				updatedUser := res.GetUser()
				require.Equal(t, user.Username, updatedUser.Username)
				require.Equal(t, newName, updatedUser.FullName)
				require.Equal(t, newEmail, updatedUser.Email)
			},
		},
		{
			name: "BankerCanUpdateUserInfo",
			req: &pb.UpdateUserRequest{
				Username: user.Username,
				FullName: &newName,
				Email:    &newEmail,
			},
			buildStubs: func(store *mockdb.MockStore) {
				arg := db.UpdateUserParams{
					Username: user.Username,
					FullName: pgtype.Text{
						String: newName,
						Valid:  true,
					},
					Email: pgtype.Text{
						String: newEmail,
						Valid:  true,
					},
				}
				updatedUser := db.User{
					Username:          user.Username,
					HashedPassword:    user.HashedPassword,
					FullName:          newName,
					Email:             newEmail,
					PasswordChangedAt: user.PasswordChangedAt,
					CreatedAt:         user.CreatedAt,
					IsEmailVerified:   user.IsEmailVerified,
				}
				store.EXPECT().
					UpdateUser(gomock.Any(), gomock.Eq(arg)).
					Times(1).
					Return(updatedUser, nil)
			},
			buildContext: func(t *testing.T, tokenMaker token.Maker) context.Context {
				return newContextWithBearerToken(t, tokenMaker, banker.Username, banker.Role, time.Minute)
			},
			checkResponse: func(t *testing.T, res *pb.UpdateUserResponse, err error) {
				require.NoError(t, err)
				require.NotNil(t, res)
				updatedUser := res.GetUser()
				require.Equal(t, user.Username, updatedUser.Username)
				require.Equal(t, newName, updatedUser.FullName)
				require.Equal(t, newEmail, updatedUser.Email)
			},
		},
		{
			name: "OtherDepositorCannotUpdateThisUserInfo",
			req: &pb.UpdateUserRequest{
				Username: user.Username,
				FullName: &newName,
				Email:    &newEmail,
			},
			buildStubs: func(store *mockdb.MockStore) {
				log.Println("Building stubs for the test case")
				store.EXPECT().
					UpdateUser(gomock.Any(), gomock.Any()).
					Times(0)
			},
			buildContext: func(t *testing.T, tokenMaker token.Maker) context.Context {
				return newContextWithBearerToken(t, tokenMaker, other.Username, other.Role, time.Minute)
			},
			checkResponse: func(t *testing.T, res *pb.UpdateUserResponse, err error) {
				log.Printf("Checking response with result: %v, error: %v", res, err)
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				require.Equal(t, codes.PermissionDenied, st.Code())
			},
		},
		{
			name: "InvalidEmail",
			req: &pb.UpdateUserRequest{
				Username: user.Username,
				FullName: &newName,
				Email:    &invalidEmail,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					UpdateUser(gomock.Any(), gomock.Any()).
					Times(0)
			},
			buildContext: func(t *testing.T, tokenMaker token.Maker) context.Context {
				return newContextWithBearerToken(t, tokenMaker, user.Username, user.Role, time.Minute)
			},
			checkResponse: func(t *testing.T, res *pb.UpdateUserResponse, err error) {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				require.Equal(t, codes.InvalidArgument, st.Code())
			},
		},
		{
			name: "ExpiredToken",
			req: &pb.UpdateUserRequest{
				Username: user.Username,
				FullName: &newName,
				Email:    &newEmail,
			},
			buildStubs: func(store *mockdb.MockStore) {
				log.Println("Building stubs for the test case")
				store.EXPECT().
					UpdateUser(gomock.Any(), gomock.Any()).
					Times(0)
			},
			buildContext: func(t *testing.T, tokenMaker token.Maker) context.Context {
				return newContextWithBearerToken(t, tokenMaker, user.Username, user.Role, -time.Minute)
			},
			checkResponse: func(t *testing.T, res *pb.UpdateUserResponse, err error) {
				log.Printf("Checking response with result: %v, error: %v", res, err)
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				require.Equal(t, codes.Unauthenticated, st.Code())
			},
		},
		{
			name: "NoAuthorization",
			req: &pb.UpdateUserRequest{
				Username: user.Username,
				FullName: &newName,
				Email:    &newEmail,
			},
			buildStubs: func(store *mockdb.MockStore) {
				log.Println("Building stubs for the test case")
				store.EXPECT().
					UpdateUser(gomock.Any(), gomock.Any()).
					Times(0)
			},
			buildContext: func(t *testing.T, tokenMaker token.Maker) context.Context {
				return context.Background()
			},
			checkResponse: func(t *testing.T, res *pb.UpdateUserResponse, err error) {
				log.Printf("Checking response with result: %v, error: %v", res, err)
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				require.Equal(t, codes.Unauthenticated, st.Code())
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			storeCtrl := gomock.NewController(t)
			defer storeCtrl.Finish()
			store := mockdb.NewMockStore(storeCtrl)

			tc.buildStubs(store)
			server := newTestServer(t, store, nil)

			ctx := tc.buildContext(t, server.tokenMaker)
			res, err := server.UpdateUser(ctx, tc.req)
			tc.checkResponse(t, res, err)
		})
	}
}

/*
测试代码通过定义一系列测试案例（testCases）来验证 UpdateUser API 的行为。

每个测试案例都有以下组成部分：

name: 测试案例的名称。
req: 一个 pb.UpdateUserRequest 请求，代表传递给 UpdateUser API 的参数。
buildStubs: 一个函数，用于配置数据库和任务分发器的模拟行为。
buildContext: 一个函数，用于构建并返回一个带有授权令牌的上下文，模拟授权用户。
checkResponse: 一个函数，用于在调用 UpdateUser 之后验证响应和错误。
测试案例
OK: 验证当所有条件都满足时，用户信息成功更新。它模拟数据库成功更新用户并返回更新后的用户信息。

BankerCanUpdateUserInfo: 验证具有银行家角色的用户可以更新其他用户的信息。
它模拟具有足够权限的用户更新了特定用户的信息。

OtherDepositorCannotUpdateThisUserInfo:
验证一个存款人不能更新另一个存款人的信息。它模拟一个权限不足的场景，
期望 API 返回 codes.PermissionDenied。

InvalidEmail: 验证当提供无效电子邮件时，更新请求被拒绝。
它模拟了一个输入验证失败的场景。

ExpiredToken: 验证当提供过期的令牌时，请求被认为是未经授权的。
它模拟一个令牌过期的场景。

NoAuthorization: 验证如果请求没有提供授权信息，
请求被拒绝。它模拟一个没有提供任何授权令牌的场景。

buildStubs 函数
buildStubs 函数为每个测试案例配置模拟数据库（MockStore）
和任务分发器（MockTaskDistributor）。这个函数通常会设置模拟对象的预期行为，
例如，期望特定的函数被调用，并定义它应该返回什么结果。

buildContext 函数
buildContext 函数构建并返回一个上下文（context.Context），
这个上下文包含了授权信息，模拟了真实的用户令牌。
这对于测试需要验证用户权限的 API 特别重要。

checkResponse 函数
checkResponse 函数在 API 调用之后运行，
用于验证返回的响应和错误是否符合预期。这可能包括检查响应是否包含正确的用户信息，
或者确保在应该返回错误的情况下确实返回了错误。

执行测试案例
循环遍历 testCases，为每个案例运行一个测试。在每个测试中：

使用 gomock.NewController 和 mockdb.NewMockStore 创建模拟的数据库接口。
调用 buildStubs 设置期望行为。
调用 buildContext 构建上下文。
执行 UpdateUser API 调用。
使用 checkResponse 验证结果。

体现认证的做法
认证主要体现在 buildContext 和测试逻辑的构建上。
通过为每个测试案例提供特定的令牌和角色，
可以模拟不同的用户状态和权限级别，从而测试 API 在各种安全和权限场景下的行为。

总结
这个测试案例展示了如何在没有真实数据库和服务的情况下，
通过模拟和上下文构建来全面测试带有认证和权限检查的 API。
*/
