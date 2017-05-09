package eventinfrastructure

type Event struct {
	Hostname         string `json:"hostname,omitempty"`
	Timestamp        string `json:"timestamp,omitempty"`
	LocalEnvironment bool   `json:"localEnvironment,omitempty"`
	Event            string `json:"event,omitempty"`
	ResponseCode     int    `json:"responseCode,omitempty"`
	Success          bool   `json:"success,omitempty"`
	Building         string `json:"building,omitempty"`
	Room             string `json:"room,omitempty"`
	Device           string `json:"device,omitempty"`
}
