package netease

import (
	"context"
	"fmt"

	"github.com/chaunsin/netease-cloud-music/api/weapi"
	"github.com/stkevintan/miko/pkg/types"
)

func (s *NMProvider) User(ctx context.Context) (*types.User, error) {
	user, err := s.request.GetUserInfo(ctx, &weapi.GetUserInfoReq{})
	if err != nil {
		return nil, fmt.Errorf("GetUserInfo: %s", err)
	}
	if user.Profile == nil {
		return nil, fmt.Errorf("get user info: no logged in user found")
	}
	return &types.User{
		Username: user.Profile.Nickname,
		UserID:   user.Profile.UserId,
	}, nil
}
