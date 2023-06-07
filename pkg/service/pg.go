package service

// Advisory locks are identified by two 32 bit integers. We arbitrarily
// reserve a value for the first one for all locks taken by Eventline.
//
// Note that go-service uses 0x0100 (pg.AdvisoryLockId1).
const PgAdvisoryLockId1 uint32 = 0x0101

const (
	PgAdvisoryLockId2ServiceInit   uint32 = 0x0001
	PgAdvisoryLockId2JobScheduling uint32 = 0x0002
	PgAdvisoryLockId2JobDeployment uint32 = 0x0003
)
