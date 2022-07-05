package v1

import (
	"context"
	"errors"
	"net/http"
	"strconv"
)

type CookieConst string

func (c CookieConst) String() string {
	return string(c)
}

var (
	UserIDCtxName CookieConst = "UserID"
)

//**********************************************************************************************************************
//checks that user is authorised and puts his id into context
func (h *Handler) checkUserIdentity(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenCookie, err := r.Cookie("AccessToken") //TODO make constant
		if errors.Is(err, http.ErrNoCookie) { //no cookie
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		} else if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		} else { //cookie found
			userID, err := h.tokenManager.Parse(tokenCookie.Value)
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}

			userIDInt, err := strconv.Atoi(userID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), UserIDCtxName, userIDInt)))
		}
	})
}