package subsonic

import "github.com/gin-gonic/gin"

func (s *Subsonic) handlePing(c *gin.Context) {
	resp := NewResponse(ResponseStatusOK)
	resp.Ping = &Ping{}
	s.sendResponse(c, resp)
}
