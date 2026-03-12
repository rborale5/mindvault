package alexa

// --- Request types ---

type Request struct {
	Version string         `json:"version"`
	Session RequestSession `json:"session"`
	Context RequestContext `json:"context"`
	Request RequestBody    `json:"request"`
}

type RequestSession struct {
	New         bool              `json:"new"`
	SessionID   string            `json:"sessionId"`
	Application SessionApp        `json:"application"`
	User        SessionUser       `json:"user"`
	Attributes  map[string]string `json:"attributes,omitempty"`
}

type SessionApp struct {
	ApplicationID string `json:"applicationId"`
}

type SessionUser struct {
	UserID      string `json:"userId"`
	AccessToken string `json:"accessToken,omitempty"`
}

type RequestContext struct {
	System SystemContext `json:"System"`
}

type SystemContext struct {
	Device      DeviceContext `json:"device"`
	Application SessionApp   `json:"application"`
	APIEndpoint string       `json:"apiEndpoint"`
}

type DeviceContext struct {
	DeviceID string `json:"deviceId"`
}

type RequestBody struct {
	Type      string `json:"type"`
	RequestID string `json:"requestId"`
	Timestamp string `json:"timestamp"`
	Locale    string `json:"locale"`
	Intent    Intent `json:"intent,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

type Intent struct {
	Name               string          `json:"name"`
	ConfirmationStatus string          `json:"confirmationStatus,omitempty"`
	Slots              map[string]Slot `json:"slots,omitempty"`
}

type Slot struct {
	Name               string `json:"name"`
	Value              string `json:"value"`
	ConfirmationStatus string `json:"confirmationStatus,omitempty"`
}

// --- Response types ---

type Response struct {
	Version           string       `json:"version"`
	SessionAttributes map[string]string `json:"sessionAttributes,omitempty"`
	Response          ResponseBody `json:"response"`
}

type ResponseBody struct {
	OutputSpeech     *OutputSpeech `json:"outputSpeech,omitempty"`
	Card             *Card         `json:"card,omitempty"`
	Reprompt         *Reprompt     `json:"reprompt,omitempty"`
	ShouldEndSession bool          `json:"shouldEndSession"`
}

type OutputSpeech struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
	SSML string `json:"ssml,omitempty"`
}

type Card struct {
	Type    string `json:"type"`
	Title   string `json:"title,omitempty"`
	Content string `json:"content,omitempty"`
	Text    string `json:"text,omitempty"`
}

type Reprompt struct {
	OutputSpeech OutputSpeech `json:"outputSpeech"`
}

// --- Response builders ---

func NewTextResponse(text string, endSession bool) Response {
	return Response{
		Version: "1.0",
		Response: ResponseBody{
			OutputSpeech: &OutputSpeech{
				Type: "PlainText",
				Text: text,
			},
			ShouldEndSession: endSession,
		},
	}
}

func NewTextResponseWithCard(text, cardTitle, cardContent string, endSession bool) Response {
	resp := NewTextResponse(text, endSession)
	resp.Response.Card = &Card{
		Type:    "Simple",
		Title:   cardTitle,
		Content: cardContent,
	}
	return resp
}

func NewRepromptResponse(text, repromptText string) Response {
	return Response{
		Version: "1.0",
		Response: ResponseBody{
			OutputSpeech: &OutputSpeech{
				Type: "PlainText",
				Text: text,
			},
			Reprompt: &Reprompt{
				OutputSpeech: OutputSpeech{
					Type: "PlainText",
					Text: repromptText,
				},
			},
			ShouldEndSession: false,
		},
	}
}
