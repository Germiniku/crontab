package common

import "errors"

var (
	LOCK_ALREADY_REQUIRE = errors.New("锁已经被占用")
)
