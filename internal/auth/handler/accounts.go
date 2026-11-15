package handler

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/kilip/opus/internal/auth/domain"
	"github.com/kilip/opus/internal/platform/httpx"
)

// listAccounts godoc
// @Summary List linked accounts
// @Description Returns all OAuth accounts linked to the authenticated user
// @ID authListAccounts
// @Tags auth
// @Produce json
// @Success 200 {object} httpx.Response[[]domain.AccountResponse]
// @Failure 401 {object} httpx.ErrorResponse
// @Router /auth/accounts [get]
func (h *Handler) listAccounts(c fiber.Ctx) error {
	sess, err := h.svc.Session(c.Context(), c.Cookies(SessionCookie))
	if err != nil {
		return mapErr(err)
	}
	accounts, err := h.svc.ListAccounts(c.Context(), sess.UserID)
	if err != nil {
		return mapErr(err)
	}
	res := make([]domain.AccountResponse, len(accounts))
	for i, a := range accounts {
		res[i] = domain.AccountResponse{
			ID:        a.ID,
			Provider:  a.Provider,
			Email:     a.Email,
			CreatedAt: a.CreatedAt,
		}
	}
	return httpx.OK(c, res)
}

// unlinkAccount godoc
// @Summary Unlink account
// @Description Unlinks an OAuth account from the authenticated user
// @ID authUnlinkAccount
// @Tags auth
// @Param id path string true "Account ID"
// @Success 204 "No Content"
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Router /auth/accounts/{id} [delete]
func (h *Handler) unlinkAccount(c fiber.Ctx) error {
	sess, err := h.svc.Session(c.Context(), c.Cookies(SessionCookie))
	if err != nil {
		return mapErr(err)
	}
	accountID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httpx.NotFound(domain.ErrAccountNotFound)
	}
	if err := h.svc.UnlinkAccount(c.Context(), sess.UserID, accountID, clientAuditMeta(c)); err != nil {
		return mapErr(err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}
