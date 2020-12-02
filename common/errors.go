package common

type VMAlreadyExistsError struct{}

func (obj *VMAlreadyExistsError) Error() string {
	return "vm already exists"
}

type VMNotFoundException struct{}

func (obj *VMNotFoundException) Error() string {
	return "vm not found"
}
