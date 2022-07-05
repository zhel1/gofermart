package service

import (
	"context"
	"errors"
	"gophermart/internal/auth"
	"gophermart/internal/domain"
	"gophermart/internal/hash"
	"gophermart/internal/storage"
	"strconv"
	"time"
)

type UserService struct {
	hasher      	hash.PasswordHasher
	storage 		storage.Users
	tokenManager 	auth.TokenManager
	accessTokenTTL	time.Duration
}

func NewUserService(h hash.PasswordHasher, s storage.Users, tm auth.TokenManager, at time.Duration) *UserService {
	return &UserService{
		hasher: h,
		storage: s,
		tokenManager: tm,
		accessTokenTTL:	at,
	}
}

func (s *UserService) SignUp(ctx context.Context, input UserSignUpInput) error {
	passwordHash, err := s.hasher.Hash(input.Password)
	if err != nil {
		return err
	}

	user := domain.User{
		Login: input.Login,
		Password: passwordHash,
	}

	if err := s.storage.Create(ctx, user); err != nil {
		return err
	}
	return nil
}

func (s *UserService) SignIn(ctx context.Context, input UserSignInInput) (Token, error) {
	passwordHash, err := s.hasher.Hash(input.Password)
	if err != nil {
		return Token{}, err
	}

	user, err := s.storage.GetByCredentials(ctx, input.Login, passwordHash)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return Token{}, err
		}

		return Token{}, err
	}

	return s.createSession(ctx, user.ID)
}

func (s *UserService) AddOrder(ctx context.Context, order domain.Order) error {
	err := s.storage.AddOrder(ctx, order)
	if err != nil {
		switch  {
		case errors.Is(err, domain.ErrOrderAlreadyExists):
			//we need to figure out the owner of the order
			existedOrder, err := s.storage.GetOrderByNumber(ctx, order.Number)
			if err != nil {
				return err
			}

			if order.UserID == existedOrder.UserID {
				return domain.ErrRepeatedOrderRequest
			} else {
				return domain.ErrForeignOrder
			}
		default:
			return err
		}
	}
	return nil
}

func (s *UserService) GetOrders(ctx context.Context, userID int) ([]domain.Order, error) {
	return s.storage.GetOrdersByUser(ctx, userID)
}

func (s *UserService) GetBalance(ctx context.Context, userID int) (domain.Balance, error) {
	return s.storage.GetUserBalance(ctx, userID)
}

func (s *UserService) GetUserWithdrawals(ctx context.Context, userID int) ([]domain.Withdrawal, error) {
	return s.storage.GetUserWithdrawals(ctx, userID)
}

func (s *UserService) Withdraw(ctx context.Context, userID int, input UserWithdrawInput) error {
	balance, err := s.storage.GetUserBalance(ctx, userID)
	if err != nil {
		return err
	}

	if balance.Current < input.Sum {
		return domain.ErrWithdrawalInsufficientFunds
	}

	err = s.storage.AddWithdrawal(ctx,domain.Withdrawal{
		UserID: userID,
		Order: input.Order,
		Sum: input.Sum,
		ProcessedAt: domain.Time(time.Now()),
	})

	if err != nil {
		switch {
		case errors.Is(err, domain.ErrWithdrawalAlreadyExists):
			return err
		default:
			return err
		}
	}

	return nil
}

func (s *UserService) createSession(ctx context.Context, userID int) (Token, error) {
	var (
		res Token
		err error
	)

	res.AccessToken, err = s.tokenManager.NewJWT(strconv.Itoa(userID), s.accessTokenTTL)
	if err != nil {
		return res, err
	}

	session := domain.Session{
		ExpiresAt:    time.Now().Add(s.accessTokenTTL),
	}

	err = s.storage.SetSession(ctx, userID, session)

	return res, err
}