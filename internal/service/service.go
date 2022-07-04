package service

import (
	"context"
	"errors"
	"gophermart/internal/auth"
	"gophermart/internal/domain"
	"gophermart/internal/hash"
	"gophermart/internal/storage"
	"log"
	"time"
)

type UserSignUpInput struct {
	Login        string
	Password     string
}

type UserSignInInput struct {
	Login        string
	Password     string
}

type Token struct {
	AccessToken  string
}

type Users interface {
	SignUp(ctx context.Context, input UserSignUpInput) error
	SignIn(ctx context.Context, input UserSignInInput) (Token, error)
	AddOrder(ctx context.Context, order domain.Order) error
}

//**********************************************************************************************************************
type AccrualStatus string

const (
	AccrualStatusRegistered AccrualStatus 	= "REGISTERED"	// order is registered, but no accrual is calculated
	AccrualStatusInvalid AccrualStatus 		= "INVALID"		// order was not accepted for settlement, and the reward will not be credited
	AccrualStatusProcessing AccrualStatus 	= "PROCESSING"	// calculation of the accrual in the process
	AccrualStatusProcessed AccrualStatus 	= "PROCESSED"	// calculation is completed
)

func (r AccrualStatus) String() string {
	return string(r)
}

func (r *AccrualStatus) MarshalJSON() ([]byte, error) {
	return []byte(`"` + r.String() + `"`), nil
}

func (r *AccrualStatus) UnmarshalJSON(data []byte) error {
	*r = AccrualStatus(data[1 : len(data)-1])
	return nil
}

type Good struct {
	Description string	`json:"description"`
	Price int			`json:"price"`
}

type Order struct {
	OrderID string			`json:"order"`
	Goods []Good			`json:"goods"`
}

type AccrualOutput struct {
	OrderID string				`json:"order"`
	Status AccrualStatus		`json:"status"`
	Accrual float32				`json:"accrual,omitempty"`
	RetryAfter time.Duration	`json:"-"`
}

type RewardType string

const (
	RewardTypePercents  RewardType = "%"
	RewardTypePunctual  RewardType = "pt"
)

func (r RewardType) String() string {
	return string(r)
}

func (r *RewardType) MarshalJSON() ([]byte, error) {
	return []byte(`"` + r.String() + `"`), nil
}
func (r *RewardType) UnmarshalJSON(data []byte) error {
	*r = RewardType(data[1 : len(data)-1])
	return nil
}

type AccrualRule struct {
	Match string			`json:"match"`
	Reward int				`json:"reward"`
	RewardType RewardType	`json:"reward_type"`
}

type Accrual interface {
	GetAccrualByOrderID(ctx context.Context, orderID string) (AccrualOutput, error)
	RegisterAccrualRule(ctx context.Context, rule AccrualRule) error
	RegisterOrders(ctx context.Context, order Order) error
}
//**********************************************************************************************************************
type Updater interface {
	AddOrder(order domain.Order)
	AddOrders(order []domain.Order)
}
//**********************************************************************************************************************
type Services struct {
	Users Users
	Accrual Accrual
	Updater Updater
}

type Deps struct {
	Storages               *storage.Storages
	Hasher                 hash.PasswordHasher
	TokenManager           auth.TokenManager
	AccessTokenTTL         time.Duration
	AccrualAddress 		   string
}

func NewServices(deps Deps) *Services {
	statuses := []domain.OrderStatus{domain.OrderStatusRegistered, domain.OrderStatusProcessing, domain.OrderStatusUnknown}
	orders, err := deps.Storages.Users.GetOrdersByStatus(context.Background(), statuses)
	if err != nil && !errors.Is(err, domain.ErrOrdersNotFound) {
		log.Fatal(err.Error())
	}

	users := NewUserService(deps.Hasher, deps.Storages.Users, deps.TokenManager, deps.AccessTokenTTL)
	accrual := NewAccrualService(deps.AccrualAddress)
	updaterService := NewUpdaterService(accrual, deps.Storages.Users)
	updaterService.AddOrders(orders)

	return &Services {
		Users: users,
		Accrual: accrual,
		Updater: updaterService,
	}
}