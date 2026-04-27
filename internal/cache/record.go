package cache

import "time"

type Record struct {
	Body       []byte
	ValidUntil time.Time
	StaleBody  []byte
	StaleUntil time.Time
}

func (r *Record) primaryOK(now time.Time) bool {
	return r != nil && len(r.Body) > 0 && now.Before(r.ValidUntil)
}

func (r *Record) staleOK(now time.Time) bool {
	return r != nil && len(r.StaleBody) > 0 && now.Before(r.StaleUntil)
}
