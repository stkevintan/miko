package service

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/chaunsin/netease-cloud-music/api"
	"github.com/chaunsin/netease-cloud-music/api/weapi"
	"github.com/chaunsin/netease-cloud-music/pkg/cookiecloud"
)

// LoginArgs represents internal login arguments for the service
type LoginArgs struct {
	Timeout  time.Duration // 超时时间
	Server   string
	UUID     string
	Password string
}

// LoginResult represents internal login result from the service
type LoginResult struct {
	Username string
	UserID   int64
}

func (s *Service) Login(ctx context.Context, c *LoginArgs) (*LoginResult, error) {
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

	cli := api.New(s.config.NmApi)
	defer cli.Close(nctx)

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
			cli.SetCookies(u, httpCookies)
		}
	}

	if cnt == 0 {
		return nil, fmt.Errorf("请确认已登录网页版网易云音乐，并且cookie已经同步到cookiecloud")
	}

	// 查询登录信息是否成功
	request := weapi.New(cli)
	user, err := request.GetUserInfo(ctx, &weapi.GetUserInfoReq{})
	if err != nil {
		return nil, fmt.Errorf("GetUserInfo: %s", err)
	}
	return &LoginResult{
		Username: user.Profile.Nickname,
		UserID:   user.Profile.UserId,
	}, nil
}
