package main

import (
	"github.com/go-kratos/kratos/v2"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"go-wind-oa/app/core/service/internal/data"
	"go-wind-oa/app/core/service/internal/data/client"
	"go-wind-oa/app/core/service/internal/server"
	"go-wind-oa/app/core/service/internal/service"
)

// initApp 手写装配整个应用,是本服务的依赖注入点(替代原 google/wire 生成)。
// initApp assembles the whole application by hand — the dependency injection point.
//
// 装配严格单向分层,自上而下的阅读顺序即依赖方向:
// The wiring is strictly layered; reading top-down follows the dependency direction:
//
//	基础设施 → 仓储层(data) → 服务层(service) → 传输层(server)
//
// 约定 / Conventions:
//   - 新增仓储/服务:在对应分层小节追加一行构造,并传给下游消费者;漏接由编译器在调用处报错。
//     To add a repo/service: append one constructor line in its layer section, then pass it to
//     downstream consumers; a missing connection is a compile error at the call site.
//   - 持有 cleanup 的资源创建成功后立即注册进 cleanups;任何一步失败,rollback 逆序执行已注册
//     的清理(shutdown 与中途失败共用同一条 LIFO 路径)。
//     Resources owning a cleanup register it immediately; rollback runs them LIFO both on
//     mid-way failure and on shutdown.
//   - 本文件只做构造与传参,不写业务逻辑。
//     Construction and parameter passing only; no business logic in this file.
func initApp(ctx *bootstrap.Context) (*kratos.App, func(), error) {
	// cleanup 注册表:rollback 时逆序执行。
	// Cleanup registry; rollback runs entries in reverse order.
	var cleanups []func()
	rollback := func() {
		for i := len(cleanups) - 1; i >= 0; i-- {
			cleanups[i]()
		}
	}

	// ═══════════════════════ 一、基础设施 ═══════════════════════

	authenticatorOption := data.NewAuthenticatorConfig(ctx)
	redisClient, cleanupRedis, err := client.NewRedisClient(ctx)
	if err != nil {
		rollback()
		return nil, nil, err
	}
	cleanups = append(cleanups, cleanupRedis)

	userTokenCache := data.NewUserTokenCache(ctx, redisClient)
	authenticator := data.NewAuthenticator(ctx, authenticatorOption, userTokenCache)
	entClient, cleanupEnt, err := client.NewEntClient(ctx)
	if err != nil {
		rollback()
		return nil, nil, err
	}
	cleanups = append(cleanups, cleanupEnt)

	crypto := data.NewPasswordCrypto()
	mfaChallengeCache := data.NewMfaChallengeCache(ctx, redisClient)
	minIOClient := client.NewMinIoClient(ctx)
	loginRateLimiter := data.NewLoginRateLimiter(ctx, redisClient)

	// ═══════════════════════ 二、仓储层(internal/data) ═══════════════════════

	userCredentialRepo := data.NewUserCredentialRepo(ctx, entClient, crypto)
	userRoleRepo := data.NewUserRoleRepo(ctx, entClient)
	userOrgUnitRepo := data.NewUserOrgUnitRepo(ctx, entClient)
	userPositionRepo := data.NewUserPositionRepo(ctx, entClient)
	membershipRoleRepo := data.NewMembershipRoleRepo(ctx, entClient)
	membershipPositionRepo := data.NewMembershipPositionRepo(ctx, entClient)
	membershipOrgUnitRepo := data.NewMembershipOrgUnitRepo(ctx, entClient)
	membershipRepo := data.NewMembershipRepo(ctx, entClient, membershipRoleRepo, membershipPositionRepo, membershipOrgUnitRepo)
	userRepo := data.NewUserRepo(ctx, entClient, userRoleRepo, userOrgUnitRepo, userPositionRepo, membershipRepo)
	rolePermissionRepo := data.NewRolePermissionRepo(ctx, entClient)
	permissionApiRepo := data.NewPermissionApiRepo(ctx, entClient)
	permissionMenuRepo := data.NewPermissionMenuRepo(ctx, entClient)
	permissionRepo := data.NewPermissionRepo(ctx, entClient, permissionApiRepo, permissionMenuRepo, rolePermissionRepo)
	roleMetadataRepo := data.NewRoleMetadataRepo(ctx, entClient)
	roleRepo := data.NewRoleRepo(ctx, entClient, rolePermissionRepo, permissionRepo, roleMetadataRepo, userRoleRepo)
	tenantRepo := data.NewTenantRepo(ctx, entClient)
	userMfaFactorRepo := data.NewUserMfaFactorRepo(ctx, entClient)
	loginPolicyRepo := data.NewLoginPolicyRepo(ctx, entClient)
	taskRepo := data.NewTaskRepo(ctx, entClient)
	fileRepo := data.NewFileRepo(ctx, entClient)
	dictEntryI18nRepo := data.NewDictEntryI18nRepo(ctx, entClient)
	dictEntryRepo := data.NewDictEntryRepo(ctx, entClient, dictEntryI18nRepo)
	dictTypeRepo := data.NewDictTypeRepo(ctx, entClient, dictEntryRepo)
	languageRepo := data.NewLanguageRepo(ctx, entClient)
	tenantUsageRepo := data.NewTenantUsageRepo(ctx, entClient, authenticator)
	positionRepo := data.NewPositionRepo(ctx, entClient)
	orgUnitRepo := data.NewOrgUnitRepo(ctx, entClient)
	menuRepo := data.NewMenuRepo(ctx, entClient)
	apiRepo := data.NewApiRepo(ctx, entClient)
	permissionGroupRepo := data.NewPermissionGroupRepo(ctx, entClient)
	permissionAuditLogRepo := data.NewPermissionAuditLogRepo(ctx, entClient)
	policyEvaluationLogRepo := data.NewPolicyEvaluationLogRepo(ctx, entClient)
	loginAuditLogRepo := data.NewLoginAuditLogRepo(ctx, entClient)
	dashboardRepo := data.NewDashboardRepo(ctx, entClient)
	planRepo := data.NewPlanRepo(ctx, entClient)
	planModuleRepo := data.NewPlanModuleRepo(ctx, entClient)
	planQuotaRepo := data.NewPlanQuotaRepo(ctx, entClient)
	redisCacheMonitorRepo := data.NewRedisCacheMonitorRepo(ctx, redisClient)
	apiAuditLogRepo := data.NewApiAuditLogRepo(ctx, entClient)
	operationAuditLogRepo := data.NewOperationAuditLogRepo(ctx, entClient)
	dataAccessAuditLogRepo := data.NewDataAccessAuditLogRepo(ctx, entClient)
	internalMessageRecipientRepo := data.NewInternalMessageRecipientRepo(ctx, entClient)
	internalMessageRepo := data.NewInternalMessageRepo(ctx, entClient, internalMessageRecipientRepo)
	internalMessageCategoryRepo := data.NewInternalMessageCategoryRepo(ctx, entClient)
	workflowDefinitionRepo := data.NewWorkflowDefinitionRepo(ctx, entClient)
	workflowInstanceRepo := data.NewWorkflowInstanceRepo(ctx, entClient)
	workflowTaskRepo := data.NewWorkflowTaskRepo(ctx, entClient)
	workflowLogRepo := data.NewWorkflowLogRepo(ctx, entClient)
	workflowResolverRepo := data.NewWorkflowResolverRepo(ctx, entClient)
	workflowInstanceJoinRepo := data.NewWorkflowInstanceJoinRepo(ctx, entClient)
	workflowInstanceParentLinkRepo := data.NewWorkflowInstanceParentLinkRepo(ctx, entClient)
	workflowDelegationRepo := data.NewWorkflowDelegationRepo(ctx, entClient)
	leaveTypeRepo := data.NewLeaveTypeRepo(ctx, entClient)
	leaveBalanceRepo := data.NewLeaveBalanceRepo(ctx, entClient)
	leaveApplicationRepo := data.NewLeaveApplicationRepo(ctx, entClient)
	attendanceRepo := data.NewAttendanceRepo(ctx, entClient)
	geofenceRepo := data.NewGeofenceRepo(ctx, entClient)
	wifiFingerprintRepo := data.NewWifiFingerprintRepo(ctx, entClient)
	expenseApplicationRepo := data.NewExpenseApplicationRepo(ctx, entClient)
	businessTripApplicationRepo := data.NewBusinessTripApplicationRepo(ctx, entClient)
	outingApplicationRepo := data.NewOutingApplicationRepo(ctx, entClient)
	overtimeApplicationRepo := data.NewOvertimeApplicationRepo(ctx, entClient)
	sealApplicationRepo := data.NewSealApplicationRepo(ctx, entClient)

	// ── register:repo ── 新模块仓储在此行后注册(make register 工具锚点,勿删)

	// ═══════════════════════ 三、服务层(internal/service) ═══════════════════════

	authenticationService := service.NewAuthenticationService(ctx, authenticator, userCredentialRepo, userRepo, roleRepo, tenantRepo, permissionRepo, userMfaFactorRepo, mfaChallengeCache)
	loginPolicyService := service.NewLoginPolicyService(ctx, loginPolicyRepo)
	userCredentialService := service.NewUserCredentialService(ctx, userCredentialRepo)
	taskService := service.NewTaskService(ctx, taskRepo, userRepo)
	fileService := service.NewFileService(ctx, fileRepo, minIOClient)
	dictTypeService := service.NewDictTypeService(ctx, dictTypeRepo)
	dictEntryService := service.NewDictEntryService(ctx, dictEntryRepo)
	languageService := service.NewLanguageService(ctx, languageRepo)
	tenantService := service.NewTenantService(ctx, tenantRepo, userRepo, userCredentialRepo, roleRepo, tenantUsageRepo)
	userService := service.NewUserService(ctx, userRepo, roleRepo, userCredentialRepo, positionRepo, orgUnitRepo, tenantRepo, membershipRepo)
	roleService := service.NewRoleService(ctx, roleRepo, tenantRepo, userRoleRepo, userRepo)
	positionService := service.NewPositionService(ctx, positionRepo, orgUnitRepo)
	orgUnitService := service.NewOrgUnitService(ctx, orgUnitRepo, userRepo)
	menuService := service.NewMenuService(ctx, menuRepo)
	apiService := service.NewApiService(ctx, apiRepo)
	permissionService := service.NewPermissionService(ctx, permissionRepo, permissionGroupRepo, menuRepo, apiRepo, roleRepo)
	permissionGroupService := service.NewPermissionGroupService(ctx, permissionGroupRepo, permissionRepo)
	permissionAuditLogService := service.NewPermissionAuditLogService(ctx, permissionAuditLogRepo)
	policyEvaluationLogService := service.NewPolicyEvaluationLogService(ctx, policyEvaluationLogRepo)
	loginAuditLogService := service.NewLoginAuditLogService(ctx, loginAuditLogRepo)
	dashboardService := service.NewDashboardService(ctx, dashboardRepo)
	planService := service.NewPlanService(ctx, planRepo)
	planModuleService := service.NewPlanModuleService(ctx, planModuleRepo)
	planQuotaService := service.NewPlanQuotaService(ctx, planQuotaRepo)
	mfaService := service.NewMFAService(ctx, userMfaFactorRepo, mfaChallengeCache, authenticator, loginRateLimiter)
	redisCacheMonitorService := service.NewRedisCacheMonitorService(ctx, redisCacheMonitorRepo)
	apiAuditLogService := service.NewApiAuditLogService(ctx, apiAuditLogRepo, apiRepo)
	operationAuditLogService := service.NewOperationAuditLogService(ctx, operationAuditLogRepo)
	dataAccessAuditLogService := service.NewDataAccessAuditLogService(ctx, dataAccessAuditLogRepo)
	internalMessageService := service.NewInternalMessageService(ctx, internalMessageRepo, internalMessageCategoryRepo, internalMessageRecipientRepo, userRepo)
	internalMessageCategoryService := service.NewInternalMessageCategoryService(ctx, internalMessageCategoryRepo)
	internalMessageRecipientService := service.NewInternalMessageRecipientService(ctx, internalMessageRepo, internalMessageRecipientRepo)
	workflowEventRegistry := service.NewWorkflowEventRegistry()
	workflowService := service.NewWorkflowService(ctx, workflowDefinitionRepo, workflowInstanceRepo, workflowTaskRepo, workflowLogRepo, workflowResolverRepo, workflowInstanceJoinRepo, workflowInstanceParentLinkRepo, workflowDelegationRepo, workflowEventRegistry, internalMessageService)
	attendanceService := service.NewAttendanceService(ctx, attendanceRepo, leaveApplicationRepo, workflowResolverRepo, geofenceRepo, wifiFingerprintRepo)
	leaveService := service.NewLeaveService(ctx, leaveTypeRepo, leaveBalanceRepo, leaveApplicationRepo, workflowDefinitionRepo, workflowResolverRepo, attendanceService, workflowService, workflowEventRegistry)
	expenseService := service.NewExpenseService(ctx, expenseApplicationRepo, workflowDefinitionRepo, workflowResolverRepo, workflowService, workflowEventRegistry)
	businessTripService := service.NewBusinessTripService(ctx, businessTripApplicationRepo, workflowDefinitionRepo, workflowResolverRepo, workflowService, workflowEventRegistry)
	outingService := service.NewOutingService(ctx, outingApplicationRepo, workflowDefinitionRepo, workflowResolverRepo, workflowService, workflowEventRegistry)
	overtimeService := service.NewOvertimeService(ctx, overtimeApplicationRepo, workflowDefinitionRepo, workflowResolverRepo, workflowService, workflowEventRegistry)
	sealApplicationService := service.NewSealApplicationService(ctx, sealApplicationRepo, workflowDefinitionRepo, workflowResolverRepo, workflowService, workflowEventRegistry)
	workflowTimeoutScheduler := service.NewWorkflowTimeoutScheduler(ctx, workflowTaskRepo, workflowInstanceRepo, workflowResolverRepo, workflowLogRepo, internalMessageService)
	attendanceScheduler := service.NewAttendanceScheduler(ctx, attendanceService)

	// ── register:service ── 新模块服务在此行后注册(make register 工具锚点,勿删)

	// ═══════════════════════ 四、传输层(internal/server) ═══════════════════════

	grpcMiddlewares := server.NewGrpcMiddleware(ctx)

	grpcServer, err := server.NewGrpcServer(ctx, grpcMiddlewares,
		authenticationService,
		loginPolicyService,
		userCredentialService,
		taskService,
		fileService,
		dictTypeService,
		dictEntryService,
		languageService,
		tenantService,
		userService,
		roleService,
		positionService,
		orgUnitService,
		menuService,
		apiService,
		permissionService,
		permissionGroupService,
		permissionAuditLogService,
		policyEvaluationLogService,
		loginAuditLogService,
		dashboardService,
		planService,
		planModuleService,
		planQuotaService,
		mfaService,
		redisCacheMonitorService,
		apiAuditLogService,
		operationAuditLogService,
		dataAccessAuditLogService,
		internalMessageService,
		internalMessageCategoryService,
		internalMessageRecipientService,
		workflowService,
		leaveService,
		expenseService,
		attendanceService,
		businessTripService,
		outingService,
		overtimeService,
		sealApplicationService,
		// register:grpc-arg ── 新模块服务实参在此行后追加(make register 工具锚点,勿删)
	)
	if err != nil {
		rollback()
		return nil, nil, err
	}

	asynqServer := server.NewAsynqServer(ctx, taskService, workflowTimeoutScheduler, attendanceScheduler)

	return newApp(ctx, grpcServer, asynqServer), rollback, nil
}
