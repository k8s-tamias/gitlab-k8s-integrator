package graylog

import (
	"testing"
	"os"
)

func TestMain(m *testing.M) {
	os.Setenv("GRAYLOG_BASE_URL","https://icc-logging.informatik.haw-hamburg.de")
	os.Setenv("GRAYLOG_ADMIN_USER","admin")
	os.Setenv("GRAYLOG_ADMIN_PASSWORD","ead4ce069ed01360e46aff03c47ec8f36748d558270351aa62d02a72e7d2379a")
	ret := m.Run()
	os.Exit(ret)
}

func TestDeleteStream(t *testing.T) {
	done := CreateStream("default")
	if !done {t.Fail()}
	DeleteStream("default")
	if isStreamAlreadyCreated("default") { t.Fail()}
}

func TestCreateStream(t *testing.T) {

	done := CreateStream("default")
	if !done {t.Fail()}
	done2 := CreateStream("default")
	if !done2 {t.Fail()}
	DeleteStream("default")
}
