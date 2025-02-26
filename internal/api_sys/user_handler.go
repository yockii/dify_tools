package sysapi

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/yockii/dify_tools/internal/constant"
	"github.com/yockii/dify_tools/internal/model"
	"github.com/yockii/dify_tools/internal/service"
)

type UserHandler struct {
	userService    service.UserService
	authService    service.AuthService
	roleService    service.RoleService
	logService     service.LogService
	sessionService service.SessionService
}

func RegisterUserHandler(
	userService service.UserService,
	authService service.AuthService,
	roleService service.RoleService,
	logService service.LogService,
	sessionService service.SessionService,
) {
	handler := &UserHandler{
		userService:    userService,
		authService:    authService,
		roleService:    roleService,
		logService:     logService,
		sessionService: sessionService,
	}
	Handlers = append(Handlers, handler)
}

func (h *UserHandler) RegisterRoutes(router fiber.Router, authMiddleware fiber.Handler) {
	auth := router.Group("/auth")
	{
		auth.Post("/login", h.Login)
		auth.Post("/refresh", authMiddleware, h.refreshToken)
		auth.Post("/logout", authMiddleware, h.logout)
	}

	users := router.Group("/users", authMiddleware)
	{
		users.Get("/profile", h.getProfile)
		users.Post("/new", h.createUser)
		users.Post("/update", h.updateUser)
		users.Post("/update_password", h.updatePassword)
		users.Post("/update_status", h.updateStatus)
		users.Post("/delete", h.deleteUser)
		users.Get("/info", h.getUser)
		users.Get("/list", h.listUsers)
	}

	roles := router.Group("/roles", authMiddleware)
	{
		roles.Post("/new", h.createRole)
		roles.Post("/update", h.updateRole)
		roles.Post("/delete", h.deleteRole)
		roles.Get("/info", h.getRole)
		roles.Get("/list", h.listRoles)
	}

	logs := router.Group("/logs", authMiddleware)
	{
		logs.Get("/list", h.listLogs)
		logs.Get("/user", h.getUserLogs)
	}
}

//////////////////////////////////////////////////////////////////////////////////
////                       AUTH                         //////////////////////////
//region//////////////////////////////////////////////////////////////////////////

// Login 登录
func (h *UserHandler) Login(c *fiber.Ctx) error {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	}
	uid, token, err := h.authService.Login(c.Context(), req.Username, req.Password)
	if err != nil {
		return c.Status(constant.GetErrorCode(err)).JSON(service.NewResponse(nil, err))
	}

	// 记录登录日志
	go h.logService.CreateLoginLog(c.Context(), uid, c.IP(), c.Get("User-Agent"), err == nil)

	return c.JSON(service.OK(fiber.Map{
		"token": token,
	}))
}

// refreshToken 刷新token
func (h *UserHandler) refreshToken(c *fiber.Ctx) error {
	authorization := c.Get("Authorization")
	token := strings.TrimPrefix(authorization, "Bearer ")

	newToken, err := h.authService.Refresh(c.Context(), token)
	if err != nil {
		return c.Status(constant.GetErrorCode(err)).JSON(service.NewResponse(nil, err))
	}

	return c.JSON(service.OK(fiber.Map{
		"token": newToken,
	}))
}

// logout 退出登录
func (h *UserHandler) logout(c *fiber.Ctx) error {
	authorization := c.Get("Authorization")
	token := strings.TrimPrefix(authorization, "Bearer ")

	if err := h.authService.Logout(c.Context(), token); err != nil {
		return c.Status(constant.GetErrorCode(err)).JSON(service.Error(err))
	}

	// 记录登出日志
	user := c.Locals("user").(*model.User)
	go h.logService.CreateOperationLog(c.Context(), user.ID, constant.LogActionLogout, c.IP(), c.Get("User-Agent"))

	return c.JSON(service.OK(nil))
}

//endregion

//////////////////////////////////////////////////////////////////////////////////
////                       USER                         //////////////////////////
//region//////////////////////////////////////////////////////////////////////////

// getProfile 获取用户信息
func (h *UserHandler) getProfile(c *fiber.Ctx) error {
	user := c.Locals("user").(*model.User)
	return c.JSON(service.OK(user))
}

// createUser 创建用户
func (h *UserHandler) createUser(c *fiber.Ctx) error {
	var user model.User
	if err := c.BodyParser(&user); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	}

	if err := h.userService.Create(c.Context(), &user); err != nil {
		return c.Status(constant.GetErrorCode(err)).JSON(service.Error(err))
	}

	// 记录操作日志
	creator := c.Locals("user").(*model.User)
	go h.logService.CreateOperationLog(c.Context(), creator.ID, constant.LogActionCreateUser, c.IP(), c.Get("User-Agent"))

	return c.Status(fiber.StatusCreated).JSON(service.OK(user))
}

// updateUser 更新用户
func (h *UserHandler) updateUser(c *fiber.Ctx) error {
	var user model.User
	if err := c.BodyParser(&user); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	}
	if user.ID == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	}

	if err := h.userService.Update(c.Context(), &user); err != nil {
		return c.Status(constant.GetErrorCode(err)).JSON(service.Error(err))
	}

	// 记录操作日志
	operator := c.Locals("user").(*model.User)
	go h.logService.CreateOperationLog(c.Context(), operator.ID, constant.LogActionUpdateUser, c.IP(), c.Get("User-Agent"))

	return c.JSON(service.OK(user))
}

// deleteUser 删除用户
func (h *UserHandler) deleteUser(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Query("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	}

	if err := h.userService.Delete(c.Context(), id); err != nil {
		return c.Status(constant.GetErrorCode(err)).JSON(service.Error(err))
	}

	// 记录操作日志
	operator := c.Locals("user").(*model.User)
	go h.logService.CreateOperationLog(c.Context(), operator.ID, constant.LogActionDeleteUser, c.IP(), c.Get("User-Agent"))

	return c.JSON(service.OK(nil))
}

// getUser 获取用户
func (h *UserHandler) getUser(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Query("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	}

	user, err := h.userService.Get(c.Context(), id)
	if err != nil {
		return c.Status(constant.GetErrorCode(err)).JSON(service.Error(err))
	}

	return c.JSON(service.OK(user))
}

// listUsers 获取用户列表
func (h *UserHandler) listUsers(c *fiber.Ctx) error {
	offset := c.QueryInt("offset", 0)
	limit := c.QueryInt("limit", service.DefaultPageSize)
	if limit > service.MaxPageSize {
		limit = service.MaxPageSize
	}
	condition := new(model.User)
	if err := c.QueryParser(condition); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	}

	users, total, err := h.userService.List(c.Context(), condition, offset, limit)
	if err != nil {
		return c.Status(constant.GetErrorCode(err)).JSON(service.Error(err))
	}

	return c.JSON(service.OK(service.NewListResponse(users, total, offset, limit)))
}

// updatePassword 更新密码
func (h *UserHandler) updatePassword(c *fiber.Ctx) error {
	user := c.Locals("user").(*model.User)

	var req struct {
		OldPassword string `json:"oldPassword"`
		NewPassword string `json:"newPassword"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	}

	if err := h.userService.UpdatePassword(c.Context(), user.ID, req.OldPassword, req.NewPassword); err != nil {
		return c.Status(constant.GetErrorCode(err)).JSON(service.NewResponse(nil, err))
	}

	// 记录操作日志
	go h.logService.CreateOperationLog(c.Context(), user.ID, constant.LogActionUpdatePassword, c.IP(), c.Get("User-Agent"))

	return c.JSON(service.OK(nil))
}

// updateStatus 更新用户状态
func (h *UserHandler) updateStatus(c *fiber.Ctx) error {
	var req struct {
		ID     uint64 `json:"id,string"`
		Status int    `json:"status"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	}

	if err := h.userService.UpdateStatus(c.Context(), req.ID, req.Status); err != nil {
		return c.Status(constant.GetErrorCode(err)).JSON(service.NewResponse(nil, err))
	}

	// 记录操作日志
	operator := c.Locals("user").(*model.User)
	go h.logService.CreateOperationLog(c.Context(), operator.ID, constant.LogActionUpdateUserStatus, c.IP(), c.Get("User-Agent"))

	return c.JSON(service.OK(nil))
}

//endregion

//////////////////////////////////////////////////////////////////////////////////
////                       ROLE                         //////////////////////////
//region//////////////////////////////////////////////////////////////////////////

// createRole 创建角色
func (h *UserHandler) createRole(c *fiber.Ctx) error {
	var role model.Role
	if err := c.BodyParser(&role); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	}

	if err := h.roleService.Create(c.Context(), &role); err != nil {
		return c.Status(constant.GetErrorCode(err)).JSON(service.Error(err))
	}

	// 记录操作日志
	user := c.Locals("user").(*model.User)
	go h.logService.CreateOperationLog(c.Context(), user.ID, constant.LogActionCreateRole, c.IP(), c.Get("User-Agent"))

	return c.Status(fiber.StatusCreated).JSON(service.OK(role))
}

// updateRole 更新角色
func (h *UserHandler) updateRole(c *fiber.Ctx) error {
	var role model.Role
	if err := c.BodyParser(&role); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	}

	if role.ID == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	}

	if err := h.roleService.Update(c.Context(), &role); err != nil {
		return c.Status(constant.GetErrorCode(err)).JSON(service.Error(err))
	}

	// 记录操作日志
	user := c.Locals("user").(*model.User)
	go h.logService.CreateOperationLog(c.Context(), user.ID, constant.LogActionUpdateRole, c.IP(), c.Get("User-Agent"))

	return c.JSON(service.OK(role))
}

// deleteRole 删除角色
func (h *UserHandler) deleteRole(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Query("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	}

	if err := h.roleService.Delete(c.Context(), id); err != nil {
		return c.Status(constant.GetErrorCode(err)).JSON(service.Error(err))
	}

	// 记录操作日志
	user := c.Locals("user").(*model.User)
	go h.logService.CreateOperationLog(c.Context(), user.ID, constant.LogActionDeleteRole, c.IP(), c.Get("User-Agent"))

	return c.JSON(service.OK(nil))
}

// getRole 获取角色信息
func (h *UserHandler) getRole(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Query("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	}

	role, err := h.roleService.Get(c.Context(), id)
	if err != nil {
		return c.Status(constant.GetErrorCode(err)).JSON(service.Error(err))
	}

	return c.JSON(service.OK(role))
}

// listRoles 获取角色列表
func (h *UserHandler) listRoles(c *fiber.Ctx) error {
	offset := c.QueryInt("offset", 0)
	limit := c.QueryInt("limit", service.DefaultPageSize)
	if limit > service.MaxPageSize {
		limit = service.MaxPageSize
	}

	condition := new(model.Role)
	if err := c.QueryParser(condition); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	}

	roles, total, err := h.roleService.List(c.Context(), condition, offset, limit)
	if err != nil {
		return c.Status(constant.GetErrorCode(err)).JSON(service.Error(err))
	}

	return c.JSON(service.OK(service.NewListResponse(roles, total, offset, limit)))
}

//endregion

//////////////////////////////////////////////////////////////////////////////////
////                       LOG                         //////////////////////////
//region/////////////////////////////////////////////////////////////////////////

// listLogs 获取日志列表
func (h *UserHandler) listLogs(c *fiber.Ctx) error {
	offset := c.QueryInt("offset", 0)
	limit := c.QueryInt("limit", service.DefaultPageSize)
	if limit > service.MaxPageSize {
		limit = service.MaxPageSize
	}
	var actions []int
	if v := c.Query("actions"); v != "" {
		for _, action := range strings.Split(v, ",") {
			if a, err := strconv.Atoi(action); err == nil {
				actions = append(actions, a)
			}
		}
	}

	logs, total, err := h.logService.ListLogs(c.Context(), 0, actions, offset, limit)
	if err != nil {
		return c.Status(constant.GetErrorCode(err)).JSON(service.Error(err))
	}

	return c.JSON(service.NewResponse(service.NewListResponse(logs, total, offset, limit), nil))
}

// getUserLogs 获取用户日志
func (h *UserHandler) getUserLogs(c *fiber.Ctx) error {
	userID, err := strconv.ParseUint(c.Query("userId"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	}

	offset := c.QueryInt("offset", 0)
	limit := c.QueryInt("limit", service.DefaultPageSize)
	if limit > service.MaxPageSize {
		limit = service.MaxPageSize
	}

	var actions []int
	if v := c.Query("actions"); v != "" {
		for _, action := range strings.Split(v, ",") {
			if a, err := strconv.Atoi(action); err == nil {
				actions = append(actions, a)
			}
		}
	}

	logs, total, err := h.logService.ListLogs(c.Context(), userID, actions, offset, limit)
	if err != nil {
		return c.Status(constant.GetErrorCode(err)).JSON(service.Error(err))
	}

	return c.JSON(service.NewResponse(service.NewListResponse(logs, total, offset, limit), nil))
}

//endregion
