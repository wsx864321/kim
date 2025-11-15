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

// Error 获取错误信息
func (e *Error) Error() string {
	return e.message
}

// Code 获取错误码
func (e *Error) Code() int32 {
	return e.code
}

func (e *Error) WithMessage(message string) *Error {
	return &Error{
		code:    e.code,
		message: message,
	}
}

// Convert 将普通错误转换为*xerr.Error
func Convert(err error) *Error {
	if err == nil {
		return OK
	}
	if xe, ok := err.(*Error); ok {
		return xe
	}
	return ErrInternalServer
}
