package types

type UserType struct {
	Id              string `json:"id"`
	Name            string `json:"name"`
	Username        string `json:"username"`
	Email           string `json:"email"`
	UniversalReader bool   `json:"universalReader"`
	UniversalWriter bool   `json:"universalWriter"`
}
