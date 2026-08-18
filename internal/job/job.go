package job

import "time"

type Job struct {
	ReportID  string    `json:"report_id"`
	ForwardID string    `json:"forward_id"`
	RadarID   string    `json:"radar_id"`
	Kind      string    `json:"kind"`
	Body      []byte    `json:"body"`
	Attempt   int       `json:"attempt"`
	NotBefore time.Time `json:"not_before"`
	CreatedAt time.Time `json:"created_at"`
	ReplayOf  string    `json:"replay_of,omitempty"`
}

func (j Job) Clone() Job {
	c := j
	if j.Body != nil {
		c.Body = append([]byte(nil), j.Body...)
	}
	return c
}
