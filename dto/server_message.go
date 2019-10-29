package dto

type Message struct {
	From     string `from:"id"`
	FromType string `fromType:"id"`
	Message  string `message:"id"`
}
