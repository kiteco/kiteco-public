package segment

type commonFields struct {
	UserID    string `json:"userId"`
	Timestamp string `json:"timestamp"`
	Event     string `json:"event"`
}

type event struct {
	Event      string `json:"event"`
	Properties struct {
		Event string `json:"event"`
	} `json:"properties"`
}
