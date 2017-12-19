package graylog

import (
	"testing"
	"os"
)

const namespaceName = "test-18F81907-7C88-438E-A648-2069EF5B95B2"

func TestMain(m *testing.M) {
	os.Setenv("GRAYLOG_BASE_URL","https://icc-logging.informatik.haw-hamburg.de")
	os.Setenv("GRAYLOG_ADMIN_USER","admin")
	os.Setenv("GRAYLOG_ADMIN_PASSWORD","ead4ce069ed01360e46aff03c47ec8f36748d558270351aa62d02a72e7d2379a")
	ret := m.Run()
	os.Exit(ret)
}

func TestCreateStream(t *testing.T) {
	done := CreateStream(namespaceName)
	if !done {t.Fail()}
	done2 := CreateStream(namespaceName)
	if !done2 {t.Fail()}
	DeleteStream(namespaceName)
}

func TestGetStreamByNamespaceName(t *testing.T){
	done := CreateStream(namespaceName)
	if !done {t.Fail()}
	str, err := getStreamByNamespaceName(namespaceName)
	if err != nil { t.Error(err)}
	if str.Title != namespaceName { t.Fail() }
	DeleteStream(namespaceName)
}

/*func TestDeleteStream(t *testing.T) {
	done := CreateStream(namespaceName)
	if !done {t.Fail()}
	DeleteStream(namespaceName)
	if isStreamAlreadyCreated(namespaceName) { t.Fail()}
}*/