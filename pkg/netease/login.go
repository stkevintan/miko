package netease

import (
	"context"
	"fmt"

	"github.com/chaunsin/netease-cloud-music/api/weapi"
	"github.com/stkevintan/miko/pkg/types"
)

func (s *NMProvider) Login(ctx context.Context) (*types.LoginResult, error) {
	user, err := s.request.GetUserInfo(ctx, &weapi.GetUserInfoReq{})
	if err != nil {
		return nil, fmt.Errorf("GetUserInfo: %s", err)
	}
	return &types.LoginResult{
		Username: user.Profile.Nickname,
		UserID:   user.Profile.UserId,
	}, nil
}
