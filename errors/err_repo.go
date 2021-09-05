package errors

var (
	CFBadRequest   = NewCFError(WithCode("400"), WithMessage("bad request"), WithStatus(400))
	CFNotFound     = NewCFError(WithCode("404"), WithMessage("not found"), WithStatus(404))
	CFUnauthorized = NewCFError(WithCode("401"), WithMessage("unauthorized"), WithStatus(401))
	CFInternalErr  = NewCFError(WithCode("500"), WithMessage("internal server error"), WithStatus(500))
)
