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

			r.Route("/balance", func(r chi.Router) {
				r.Get("/", h.GetBalance())
				r.Post("/withdraw", h.PostWithdraw())
				r.Get("/withdrawals", h.GetWithdrawals())
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
		fmt.Println("PostRegister")
		inp := signInInput{}
		if err := json.NewDecoder(r.Body).Decode(&inp); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest) //400
			return
		}

		err := h.services.Users.SignUp(r.Context(), service.UserSignUpInput {
			Login:    inp.Login,
			Password: inp.Password,
		})

		if err != nil {
			if errors.Is(err, domain.ErrUserAlreadyExists) {
				http.Error(w, err.Error(), http.StatusConflict) //409
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError) //500
			return
		}

		//authentication
		token, err := h.services.Users.SignIn(r.Context(), service.UserSignInInput {
			Login:    inp.Login,
			Password: inp.Password,
		})

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError) //500
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
		fmt.Println("PostLogin")
		inp := signUpInput{}
		if err := json.NewDecoder(r.Body).Decode(&inp); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest) //400
			return
		}

		token, err := h.services.Users.SignIn(r.Context(), service.UserSignInInput {
			Login:    inp.Login,
			Password: inp.Password,
		})

		if err != nil {
			if errors.Is(err, domain.ErrUserNotFound) {
				http.Error(w, err.Error(), http.StatusUnauthorized) //401
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError) //500
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
		fmt.Println("PostOrders")
		orderID, err := io.ReadAll(r.Body)
		if err != nil {
			//400 — неверный формат запроса;
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		orderIDStr := strings.TrimSpace(string(orderID))

		if !luhn.Valid(orderIDStr) {
			//422 — неверный формат номера заказа;
			http.Error(w,"invalid order number format", http.StatusUnprocessableEntity)
			return
		}

		var userIDCtx int
		if id := r.Context().Value(userCtx); id != nil {
			userIDCtx = id.(int)
		}

		order := domain.Order{
			Number: orderIDStr,
			UserID: userIDCtx,
			Status:domain.OrderStatusUnknown,
			Accrual: 0,
			UploadedAt: time.Now(),
		}

		err = h.services.Users.AddOrder(r.Context(), order)
		if err != nil {
			switch {
			case errors.Is(err, domain.ErrRepeatedOrderRequest):
				//200 — номер заказа уже был загружен этим пользователем;
				http.Error(w, err.Error(), http.StatusUnprocessableEntity)
				return
			case errors.Is(err, domain.ErrForeignOrder):
				//409 — номер заказа уже был загружен другим пользователем;
				http.Error(w, err.Error(), http.StatusConflict)
				return
			default:
				//500 — внутренняя ошибка сервера.
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		h.services.Updater.AddOrder(order)

		//202 — новый номер заказа принят в обработку;
		w.WriteHeader(http.StatusAccepted)
	}
}

//get a list of order numbers uploaded by the user, their processing statuses and information about accruals
func (h *Handler)GetOrders() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("GetOrders")
	}
}

//get the current account balance of the user's loyalty points
func (h *Handler)GetBalance() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("GetBalance")
	}
}

//request to withdraw points from the account to pay for a new order
func (h *Handler)PostWithdraw() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("PostWithdraw")
	}
}

//getting information about the withdrawal of funds from the savings account by the user.
func (h *Handler)GetWithdrawals() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("GetWithdrawals")
	}
}