package auth

import (
	"quadsmith/api/internal/authcontext"
)

type UserContext = authcontext.UserContext

var WithUserContext = authcontext.WithUserContext
var UserContextFromContext = authcontext.UserContextFromContext
var NewUUIDv7 = authcontext.NewUUIDv7
