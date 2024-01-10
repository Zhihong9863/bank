package gapi

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	mockdb "github.com/techschool/bank/db/mock"
	db "github.com/techschool/bank/db/sqlc"
	"github.com/techschool/bank/pb"
	"github.com/techschool/bank/util"
	"github.com/techschool/bank/worker"
	mockwk "github.com/techschool/bank/worker/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type eqCreateUserTxParamsMatcher struct {
	arg      db.CreateUserTxParams
	password string
	user     db.User
}

func (expected eqCreateUserTxParamsMatcher) Matches(x interface{}) bool {
	actualArg, ok := x.(db.CreateUserTxParams)
	if !ok {
		return false
	}

	err := util.CheckPassword(expected.password, actualArg.HashedPassword)
	if err != nil {
		return false
	}

	expected.arg.HashedPassword = actualArg.HashedPassword
	if !reflect.DeepEqual(expected.arg.CreateUserParams, actualArg.CreateUserParams) {
		return false
	}

	err = actualArg.AfterCreate(expected.user)
	return err == nil
}

func (e eqCreateUserTxParamsMatcher) String() string {
	return fmt.Sprintf("matches arg %v and password %v", e.arg, e.password)
}

func EqCreateUserTxParams(arg db.CreateUserTxParams, password string, user db.User) gomock.Matcher {
	return eqCreateUserTxParamsMatcher{arg, password, user}
}

func randomUser(t *testing.T, role string) (user db.User, password string) {
	password = util.RandomString(6)
	hashedPassword, err := util.HashPassword(password)
	require.NoError(t, err)

	user = db.User{
		Username:       util.RandomOwner(),
		Role:           role,
		HashedPassword: hashedPassword,
		FullName:       util.RandomOwner(),
		Email:          util.RandomEmail(),
	}
	return
}

func TestCreateUserAPI(t *testing.T) {
	user, password := randomUser(t, util.DepositorRole)

	testCases := []struct {
		name          string
		req           *pb.CreateUserRequest
		buildStubs    func(store *mockdb.MockStore, taskDistributor *mockwk.MockTaskDistributor)
		checkResponse func(t *testing.T, res *pb.CreateUserResponse, err error)
	}{
		{
			name: "OK",
			req: &pb.CreateUserRequest{
				Username: user.Username,
				Password: password,
				FullName: user.FullName,
				Email:    user.Email,
			},
			buildStubs: func(store *mockdb.MockStore, taskDistributor *mockwk.MockTaskDistributor) {
				arg := db.CreateUserTxParams{
					CreateUserParams: db.CreateUserParams{
						Username: user.Username,
						FullName: user.FullName,
						Email:    user.Email,
					},
				}
				store.EXPECT().
					CreateUserTx(gomock.Any(), EqCreateUserTxParams(arg, password, user)).
					Times(1).
					Return(db.CreateUserTxResult{User: user}, nil)

				taskPayload := &worker.PayloadSendVerifyEmail{
					Username: user.Username,
				}
				taskDistributor.EXPECT().
					DistributeTaskSendVerifyEmail(gomock.Any(), taskPayload, gomock.Any()).
					Times(1).
					Return(nil)
			},
			checkResponse: func(t *testing.T, res *pb.CreateUserResponse, err error) {
				require.NoError(t, err)
				require.NotNil(t, res)
				createdUser := res.GetUser()
				require.Equal(t, user.Username, createdUser.Username)
				require.Equal(t, user.FullName, createdUser.FullName)
				require.Equal(t, user.Email, createdUser.Email)
			},
		},
		{
			name: "InternalError",
			req: &pb.CreateUserRequest{
				Username: user.Username,
				Password: password,
				FullName: user.FullName,
				Email:    user.Email,
			},
			buildStubs: func(store *mockdb.MockStore, taskDistributor *mockwk.MockTaskDistributor) {
				store.EXPECT().
					CreateUserTx(gomock.Any(), gomock.Any()).
					Times(1).
					Return(db.CreateUserTxResult{}, sql.ErrConnDone)

				taskDistributor.EXPECT().
					DistributeTaskSendVerifyEmail(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(t *testing.T, res *pb.CreateUserResponse, err error) {
				// 首先，确保返回了错误
				require.Error(t, err)

				// 尝试从错误中获取 gRPC 状态信息
				st, ok := status.FromError(err)

				// 如果错误是 gRPC 状态错误
				if ok {
					// 检查状态码是否为 Internal
					require.Equal(t, codes.Internal, st.Code())
				} else {
					// 如果错误不是 gRPC 状态错误，则在这里记录或处理
					t.Log("Error is not a gRPC status error")
				}
			},
		},
		{
			name: "DuplicateUsername",
			req: &pb.CreateUserRequest{
				Username: user.Username,
				Password: password,
				FullName: user.FullName,
				Email:    user.Email,
			},
			buildStubs: func(store *mockdb.MockStore, taskDistributor *mockwk.MockTaskDistributor) {
				store.EXPECT().
					CreateUserTx(gomock.Any(), gomock.Any()).
					Times(1).
					Return(db.CreateUserTxResult{}, db.ErrUniqueViolation)

				taskDistributor.EXPECT().
					DistributeTaskSendVerifyEmail(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(t *testing.T, res *pb.CreateUserResponse, err error) {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				require.Equal(t, codes.AlreadyExists, st.Code())
			},
		},
		{
			name: "InvalidEmail",
			req: &pb.CreateUserRequest{
				Username: user.Username,
				Password: password,
				FullName: user.FullName,
				Email:    "invalid-email",
			},
			buildStubs: func(store *mockdb.MockStore, taskDistributor *mockwk.MockTaskDistributor) {
				store.EXPECT().
					CreateUserTx(gomock.Any(), gomock.Any()).
					Times(0)

				taskDistributor.EXPECT().
					DistributeTaskSendVerifyEmail(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(t *testing.T, res *pb.CreateUserResponse, err error) {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				require.Equal(t, codes.InvalidArgument, st.Code())
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			storeCtrl := gomock.NewController(t)
			defer storeCtrl.Finish()
			store := mockdb.NewMockStore(storeCtrl)

			taskCtrl := gomock.NewController(t)
			defer taskCtrl.Finish()
			taskDistributor := mockwk.NewMockTaskDistributor(taskCtrl)

			tc.buildStubs(store, taskDistributor)
			server := newTestServer(t, store, taskDistributor)

			res, err := server.CreateUser(context.Background(), tc.req)
			tc.checkResponse(t, res, err)
		})
	}
}

/*
单元测试通常的写法思路是模拟外部依赖，然后调用要测试的函数，并验证结果是否符合预期。对于 CreateUser API 的测试，这个过程涉及以下几个步骤：

1. 准备输入数据
这通常涉及创建API请求所需的所有参数。在测试代码中，
这是通过 randomUser 函数来完成的，它生成了一个随机的用户和密码。

2. 构建模拟（Mocking）行为
使用 gomock 库，您可以创建模拟对象，
这些对象在测试中替代真实的依赖。
在测试中，MockStore 用于模拟数据库操作，
MockTaskDistributor 用于模拟异步任务的分发。

模拟对象可以定义期望的行为，例如期望调用特定的函数，并返回特定的结果。
这样，当测试的函数调用这些依赖时，它们会返回您设定的模拟结果。

3. 定义测试案例
测试案例定义了测试的具体情景，包括预期的输入、预期的行为（模拟的返回值或行为），
以及预期的输出（函数的返回值或状态变化）。

在测试代码中，testCases 数组中的每个元素都定义了一个名字、
一个请求对象、一个 buildStubs 函数来设置模拟行为，
以及一个 checkResponse 函数来验证输出。

4. 执行测试并验证结果
测试循环通过遍历所有测试案例，对每个案例调用要测试的函数，
并使用 checkResponse 函数来验证结果是否符合预期。

在测试代码中，checkResponse 函数确保没有错误发生，
并且返回的用户信息与输入数据相匹配。

代码细节
gomock的使用：gomock 允许您指定当测试代码调用某个函数时应该发生什么。
例如，当调用 CreateUserTx 时，可以设置一个期望值，
表示这个函数应该被调用一次，并返回一个特定的结果。

参数匹配器：EqCreateUserTxParams 是一个自定义的参数匹配器，
它确保传递给 CreateUserTx 的参数符合特定的条件。
在这里，它检查传递的参数是否与 CreateUserTxParams 结构体匹配，
并且密码是否经过正确的哈希处理。

验证响应：在每个测试案例中，checkResponse 都是一个闭包，
它接受 t *testing.T 参数，可以使用 require 或 assert 函数来验证测试结果。
*/
