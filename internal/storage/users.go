package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/jackc/pgerrcode"
	"github.com/lib/pq"
	"gophermart/internal/domain"
	"sync"
	"time"
)

type UserStorage struct {
	mu  sync.Mutex
	db *sql.DB
}

func NewUserStorage(db *sql.DB) *UserStorage {
	return &UserStorage{
		db: db,
	}
}

func (r *UserStorage) Create(ctx context.Context, user domain.User) error {
	crUserStmt, err := r.db.PrepareContext(ctx, "INSERT INTO users (login, password) VALUES ($1, $2) RETURNING id;")
	if err != nil {
		return &StatementPSQLError{Err: err}
	}
	defer crUserStmt.Close()

	r.mu.Lock()
	defer r.mu.Unlock()
	if err := crUserStmt.QueryRowContext(ctx, user.Login, user.Password).Scan(&user.ID); err != nil {
		errCode := err.(*pq.Error).Code
		if pgerrcode.IsIntegrityConstraintViolation(string(errCode)) {
			return &AlreadyExistsError{Err: domain.ErrUserAlreadyExists}
		}
		return &ExecutionPSQLError{Err: err}
	}

	return nil
}

func (r *UserStorage) GetByCredentials(ctx context.Context, login, password string) (domain.User, error) {
	user := domain.User{}

	getUserStmt, err := r.db.PrepareContext(ctx, "SELECT id,password FROM users WHERE login=$1;")
	if err != nil {
		return user, &StatementPSQLError{Err: err}
	}
	defer getUserStmt.Close()

	r.mu.Lock()
	if err := getUserStmt.QueryRowContext(ctx, login).Scan(&user.ID, &user.Password); err != nil {
		r.mu.Unlock()
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return user, &NotFoundError{Err: domain.ErrUserNotFound}
		default:
			return user, &ExecutionPSQLError{Err: err}
		}
	}

	r.mu.Unlock()

	if user.Password == password {
		user.Login = login
		return user, nil
	} else {
		return user, domain.ErrUserBadPassword
	}
}

func (r *UserStorage) SetSession(ctx context.Context, userID int, session domain.Session) error  {
	//TODO
	return nil
}

func (r *UserStorage) GetOrderByNumber(ctx context.Context, orderNumber string) (domain.Order, error) {
	order := domain.Order{}

	getOrderStmt, err := r.db.PrepareContext(ctx, "SELECT user_id,status,accrual,uploaded_at FROM orders WHERE number=$1;")
	if err != nil {
		return order, &StatementPSQLError{Err: err}
	}
	defer getOrderStmt.Close()


	r.mu.Lock()
	if err := getOrderStmt.QueryRowContext(ctx, orderNumber).Scan(&order.UserID, &order.Status, &order.Accrual, &order.UploadedAt); err != nil {
		r.mu.Unlock()
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return order, &NotFoundError{Err: domain.ErrUserNotFound}
		default:
			return order, &ExecutionPSQLError{Err: err}
		}
	}

	r.mu.Unlock()

	return order, nil
}

func (r *UserStorage) GetOrdersByUser(ctx context.Context, userID int) ([]domain.Order, error) {
	getOrdersStmt, err := r.db.PrepareContext(ctx, "SELECT number,status,accrual,uploaded_at FROM orders WHERE user_id=$1 ORDER BY uploaded_at DESC;")
	if err != nil {
		return nil, &StatementPSQLError{Err: err}
	}
	defer getOrdersStmt.Close()

	r.mu.Lock()
	defer r.mu.Unlock()

	rows, err := getOrdersStmt.QueryContext(ctx, userID)
	if err != nil {
		return nil, &ExecutionPSQLError{Err: err}
	}
	defer rows.Close()

	orders := make([]domain.Order,0)
	for rows.Next() {
		var order domain.Order
		err = rows.Scan(&order.Number, &order.Status, &order.Accrual, &order.UploadedAt)
		if err != nil {
			switch {
			case errors.Is(err, sql.ErrNoRows):
				return nil, &NotFoundError{Err: domain.ErrOrdersNotFound}
			default:
				return nil, &ExecutionPSQLError{Err: err}
			}
		}
		order.UserID = userID

		orders = append(orders, order)
	}

	err = rows.Err()
	if err != nil {
		return nil, &ExecutionPSQLError{Err: err}
	}

	return orders, nil
}

func (r *UserStorage) GetOrdersByStatus(ctx context.Context, orderStatuses []domain.OrderStatus)([]domain.Order, error) {
	getOrdersStmt, err := r.db.PrepareContext(ctx, "SELECT number,user_id,status,accrual,uploaded_at FROM orders WHERE status=ANY($1);")
	if err != nil {
		return nil, &StatementPSQLError{Err: err}
	}
	defer getOrdersStmt.Close()

	r.mu.Lock()
	defer r.mu.Unlock()

	rows, err := getOrdersStmt.QueryContext(ctx, pq.Array(orderStatuses))
	if err != nil {
		return nil, &ExecutionPSQLError{Err: err}
	}
	defer rows.Close()

	orders := make([]domain.Order,0)
	for rows.Next() {
		var order domain.Order
		err = rows.Scan(&order.Number, &order.UserID, &order.Status, &order.Accrual, &order.UploadedAt)
		if err != nil {
			switch {
			case errors.Is(err, sql.ErrNoRows):
				return nil, &NotFoundError{Err: domain.ErrOrdersNotFound}
			default:
				return nil, &ExecutionPSQLError{Err: err}
			}
		}

		orders = append(orders, order)
	}

	err = rows.Err()
	if err != nil {
		return nil, &ExecutionPSQLError{Err: err}
	}

	return orders, nil
}

func (r *UserStorage) AddOrder(ctx context.Context, order domain.Order) error {
	//check if user exists
	checkUserStmt, err := r.db.PrepareContext(ctx, "SELECT id, current FROM users WHERE id = $1;")
	if err != nil {
		return &StatementPSQLError{Err: err}
	}
	defer checkUserStmt.Close()

	var userID int
	var userCurrent float32

	r.mu.Lock()
	defer r.mu.Unlock()

	if err := checkUserStmt.QueryRowContext(ctx, order.UserID).Scan(&userID, &userCurrent); err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return &NotFoundError{Err: domain.ErrUserNotFound}
		default:
			return &ExecutionPSQLError{Err: err}
		}
	}

	//we need to add order and if its status is final, add accrual to his loyalty points

	//begin transaction
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return &ExecutionPSQLError{Err: err}
	}
	defer tx.Rollback()

	addOrderStmt, err := r.db.PrepareContext(ctx, "INSERT INTO orders (number, user_id, status, accrual, uploaded_at) VALUES ($1, $2, $3, $4, $5);")
	if err != nil {
		return &StatementPSQLError{Err: err}
	}
	defer addOrderStmt.Close()

	txAddOrderStmt := tx.StmtContext(ctx, addOrderStmt)
	defer txAddOrderStmt.Close()

	_, err = txAddOrderStmt.ExecContext(
		ctx,
		order.Number,
		order.UserID,
		order.Status,
		order.Accrual,
		time.Time(order.UploadedAt),
		)
	if err != nil {
		fmt.Println(err)
		errCode := err.(*pq.Error).Code
		if pgerrcode.IsIntegrityConstraintViolation(string(errCode)) {
			return &AlreadyExistsError{Err: domain.ErrOrderAlreadyExists}
		}
		return &ExecutionPSQLError{Err: err}
	}

	//add accrual to his loyalty points
	if order.Status == domain.OrderStatusProcessed {
		userCurrent = userCurrent + order.Accrual
		updateAccrualStmt, err := r.db.PrepareContext(ctx, "UPDATE users SET current = $1 WHERE id = $2;")
		if err != nil {
			return &StatementPSQLError{Err: err}
		}
		defer addOrderStmt.Close()

		txUpdateAccrualStmt := tx.StmtContext(ctx, updateAccrualStmt)
		defer txUpdateAccrualStmt.Close()

		_, err = txUpdateAccrualStmt.ExecContext(ctx, userCurrent, userID)
		if err != nil {
			return &ExecutionPSQLError{Err: err}
		}
	}

	err = tx.Commit()
	if err != nil {
		return &ExecutionPSQLError{Err: err}
	}

	return nil
}

func (r *UserStorage) UpdateOrders(ctx context.Context, orders []domain.Order) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// begin transaction
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return &ExecutionPSQLError{Err: err}
	}
	defer tx.Rollback()

	//statements to update orders
	setOrderStmt, err := r.db.PrepareContext(ctx, "UPDATE orders SET user_id = $1, status = $2, accrual = $3, uploaded_at = $4 WHERE number = $5;")
	if err != nil {
		return &StatementPSQLError{Err: err}
	}
	defer setOrderStmt.Close()

	txSetOrderStmtStmt := tx.StmtContext(ctx, setOrderStmt)
	defer txSetOrderStmtStmt.Close()

	//statements to update current
	updateAccrualStmt, err := r.db.PrepareContext(ctx, "UPDATE users SET current = $1 WHERE id = $2;")
	if err != nil {
		return &StatementPSQLError{Err: err}
	}
	defer updateAccrualStmt.Close()

	txUpdateAccrualStmt := tx.StmtContext(ctx, updateAccrualStmt)
	defer txUpdateAccrualStmt.Close()

	for _, order := range orders {
		_, err = txSetOrderStmtStmt.ExecContext(
			ctx,
			order.UserID,
			order.Status,
			order.Accrual,
			time.Time(order.UploadedAt),
			order.Number,
		)
		if err != nil {
			return &ExecutionPSQLError{Err: err}
		}

		if order.Status == domain.OrderStatusProcessed {
			var userCurrent float32
			if err = tx.QueryRowContext(ctx, "SELECT current FROM users WHERE id = $1;", order.UserID).Scan(&userCurrent); err != nil {
				switch {
				case errors.Is(err, sql.ErrNoRows):
					return &NotFoundError{Err: domain.ErrUserNotFound}
				default:
					return &ExecutionPSQLError{Err: err}
				}
			}

			userCurrent = userCurrent + order.Accrual

			_, err = txUpdateAccrualStmt.ExecContext(ctx, userCurrent, order.UserID)
			if err != nil {
				return &ExecutionPSQLError{Err: err}
			}
		}
	}

	err = tx.Commit()
	if err != nil {
		return &ExecutionPSQLError{Err: err}
	}

	return nil
}

func (r *UserStorage) GetUserBalance(ctx context.Context, userID int) (domain.Balance, error) {
	balance := domain.Balance{}

	getBalanceStmt, err := r.db.PrepareContext(ctx, "SELECT current,withdrawn FROM users WHERE id=$1;")
	if err != nil {
		return balance, &StatementPSQLError{Err: err}
	}
	defer getBalanceStmt.Close()


	r.mu.Lock()
	if err := getBalanceStmt.QueryRowContext(ctx, userID).Scan(&balance.Current, &balance.Withdrawn); err != nil {
		r.mu.Unlock()
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return balance, &NotFoundError{Err: domain.ErrUserNotFound}
		default:
			return balance, &ExecutionPSQLError{Err: err}
		}
	}

	r.mu.Unlock()

	return balance, nil
}

func (r *UserStorage) GetUserWithdrawals(ctx context.Context, userID int) ([]domain.Withdrawal, error) {
	getWithdrawalsStmt, err := r.db.PrepareContext(ctx, "SELECT order_number,sum,processed_at FROM withdrawals WHERE user_id=$1 ORDER BY processed_at DESC;")
	if err != nil {
		return nil, &StatementPSQLError{Err: err}
	}
	defer getWithdrawalsStmt.Close()

	r.mu.Lock()
	defer r.mu.Unlock()

	rows, err := getWithdrawalsStmt.QueryContext(ctx, userID)
	if err != nil {
		return nil, &ExecutionPSQLError{Err: err}
	}
	defer rows.Close()

	withdrawals := make([]domain.Withdrawal,0)
	for rows.Next() {
		var withdrawal domain.Withdrawal
		err = rows.Scan(&withdrawal.Order, &withdrawal.Sum, &withdrawal.ProcessedAt)
		if err != nil {
			switch {
			case errors.Is(err, sql.ErrNoRows):
				return nil, &NotFoundError{Err: domain.ErrWithdrawalNotFound}
			default:
				return nil, &ExecutionPSQLError{Err: err}
			}
		}
		withdrawal.UserID = userID

		withdrawals = append(withdrawals, withdrawal)
	}

	err = rows.Err()
	if err != nil {
		return nil, &ExecutionPSQLError{Err: err}
	}

	return withdrawals, nil
}

func (r *UserStorage) AddWithdrawal(ctx context.Context, withdrawal domain.Withdrawal) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	//statements to add withdrawal
	addWithdrawalStmt, err := r.db.PrepareContext(ctx, "INSERT INTO withdrawals (user_id,order_number,sum,processed_at) VALUES ($1,$2,$3,$4);")
	if err != nil {
		return &StatementPSQLError{Err: err}
	}
	defer addWithdrawalStmt.Close()

	//statements to update accounts
	updateAccountsStmt, err := r.db.PrepareContext(ctx, "UPDATE users SET current = current - $1, withdrawn = withdrawn + $1 WHERE id = $2;")
	if err != nil {
		return &StatementPSQLError{Err: err}
	}
	defer updateAccountsStmt.Close()

	// begin transaction
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return &ExecutionPSQLError{Err: err}
	}
	defer tx.Rollback()

	//add withdrawal
	txAddWithdrawalStmt := tx.StmtContext(ctx, addWithdrawalStmt)
	defer txAddWithdrawalStmt.Close()

	_, err = txAddWithdrawalStmt.ExecContext(
		ctx,
		withdrawal.UserID,
		withdrawal.Order,
		withdrawal.Sum,
		time.Time(withdrawal.ProcessedAt),
	)
	if err != nil {
		errCode := err.(*pq.Error).Code
		if pgerrcode.IsIntegrityConstraintViolation(string(errCode)) {
			return &AlreadyExistsError{Err: domain.ErrWithdrawalAlreadyExists}
		}
		return &ExecutionPSQLError{Err: err}
	}

	//update accounts
	txUpdateAccountsStmt := tx.StmtContext(ctx, updateAccountsStmt)
	defer txUpdateAccountsStmt.Close()

	_, err = txUpdateAccountsStmt.ExecContext(ctx, withdrawal.Sum, withdrawal.UserID)
	if err != nil {
		return &ExecutionPSQLError{Err: err}
	}

	err = tx.Commit()
	if err != nil {
		return &ExecutionPSQLError{Err: err}
	}

	return nil
}
