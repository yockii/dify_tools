package constant

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
	LogActionUpdateApplicationConfig
)

const (
	LogActionCreateDataSource = 41 + iota
	LogActionUpdateDataSource
	LogActionDeleteDataSource
	LogActionSyncDataSource
	LogActionUpdateTableInfo
	LogActionUpdateColumnInfo
	LogActionDeleteTableInfo
	LogActionDeleteColumnInfo
)

const (
	LogActionCreateDict = 51 + iota
	LogActionUpdateDict
	LogActionDeleteDict
)
