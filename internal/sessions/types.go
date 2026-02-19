package sessions

import "time"

type Session struct {
	ID          string    `json:"id"`
	OwnerUserID string    `json:"owner_user_id"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

type User struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
}

type ServerAllocation struct {
	IP     string `json:"ip"`
	Port   int    `json:"port"`
	Region string `json:"region"`
}

type AssignServerResponse struct {
	Server ServerAllocation `json:"server"`
}
