package types

type QueryChallengesResponse struct {
	Challenges []Challenge `json:"challenges"`
	Next       *string     `json:"next,omitempty"`
}
