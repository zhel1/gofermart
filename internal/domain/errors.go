package domain

import "errors"

//user
var (
	ErrUserNotFound 						= errors.New("user doesn't exists")
	ErrUserAlreadyExists       				= errors.New("user with such login already exists")
	ErrUserBadPassword		 				= errors.New("bad password")

	ErrOrdersNotFound			 			= errors.New("orders were not found")
	ErrOrderAlreadyExists			 		= errors.New("order was already added")

	ErrRepeatedOrderRequest					= errors.New("order was already added by you")
	ErrForeignOrder					 		= errors.New("order was already added by another user")

	ErrWithdrawalNotFound          			= errors.New("withdrawal were not found")
	ErrWithdrawalAlreadyExists     			= errors.New("withdrawal was already added")
	ErrWithdrawalInsufficientFunds 			= errors.New("insufficient funds to withdraw")
)

//accrual
var (
	ErrAccrualRequestError					= errors.New("failed to request accrual service")

	ErrAccrualTooManyRequests 				= errors.New("number of requests to accrual the service has been exceeded")
	ErrAccrualNoContent 					= errors.New("unknown order")
	ErrAccrualInternalServerError			= errors.New("internal accrual server error")

	ErrAccrualBadRequest					= errors.New("bad request to accrual service")
	ErrAccrualOrderAlreadyAccepted			= errors.New("order in accrual server has already been accepted")
	ErrAccrualSearchKeyAlreadyRegistered	= errors.New("search key in accrual server is already registered")
)