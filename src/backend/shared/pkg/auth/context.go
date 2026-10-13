package auth

import (
	"quadsmith/shared/pkg/authcontext"
)

type UserContext = authcontext.UserContext

var WithUserContext = authcontext.WithUserContext
var UserContextFromContext = authcontext.UserContextFromContext
var NewUUIDv7 = authcontext.NewUUIDv7
