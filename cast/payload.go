package cast

var (
	// Known Payload headers
	ConnectHeader     = PayloadHeader{Type: "CONNECT"}
	CloseHeader       = PayloadHeader{Type: "CLOSE"}
	GetStatusHeader   = PayloadHeader{Type: "GET_STATUS"}
	PongHeader        = PayloadHeader{Type: "PONG"}         // Response to PING payload
	LaunchHeader      = PayloadHeader{Type: "LAUNCH"}       // Launches a new chromecast app
	StopHeader        = PayloadHeader{Type: "STOP"}         // Stop playing current media
	PlayHeader        = PayloadHeader{Type: "PLAY"}         // Plays / unpauses the running app
	PauseHeader       = PayloadHeader{Type: "PAUSE"}        // Pauses the running app
	SeekHeader        = PayloadHeader{Type: "SEEK"}         // Seek into the running app
	VolumeHeader      = PayloadHeader{Type: "SET_VOLUME"}   // Sets the volume
	LoadHeader        = PayloadHeader{Type: "LOAD"}         // Loads an application onto the chromecast
	QueueLoadHeader   = PayloadHeader{Type: "QUEUE_LOAD"}   // Loads an application onto the chromecast
	QueueUpdateHeader = PayloadHeader{Type: "QUEUE_UPDATE"} // Loads an application onto the chromecast
)

type Payload interface {
	SetRequestId(id int)
}

type PayloadHeader struct {
	Type      string `json:"type"`
	RequestId int    `json:"requestId,omitempty"`
}

func (p *PayloadHeader) SetRequestId(id int) {
	p.RequestId = id
}

type QueueUpdate struct {
	PayloadHeader
	MediaSessionId int `json:"mediaSessionId,omitempty"`
	Jump           int `json:"jump,omitempty"`
}

type QueueLoad struct {
	PayloadHeader
	MediaSessionId int             `json:"mediaSessionId,omitempty"`
	CurrentTime    float32         `json:"currentTime"`
	StartIndex     int             `json:"startIndex"`
	RepeatMode     string          `json:"repeatMode"`
	Items          []QueueLoadItem `json:"items"`
}

type QueueLoadItem struct {
	Media            MediaItem `json:"media"`
	Autoplay         bool      `json:"autoplay"`
	PlaybackDuration int       `json:"playbackDuration"`
}

type MediaHeader struct {
	PayloadHeader
	MediaSessionId int     `json:"mediaSessionId"`
	CurrentTime    float32 `json:"currentTime"`
	RelativeTime   float32 `json:"relativeTime,omitempty"`
	ResumeState    string  `json:"resumeState"`
}

type Volume struct {
	Level float32 `json:"level"`
	Muted bool    `json:"muted"`
}

type ReceiverStatusResponse struct {
	PayloadHeader
	Status struct {
		Applications []Application `json:"applications"`
		Volume       Volume        `json:"volume"`
	} `json:"status"`
}

type Application struct {
	AppId        string `json:"appId"`
	DisplayName  string `json:"displayName"`
	IsIdleScreen bool   `json:"isIdleScreen"`
	SessionId    string `json:"sessionId"`
	StatusText   string `json:"statusText"`
	TransportId  string `json:"transportId"`
}

type ReceiverStatusRequest struct {
	PayloadHeader
	Applications []Application `json:"applications"`

	Volume Volume `json:"volume"`
}

type LaunchRequest struct {
	PayloadHeader
	AppId string `json:"appId"`
}

type LoadMediaCommand struct {
	PayloadHeader
	Media       MediaItem   `json:"media"`
	CurrentTime int         `json:"currentTime"`
	Autoplay    bool        `json:"autoplay"`
	QueueData   QueueData   `json:"queueData"`
	CustomData  interface{} `json:"customData"`
}

type QueueData struct {
	StartIndex int `json:"startIndex"`
}

type MediaItem struct {
	ContentId   string  `json:"contentId"`
	ContentType string  `json:"contentType"`
	StreamType  string  `json:"streamType"`
	Duration    float32 `json:"duration"`
	Metadata    struct {
		MetadataType int    `json:"metadataType`
		Title        string `json:"title"`
		SongName     string `json:"songName"`
		Artist       string `json:"artist"`
	} `json:"metadata"`
}

type Media struct {
	MediaSessionId int     `json:"mediaSessionId"`
	PlayerState    string  `json:"playerState"`
	CurrentTime    float32 `json:"currentTime"`
	IdleReason     string  `json:"idleReason"`
	Volume         Volume  `json:"volume"`

	Media MediaItem `json:"media"`
}

type MediaStatusResponse struct {
	PayloadHeader
	Status []Media `json:"status"`
}
