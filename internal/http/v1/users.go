package v1

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5"
	"gophermart/internal/domain"
	"gophermart/internal/luhn"
	"gophermart/internal/service"
	"io"
	"net/http"
	"strings"
	"time"
)

//TODO make all handlers private?

func (h *Handler) initUserRoutes(r chi.Router) {
	r.Route("/user", func(r chi.Router) {
		r.Post("/register", h.PostRegister())
		r.Post("/login", h.PostLogin())

		r.Group(func(r chi.Router) {
			r.Use(h.checkUserIdentity)

			r.Post("/orders", h.PostOrders())
			r.Get("/orders", h.GetOrders())
			r.Get("/withdrawals", h.GetWithdrawals()) //*

			r.Route("/balance", func(r chi.Router) {
				r.Get("/", h.GetBalance())
				r.Post("/withdraw", h.PostWithdraw())
				r.Get("/withdrawals", h.GetWithdrawals()) //*
			})
		})
	})
}

//TODO tags
type signInInput struct {
	Login    string `json:"login" binding:"required,max=64"`
	Password string `json:"password" binding:"required,min=8,max=64"`
}

//user registration
func (h *Handler)PostRegister() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		inp := signInInput{}
		if err := json.NewDecoder(r.Body).Decode(&inp); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		err := h.services.Users.SignUp(r.Context(), service.UserSignUpInput {
			Login:    inp.Login,
			Password: inp.Password,
		})

		if err != nil {
			if errors.Is(err, domain.ErrUserAlreadyExists) {
				http.Error(w, err.Error(), http.StatusConflict)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		//authentication
		token, err := h.services.Users.SignIn(r.Context(), service.UserSignInInput {
			Login:    inp.Login,
			Password: inp.Password,
		})

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		cookie := &http.Cookie{
			Name: "AccessToken", //TODO make constant
			Value: token.AccessToken,
			Path:  "/",
		}
		http.SetCookie(w, cookie)
		w.WriteHeader(http.StatusOK)
	}
}

type signUpInput struct {
	Login    string `json:"login" binding:"required,max=64"`
	Password string `json:"password" binding:"required,min=8,max=64"`
}

//user authentication
func (h *Handler)PostLogin() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		inp := signUpInput{}
		if err := json.NewDecoder(r.Body).Decode(&inp); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		token, err := h.services.Users.SignIn(r.Context(), service.UserSignInInput {
			Login:    inp.Login,
			Password: inp.Password,
		})

		if err != nil {
			if errors.Is(err, domain.ErrUserNotFound) {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		cookie := &http.Cookie{
			Name: "AccessToken", //TODO make constant
			Value: token.AccessToken,
			Path:  "/",
		}
		http.SetCookie(w, cookie)
		w.WriteHeader(http.StatusOK)
	}
}

//upload the order number by user for calculation
func (h *Handler)PostOrders() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orderID, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		orderIDStr := strings.TrimSpace(string(orderID))

		if !luhn.Valid(orderIDStr) {
			http.Error(w,"invalid order number format", http.StatusUnprocessableEntity)
			return
		}

		var userIDCtx int
		if id := r.Context().Value(UserIDCtxName); id != nil {
			userIDCtx = id.(int)
		}

		order := domain.Order{
			Number: orderIDStr,
			UserID: userIDCtx,
			Status:domain.OrderStatusNew,
			Accrual: 0,
			UploadedAt: domain.Time(time.Now()),
		}

		err = h.services.Users.AddOrder(r.Context(), order)
		if err != nil {
			switch {
			case errors.Is(err, domain.ErrRepeatedOrderRequest):
				http.Error(w, err.Error(), http.StatusOK)
				return
			case errors.Is(err, domain.ErrForeignOrder):
				http.Error(w, err.Error(), http.StatusConflict)
				return
			default:
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		h.services.Updater.AddOrder(order)

		w.WriteHeader(http.StatusAccepted)
	}
}

//get a list of order numbers uploaded by the user, their processing statuses and information about accruals
func (h *Handler)GetOrders() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var userIDCtx int
		if id := r.Context().Value(UserIDCtxName); id != nil {
			userIDCtx = id.(int)
		}

		orders, err := h.services.Users.GetOrders(r.Context(), userIDCtx)
		if err != nil {
			switch {
			case errors.Is(err, domain.ErrOrdersNotFound):
				http.Error(w, err.Error(), http.StatusNoContent)
				return
			default:

				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		result, err := json.Marshal(orders)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("content-type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, string(result))
	}
}

//get the current account balance of the user's loyalty points
func (h *Handler)GetBalance() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var userIDCtx int
		if id := r.Context().Value(UserIDCtxName); id != nil {
			userIDCtx = id.(int)
		}

		balance, err := h.services.Users.GetBalance(r.Context(), userIDCtx)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		result, err := json.Marshal(balance)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("content-type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, string(result))
	}
}

type withdrawInput struct {
	Order    string `json:"order"`
	Sum 	 float32 `json:"sum"`
}

//request to withdraw points from the account to pay for a new order
func (h *Handler)PostWithdraw() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var userIDCtx int
		if id := r.Context().Value(UserIDCtxName); id != nil {
			userIDCtx = id.(int)
		}

		inp := withdrawInput{}
		if err := json.NewDecoder(r.Body).Decode(&inp); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest) //400
			return
		}

		if !luhn.Valid(inp.Order) {
			http.Error(w,"invalid order number format", http.StatusUnprocessableEntity)
			return
		}

		err := h.services.Users.Withdraw(r.Context(), userIDCtx, service.UserWithdrawInput {
			Order:  inp.Order,
			Sum:    inp.Sum,
		})

		if err != nil {
			if errors.Is(err, domain.ErrWithdrawalInsufficientFunds) {
				http.Error(w, err.Error(), http.StatusPaymentRequired)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

//getting information about the withdrawal of funds from the savings account by the user.
func (h *Handler)GetWithdrawals() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var userIDCtx int
		if id := r.Context().Value(UserIDCtxName); id != nil {
			userIDCtx = id.(int)
		}

		withdrawals, err := h.services.Users.GetUserWithdrawals(r.Context(), userIDCtx)
		if err != nil {
			switch {
			case errors.Is(err, domain.ErrWithdrawalNotFound):
				http.Error(w, err.Error(), http.StatusNoContent)
				return
			default:
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		result, err := json.Marshal(withdrawals)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("content-type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, string(result))

	}
}