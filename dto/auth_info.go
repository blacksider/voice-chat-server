package dto

type AuthAuthority struct {
	Id        int64  `json:"id"`
	Authority string `json:"authority"`
}

type AuthInfo struct {
	Username    string          `json:"username"`
	Authorities []AuthAuthority `json:"authorities"`
}
