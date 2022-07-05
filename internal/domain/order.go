package domain

type OrderStatus string

const (
	OrderStatusRegistered OrderStatus = "REGISTERED" // order is registered, but no accrual is calculated
	OrderStatusInvalid    OrderStatus = "INVALID"    // order was not accepted for settlement, and the reward will not be credited
	OrderStatusProcessing OrderStatus = "PROCESSING" // calculation of the accrual in the process
	OrderStatusProcessed  OrderStatus = "PROCESSED"  // calculation is completed
	OrderStatusNew        OrderStatus = "NEW"     	//  the order was uploaded to the system, but was not processed
)

func (r OrderStatus) String() string {
	return string(r)
}

func (r *OrderStatus) MarshalJSON() ([]byte, error) {
	return []byte(`"` + r.String() + `"`), nil
}

func (r *OrderStatus) UnmarshalJSON(data []byte) error {
	*r = OrderStatus(data[1 : len(data)-1])
	return nil
}
//**********************************************************************************************************************
type Order struct {
	Number string			`json:"number"`
	UserID int				`json:"-"`
	Status OrderStatus		`json:"status"`
	Accrual float32			`json:"accrual,omitempty"`
	UploadedAt Time			`json:"uploaded_at"`
}