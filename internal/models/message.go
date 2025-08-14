package models

type UserMessage struct {
    UserID  int64  `json:"user_id"`
    Text    string `json:"text"`
}

type AdminResponse struct {
    Approved bool   `json:"approved"`
    MessageID int64 `json:"message_id"`
}

type GroupMessage struct {
    Text string `json:"text"`
}