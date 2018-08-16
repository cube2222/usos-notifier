package users

type UserID string

func NewUserID(id string) UserID {
	return UserID(id)
}

func (id UserID) String() string {
	return string(id)
}
