package circuitbreaker

type (
	conversionError string
)

var (
	ConversionError conversionError = "conversion error"
)

const (
	ConversionErrorCode = 2400
)

func (conversionError) ErrorCode() int {
	return ConversionErrorCode
}

func (e conversionError) Error() string {
	return string(e)
}
