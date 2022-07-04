package service

import (
	"context"
	"fmt"
	"gophermart/internal/domain"
	"gophermart/internal/storage"
	"log"
	"sync"
	"time"
)

const (
	askAccrualServicePeriod = 1 * time.Second
	ordersBunchSize = 5
)

type UpdaterService struct {
	accrual Accrual
	storage storage.Users

	orders []domain.Order

	ctx context.Context
	cancel context.CancelFunc
	mu sync.Mutex
}

func NewUpdaterService(accrual Accrual, storage storage.Users) *UpdaterService {
	us := &UpdaterService{
		accrual: accrual,
		storage: storage,
		orders: make([]domain.Order,0),
	}
	us.Start()
	return us
}

func (u *UpdaterService)AddOrder(order domain.Order) {
	if u.ctx == nil {
		log.Fatal("start updater service before using AddOrder function")
	}

	u.mu.Lock()
	u.orders = append(u.orders, order)
	u.mu.Unlock()
}

func (u *UpdaterService)AddOrders(orders []domain.Order) {
	if u.ctx == nil {
		log.Fatal("start updater service before using AddOrder function")
	}

	u.mu.Lock()
	u.orders = append(u.orders, orders...)
	u.mu.Unlock()
}

func (u *UpdaterService) Start() <-chan error {
	u.ctx, u.cancel = context.WithCancel(context.Background())
	errc := make(chan error)
	go func() {
		defer close(errc)
		if err := u.run(); err != nil {
			fmt.Println(err.Error())
			errc <- err
		}
	}()

	return errc
}

func (u *UpdaterService) Stop() {
	u.cancel()
}

func (u *UpdaterService)run() error {
	ordersWithNewStatus := make(chan domain.Order)
	askAccrualServiceError := make(chan error)
	databaseError := make(chan error)

	askAccrualServiceTicker := time.NewTicker(askAccrualServicePeriod) //can be longer then period
	updateInDatabaseTicker := time.NewTicker(askAccrualServicePeriod)

	//orders's status updater
	go func() {
		for {
			select {
			case <-u.ctx.Done():
				return
			case <- askAccrualServiceTicker.C:
				fmt.Println("askAccrualServiceTicker")
				//опрашиваем сервис на предмет обработки заказов из кеша
				u.mu.Lock()
				reducedOrders := make([]domain.Order,0)
				for i, order := range u.orders {
					accrual, err := u.accrual.GetAccrualByOrderID(u.ctx, order.Number)
					if err != nil {
						switch err {
						case domain.ErrAccrualTooManyRequests:
							time.Sleep(accrual.RetryAfter)  //wait and left this order for the next time
						case domain.ErrAccrualNoContent:
							//TODO User sent order, which he didn't do? Remove it from database?
							//TODO Or maybe user sent order before accrual service found out about it?
							//keep monitor
						default:
							askAccrualServiceError <- err
						}
						continue
					}

					//if status was updated
					if accrual.Status.String() != order.Status.String() { //TODO bad solution, not reliable
						switch accrual.Status {
						case AccrualStatusRegistered:
							u.orders[i].Status = domain.OrderStatusRegistered
						case AccrualStatusInvalid: //final status
							u.orders[i].Status = domain.OrderStatusInvalid
						case AccrualStatusProcessing:
							u.orders[i].Status = domain.OrderStatusProcessing
						case AccrualStatusProcessed: //final status
							u.orders[i].Status = domain.OrderStatusProcessed
							u.orders[i].Accrual = accrual.Accrual
						default:
							u.orders[i].Status = domain.OrderStatusUnknown
							askAccrualServiceError <- fmt.Errorf("unknown accrual status")
						}

						ordersWithNewStatus <- u.orders[i] //copy will be sent, don't be afraid
					}

					if u.orders[i].Status != domain.OrderStatusProcessed && u.orders[i].Status != domain.OrderStatusInvalid { //final statuses
						reducedOrders = append(reducedOrders, u.orders[i])
					}
				}
				u.orders = reducedOrders
				u.mu.Unlock()
			}
		}
	}()

	//database updater
	go func() {
		parts := make([]domain.Order,0,ordersBunchSize)

		pushInDB := func() {
			err := u.storage.UpdateOrders(u.ctx, parts)
			if err != nil {
				databaseError <- err
			}
			parts = make([]domain.Order,0,ordersBunchSize)
		}

		for {
			select {
			case <-u.ctx.Done():
				return
			case <-updateInDatabaseTicker.C:
				log.Println("Save due to timeout")
				pushInDB()
			case order, ok := <-ordersWithNewStatus:
				if !ok { // if chanel was closed
					return
				}
				parts = append(parts, order)
				if len(parts) >= ordersBunchSize {
					log.Println("Save due to exceeding capacity")
					pushInDB()
				}
			}
		}
	}()

	// wait for the first channel to retrieve a value
	select {
	case <-u.ctx.Done():
		return fmt.Errorf("ubdater was canseled") //TODO define error
	case err := <-askAccrualServiceError:
		return fmt.Errorf("error during convercation with accrual service: %w", err)
	case err := <-databaseError:
		return err
		//return fmt.Errorf("DB error: %w", err)
	}

}