package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"gophermart/internal/domain"
	"io"
	"net/http"
	"strconv"
	"time"
)

type AccrualService struct {
	addr string
	client *http.Client
}

func NewAccrualService(addr string) *AccrualService {
	return &AccrualService{
		addr: addr,
		client: &http.Client{},
	}
}

//GET /api/orders/{number} - getting information on the calculation of accruals
func (a *AccrualService)GetAccrualByOrderID(ctx context.Context, orderID string) (AccrualOutput, error) {
	result := AccrualOutput{}
	endpoint := a.addr + "/api/orders/" + orderID
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return result, err
	}

	response, err := a.client.Do(request)
	if err != nil {
		return result, err
	}
	defer response.Body.Close()

	switch response.StatusCode {
	case http.StatusOK:
		body, err := io.ReadAll(response.Body)
		if err != nil {
			return result, err
		}
		if err = json.Unmarshal(body, &result); err != nil {
			return result, err
		}
		return result, nil
	case http.StatusTooManyRequests:
		secStr := response.Header.Get("Retry-After")
		sec, err := strconv.Atoi(secStr)
		if err != nil {
			return result, domain.ErrAccrualRequestError
		}
		result.RetryAfter = time.Duration(sec) * time.Second
		return result, domain.ErrAccrualTooManyRequests
	case http.StatusNoContent:
		return result, domain.ErrAccrualNoContent
	case http.StatusInternalServerError:
		return result, domain.ErrAccrualInternalServerError
	default:
		return result, domain.ErrAccrualRequestError
	}
}

//POST /api/goods - registration of information about the new reward mechanics for goods.
func (a *AccrualService)RegisterOrders(ctx context.Context, order Order) error {
	jsonOrders, err := json.Marshal(order)

	fmt.Println("json orders: " + string(jsonOrders))

	if err != nil {
		return err
	}

	endpoint := a.addr + "/api/orders"
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBufferString(string(jsonOrders)))
	if err != nil {
		return err
	}
	request.Header.Add("Content-Type", "application/json")
	response, err := a.client.Do(request)
	if err != nil  {
		return err
	}
	defer response.Body.Close()

	switch response.StatusCode {
	case http.StatusAccepted:
		return nil
	case http.StatusBadRequest:
		return domain.ErrAccrualBadRequest
	case http.StatusConflict:
		return domain.ErrAccrualOrderAlreadyAccepted
	case http.StatusInternalServerError:
		return domain.ErrAccrualInternalServerError
	default:
		return domain.ErrAccrualRequestError
	}
}

//POST /api/orders - registration of a new completed order;
func (a *AccrualService)RegisterAccrualRule(ctx context.Context, rule AccrualRule) error {
	jsonRule, err := json.Marshal(rule)
	if err != nil {
		return err
	}

	fmt.Println("json rule: " + string(jsonRule))

	endpoint := a.addr + "/api/goods"
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBufferString(string(jsonRule)))
	if err != nil {
		return err
	}
	request.Header.Add("Content-Type", "application/json")
	response, err := a.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	switch response.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusBadRequest:
		return domain.ErrAccrualBadRequest
	case http.StatusConflict:
		return domain.ErrAccrualSearchKeyAlreadyRegistered
	case http.StatusInternalServerError:
		return domain.ErrAccrualInternalServerError
	default:
		return domain.ErrAccrualRequestError
	}
}

//TODO move in unit test
func (a *AccrualService)FillWithTestData(ctx context.Context) error {
	rules := []AccrualRule{
		{
			Match: "Bork",
			Reward: 10,
			RewardType: RewardTypePercents,
		},
		{
			Match: "Indesit",
			Reward: 5,
			RewardType: RewardTypePunctual,
		},
		{
			Match: "Bosh",
			Reward: 15,
			RewardType: RewardTypePercents,
		},
	}

	for _, rule := range rules {
		if err := a.RegisterAccrualRule(context.Background(), rule); err != nil {
			fmt.Println("Register accrual rule error: ", err.Error())
			return err
		}
		time.Sleep(100 * time.Millisecond)
	}

	orders := []Order{
		{
			OrderID: "3004",
			Goods: []Good {
				{
					Description: "Чайник Bork",
					Price: 7000,
				},
				{
					Description: "Утюг Bork",
					Price: 5000,
				},
				{
					Description: "Холодильник Indesit",
					Price: 20000,
				},
			},
		},
		{
			OrderID: "30049",
			Goods: []Good {
				{
					Description: "Пылесос Bork",
					Price: 8000,
				},
				{
					Description: "Печь Bosh",
					Price: 9000,
				},
			},
		},
		{
			OrderID: "300491",
			Goods: []Good {
				{
					Description: "Плита Bosh",
					Price: 23000,
				},
			},
		},
		{
			OrderID: "3004918",
			Goods: []Good {
				{
					Description: "Мультиварка Indesit",
					Price: 11000,
				},
			},
		},
	}

	for _, order := range orders {
		if err := a.RegisterOrders(context.Background(), order); err != nil {
			fmt.Println("Register orders error: ", err.Error())
			return err
		}
		time.Sleep(100 * time.Millisecond)
	}

	/*
	orderNumbers := []string {
		"3001",
		"3002",
		"3003",
		"3004",
	}

	for _, n := range orderNumbers {
		if output, err := a.GetAccrualByOrderID(context.Background(), n); err != nil {
			fmt.Println("Error3", err.Error())
			return err
		}
		time.Sleep(100 * time.Millisecond)
	}
	*/
	return nil
}


