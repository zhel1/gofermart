package domain

import "time"

type Time time.Time

func (r *Time) MarshalJSON() ([]byte, error) {
	return []byte(`"` + time.Time(*r).Format(time.RFC3339) + `"`), nil
}

func (r *Time) UnmarshalJSON(data []byte) error {
	t, err := time.Parse(time.RFC3339, string(data[1 : len(data)-1]))
	*r = Time(t)
	return err
}
