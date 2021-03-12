package common

// VMAlreadyExistsError returns the vm already exists error
type VMAlreadyExistsError struct{}

func (obj *VMAlreadyExistsError) Error() string {
	return "vm already exists"
}

// VMNotFoundException returns the vm not found error
type VMNotFoundException struct{}

func (obj *VMNotFoundException) Error() string {
	return "vm not found"
}
