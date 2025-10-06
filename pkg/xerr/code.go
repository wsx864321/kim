package xerr

const (
	// 通用错误码 10000 - 19999
	ErrOKCode                 int32 = 0
	ErrBadRequestCode         int32 = 10001
	ErrUnauthorizedCode       int32 = 10002
	ErrForbiddenCode          int32 = 10003
	ErrNotFoundCode           int32 = 10004
	ErrConflictCode           int32 = 10005
	ErrInternalServerCode     int32 = 10006
	ErrServiceUnavailableCode int32 = 10007
	ErrDeadlineExceededCode   int32 = 10008
	ErrTooManyRequestsCode    int32 = 10009
	ErrInvalidParamsCode      int32 = 10010

	// Session 模块错误码 20000 - 29999
	ErrSessionNotFoundCode       int32 = 20001
	ErrSessionExpiredCode        int32 = 20002
	ErrSessionDuplicateLoginCode int32 = 20003
	ErrSessionKickOffCode        int32 = 20004
	ErrSessionBindFailedCode     int32 = 20005
	ErrSessionTokenInvalidCode   int32 = 20006
	ErrSessionUserMismatchCode   int32 = 20007
	ErrSessionAlreadyOfflineCode int32 = 20008
	ErrSessionStateCorruptCode   int32 = 20009
)

// 通用错误实例10000 - 19999
var (
	OK                    = NewError(ErrOKCode, "ok")
	ErrBadRequest         = NewError(ErrBadRequestCode, "bad request")
	ErrUnauthorized       = NewError(ErrUnauthorizedCode, "unauthorized")
	ErrForbidden          = NewError(ErrForbiddenCode, "forbidden")
	ErrNotFound           = NewError(ErrNotFoundCode, "not found")
	ErrConflict           = NewError(ErrConflictCode, "conflict")
	ErrInternalServer     = NewError(ErrInternalServerCode, "internal server error")
	ErrServiceUnavailable = NewError(ErrServiceUnavailableCode, "service unavailable")
	ErrDeadlineExceeded   = NewError(ErrDeadlineExceededCode, "deadline exceeded")
	ErrTooManyRequests    = NewError(ErrTooManyRequestsCode, "too many requests")
	ErrInvalidParams      = NewError(ErrInvalidParamsCode, "invalid parameters")
)

// Session 模块错误实例20000 - 29999
var (
	ErrSessionNotFound       = NewError(ErrSessionNotFoundCode, "session not found")
	ErrSessionExpired        = NewError(ErrSessionExpiredCode, "session expired")
	ErrSessionDuplicateLogin = NewError(ErrSessionDuplicateLoginCode, "duplicate login")
	ErrSessionKickOff        = NewError(ErrSessionKickOffCode, "kicked off")
	ErrSessionBindFailed     = NewError(ErrSessionBindFailedCode, "bind connection failed")
	ErrSessionTokenInvalid   = NewError(ErrSessionTokenInvalidCode, "invalid token")
	ErrSessionUserMismatch   = NewError(ErrSessionUserMismatchCode, "user mismatch")
	ErrSessionAlreadyOffline = NewError(ErrSessionAlreadyOfflineCode, "already offline")
	ErrSessionStateCorrupt   = NewError(ErrSessionStateCorruptCode, "session state corrupt")
)
