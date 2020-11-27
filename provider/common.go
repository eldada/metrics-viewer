package provider

import "time"

var nowFunc = time.Now

func now() time.Time {
	return nowFunc()
}
