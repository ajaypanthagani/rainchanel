package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"rainchanel.com/internal/api/request"
	"rainchanel.com/internal/api/response"
	"rainchanel.com/internal/service"
)

type AuthHandler interface {
	Register(*gin.Context)
	Login(*gin.Context)
}

type authHandler struct {
	authService service.AuthService
}

func NewAuthHandler(authService service.AuthService) AuthHandler {
	return &authHandler{
		authService: authService,
	}
}

func (h *authHandler) Register(ctx *gin.Context) {
	var req request.RegisterRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, response.Response{
			Error: &response.Error{
				Code:    http.StatusBadRequest,
				Message: err.Error(),
			},
		})
		return
	}

	if err := h.authService.Register(req.Username, req.Password); err != nil {
		ctx.JSON(http.StatusBadRequest, response.Response{
			Error: &response.Error{
				Code:    http.StatusBadRequest,
				Message: err.Error(),
			},
		})
		return
	}

	ctx.JSON(http.StatusOK, response.Response{
		Data: response.RegisterResponse{
			Message: "User registered successfully",
		},
	})
}

func (h *authHandler) Login(ctx *gin.Context) {
	var req request.LoginRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, response.Response{
			Error: &response.Error{
				Code:    http.StatusBadRequest,
				Message: err.Error(),
			},
		})
		return
	}

	token, userID, username, err := h.authService.Login(req.Username, req.Password)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, response.Response{
			Error: &response.Error{
				Code:    http.StatusUnauthorized,
				Message: err.Error(),
			},
		})
		return
	}

	ctx.JSON(http.StatusOK, response.Response{
		Data: response.LoginResponse{
			Token:    token,
			UserID:   userID,
			Username: username,
		},
	})
}

