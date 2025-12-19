package netease

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/chaunsin/netease-cloud-music/api/weapi"
	"github.com/chaunsin/netease-cloud-music/pkg/cookiecloud"
	"github.com/stkevintan/miko/pkg/types"
)

func (s *NMProvider) Login(ctx context.Context, c *types.LoginArgs) (*types.LoginResult, error) {
	if c.Server == "" {
		return nil, fmt.Errorf("server is required")
	}
	if c.UUID == "" {
		return nil, fmt.Errorf("uuid is required")
	}
	if c.Password == "" {
		return nil, fmt.Errorf("password is required")
	}

	nctx, cancel := context.WithTimeout(ctx, c.Timeout)
	defer cancel()

	cc, err := cookiecloud.NewClient(&cookiecloud.Config{
		ApiUrl:  c.Server,
		Timeout: c.Timeout,
		Retry:   3,
	})

	if err != nil {
		return nil, fmt.Errorf("NewClient: %w", err)
	}
	resp, err := cc.Get(nctx, &cookiecloud.GetReq{
		Uuid:     c.UUID,
		Password: c.Password,
	})
	if err != nil {
		return nil, fmt.Errorf("cookiecloud.Get: %w", err)
	}
	var cnt int
	for domain, cookies := range resp.CookieData {
		if !strings.HasSuffix(domain, "music.163.com") {
			continue
		}
		// Parse the domain into a URL (adjust a scheme if needed)
		u, err := url.Parse("https://music.163.com")
		if err != nil {
			return nil, fmt.Errorf("failed to parse domain URL: %v", err)
		}

		// Convert a custom cookie type to http.Cookie
		var httpCookies []*http.Cookie
		for _, v := range cookies {
			if v.Name == "MUSIC_U" {
				cnt++
			}
			httpCookies = append(httpCookies, &http.Cookie{
				Domain:   domain, // Use original domain value
				Expires:  v.GetExpired(),
				HttpOnly: v.HttpOnly,
				Name:     v.Name,
				Path:     v.Path,
				Secure:   v.Secure,
				Value:    v.Value,
				SameSite: sameSite(v.SameSite),
				// Quoted:   false,
			})
		}
		if len(httpCookies) > 0 {
			s.cli.SetCookies(u, httpCookies)
		}
	}

	if cnt == 0 {
		return nil, fmt.Errorf("请确认已登录网页版网易云音乐，并且cookie已经同步到cookiecloud")
	}

	// 查询登录信息是否成功
	user, err := s.request.GetUserInfo(ctx, &weapi.GetUserInfoReq{})
	if err != nil {
		return nil, fmt.Errorf("GetUserInfo: %s", err)
	}
	return &types.LoginResult{
		Username: user.Profile.Nickname,
		UserID:   user.Profile.UserId,
	}, nil
}
