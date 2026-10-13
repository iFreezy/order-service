package horder

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/iFreezy/order-service/internal/app/entity"
	rhandler "github.com/iFreezy/order-service/internal/app/handler/http"
	"github.com/iFreezy/order-service/internal/app/service"
	"github.com/iFreezy/order-service/internal/pkg/http/httph"
)

type handler struct{ srv service.Order }

func NewHandler(srv service.Order) rhandler.Order { return &handler{srv: srv} }
func (h *handler) Create(c *gin.Context) {
	var req entity.RequestOrderCreate
	if err := c.ShouldBindJSON(&req); err != nil {
		httph.HandleError(c.Writer, c.Request, entity.ErrIncorrectParameters)
		return
	}
	order, err := h.srv.Create(c.Request.Context(), req)
	if err != nil {
		httph.HandleError(c.Writer, c.Request, err)
		return
	}
	httph.SendJSON(c.Writer, http.StatusCreated, createResponse(order))
}
func (h *handler) GetByGUID(c *gin.Context) {
	guid, ok := parseGUID(c)
	if !ok {
		return
	}
	order, err := h.srv.GetByGUID(c.Request.Context(), guid)
	if err != nil {
		httph.HandleError(c.Writer, c.Request, err)
		return
	}
	httph.SendJSON(c.Writer, http.StatusOK, entity.ResponseOrderGet{ResponseOrderCreate: createResponse(order), UpdatedAt: order.UpdatedAt})
}
func (h *handler) Update(c *gin.Context) {
	guid, ok := parseGUID(c)
	if !ok {
		return
	}
	var req entity.RequestOrderUpdate
	if err := c.ShouldBindJSON(&req); err != nil {
		httph.HandleError(c.Writer, c.Request, entity.ErrIncorrectParameters)
		return
	}
	order, err := h.srv.Update(c.Request.Context(), guid, req)
	if err != nil {
		httph.HandleError(c.Writer, c.Request, err)
		return
	}
	httph.SendJSON(c.Writer, http.StatusOK, entity.ResponseOrderUpdate{ResponseOrderCreate: createResponse(order), UpdatedAt: order.UpdatedAt})
}
func (h *handler) Delete(c *gin.Context) {
	guid, ok := parseGUID(c)
	if !ok {
		return
	}
	if err := h.srv.Delete(c.Request.Context(), guid); err != nil {
		httph.HandleError(c.Writer, c.Request, err)
		return
	}
	httph.SendEmpty(c.Writer, http.StatusOK)
}
func (h *handler) List(c *gin.Context) {
	var req entity.RequestOrderList
	if c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			httph.HandleError(c.Writer, c.Request, entity.ErrIncorrectParameters)
			return
		}
	}
	orders, err := h.srv.List(c.Request.Context(), req)
	if err != nil {
		httph.HandleError(c.Writer, c.Request, err)
		return
	}
	resp := entity.ResponseOrderList{Data: make([]entity.ResponseOrderListItem, 0, len(orders))}
	for _, order := range orders {
		resp.Data = append(resp.Data, entity.ResponseOrderListItem{GUID: order.GUID, UserGUID: order.UserGUID, TotalPrice: order.TotalPrice, Currency: order.Currency, Status: order.Status, CreatedAt: order.CreatedAt, UpdatedAt: order.UpdatedAt})
	}
	httph.SendJSON(c.Writer, http.StatusOK, resp)
}
func parseGUID(c *gin.Context) (uuid.UUID, bool) {
	guid, err := uuid.FromString(c.Param("guid"))
	if err != nil {
		httph.HandleError(c.Writer, c.Request, entity.ErrIncorrectParameters)
		return uuid.Nil, false
	}
	return guid, true
}
func createResponse(order entity.Order) entity.ResponseOrderCreate {
	items := make([]entity.ResponseOrderItem, 0, len(order.Items))
	for _, item := range order.Items {
		items = append(items, entity.ResponseOrderItem{GUID: item.GUID, ProductGUID: item.ProductGUID, Quantity: item.Quantity, UnitPrice: item.UnitPrice})
	}
	return entity.ResponseOrderCreate{GUID: order.GUID, UserGUID: order.UserGUID, TotalPrice: order.TotalPrice, Currency: order.Currency, Status: order.Status, Items: items, CreatedAt: order.CreatedAt}
}
