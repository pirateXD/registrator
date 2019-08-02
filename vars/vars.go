package vars

import "log"

var ConfigTTL int
var lastErrCode error

func InitConfigTTL(ttl int) {
	ConfigTTL = ttl
}

func SetLastErrCode(err error) {
	lastErrCode = err
	if err != nil {
		log.Println("bridge set errcode: ", err)
	} else {
		log.Println("bridge clear errcode: ")
	}
}

func GetLastErrCode() error {
	return lastErrCode
}
