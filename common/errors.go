package common

func (obj *VMAlreadyExistsError) Error() string {
	return "vm already exists"
}

type VMAlreadyExistsError struct {}