package domain

type Withdrawal struct {
	UserID int				`json:"-"`
	Order string			`json:"order"`
	Sum float32				`json:"sum"`
	ProcessedAt Time		`json:"processed_at"`
}