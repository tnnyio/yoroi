package transport

type (
	invalidRequest string
)

var (
	InvalidRequest invalidRequest = "invalid request"
)

const (
	InvalidRequestCode = 2400
)

func (invalidRequest) ErrorCode() int {
	return InvalidRequestCode
}

func (e invalidRequest) Error() string {
	return string(e)
}
