package xerr

// Error 错误码和信息
type Error struct {
	code    int32
	message string
}

// NewError 生成一个error
func NewError(code int32, message string) *Error {
	return &Error{
		code:    code,
		message: message,
	}
}

func (e *Error) Error() string {
	return e.message
}

func (e *Error) Code() int32 {
	return e.code
}
