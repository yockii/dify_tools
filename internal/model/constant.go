package model

const (
	LogActionLogin = 1 + iota
	LogActionLogout
)
const (
	LogActionCreateUser = 11 + iota
	LogActionUpdateUser
	LogActionDeleteUser
	LogActionUpdatePassword
	LogActionUpdateUserStatus
)
const (
	LogActionCreateRole = 21 + iota
	LogActionUpdateRole
	LogActionDeleteRole
)
const (
	LogActionCreateApplication = 31 + iota
	LogActionUpdateApplication
	LogActionDeleteApplication
)

const (
	LogActionUpdateApplicationConfig = 41 + iota
)

const (
	LogActionCreateDataSource = 51 + iota
	LogActionUpdateDataSource
	LogActionDeleteDataSource
	LogActionSyncDataSource
	LogActionUpdateTableInfo
	LogActionUpdateColumnInfo
)

const (
	LogActionCreateDict = 61 + iota
	LogActionUpdateDict
	LogActionDeleteDict
)
