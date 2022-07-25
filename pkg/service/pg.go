package service

// Advisory locks are identified by two 32 bit integers. We arbitrarily
// reserve a value for the first one for all locks taken by Eventline.
//
// Note that go-daemon uses 0x00ff.
const PgAdvisoryLockId1 uint32 = 0x0100

const (
	PgAdvisoryLockId2ServiceInit   uint32 = 0x0001
	PgAdvisoryLockId2JobScheduling uint32 = 0x0002
	PgAdvisoryLockId2JobDeployment uint32 = 0x0003
)
