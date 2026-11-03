package data

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	entCrud "github.com/tx7do/go-crud/entgo"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	"github.com/tx7do/go-wind-flow/flow"
	"github.com/tx7do/go-wind-flow/format/windjson"

	"go-wind-oa/app/core/service/internal/data/ent"
	"go-wind-oa/app/core/service/internal/data/ent/predicate"
	"go-wind-oa/app/core/service/internal/data/ent/workflowdefinition"
	"go-wind-oa/app/core/service/internal/data/ent/workflowinstance"
	"go-wind-oa/app/core/service/internal/data/ent/workflowlog"
	"go-wind-oa/app/core/service/internal/data/ent/workflowtask"
	"go-wind-oa/app/core/service/internal/data/ent/workflowtoken"
)

// FlowStore go-wind-flow flow.Store SPI 的 ent 适配实现。
//
// 将引擎的五类运行时数据映射到现有 oa_workflow_* 表：
//   - Definition → oa_workflow_definition（node_config 列即原始定义文本，
//     读取时经 format/windjson 解析——存量定义零迁移）；
//   - Instance   → oa_workflow_instance（Variables ↔ form_data JSON 列）；
//   - Token      → oa_workflow_token（新增表，引擎令牌状态机游标）；
//   - Task       → oa_workflow_task（补列 token_id/outcome/due_at 等）；
//   - History    → oa_workflow_log（仅映射 SUBMIT/APPROVE/REJECT/FORWARD/WITHDRAW
//     五种可展示事件，其余流转事件不落日志，与旧引擎日志粒度一致）。
//
// ID 约定：引擎全链使用字符串 ID。本实现不接管 ID 生成——引擎在 Save 前
// 由 idgen 预置的字符串 ID 落库时被丢弃，改为数据库自增分配，再把真实 ID
// 以十进制字符串回写到 flow 结构体（引擎在 Save 之后才引用各实体 ID，指针
// 传递语义下回写安全）。字符串 ID ↔ uint32 列换算见 FlowIDOf/FromFlowID。
//
// 状态映射：flow 无 SUSPENDED（父实例保持 RUNNING，令牌停在 callActivity），
// 旧表 SUSPENDED 行读取时按 RUNNING 处理，展示层经 SuspendedInstanceIDs
// 推断还原旧语义；flow ABORTED 落库按 REJECTED（fail-closed 语义与旧引擎
// 一致，proto 枚举亦无 ABORTED）。
//
// 事务：宿主在引擎入口用 WithTx 包裹，一次引擎调用内的全部落库共享一个
// 事务；ctx 中已有事务时直接复用（可嵌套）。
//
// 并发约定与引擎一致：同一实例的引擎入口调用由宿主串行化（引擎无锁；
// 事务内数据库行级写锁提供底线保护，与旧引擎的并发语义持平）。
type FlowStore struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper
}

func NewFlowStore(
	ctx *bootstrap.Context,
	entClient *entCrud.EntClient[*ent.Client],
) *FlowStore {
	return &FlowStore{
		log:       ctx.NewLoggerHelper("workflow-flow-store/repo/core-service"),
		entClient: entClient,
	}
}

// ===================== 事务 =====================

type flowTxKey struct{}

// WithTx 在事务中执行 fn。ctx 已携带事务时直接复用（嵌套安全）。
func (s *FlowStore) WithTx(ctx context.Context, fn func(context.Context) error) error {
	if tx, ok := ctx.Value(flowTxKey{}).(*ent.Tx); ok && tx != nil {
		return fn(ctx)
	}
	tx, err := s.entClient.Client().Tx(ctx)
	if err != nil {
		return fmt.Errorf("flow store: begin tx: %w", err)
	}
	defer func() {
		if v := recover(); v != nil {
			_ = tx.Rollback()
			panic(v)
		}
	}()
	if err := fn(context.WithValue(ctx, flowTxKey{}, tx)); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

// clientOf 返回当前 ctx 绑定的执行句柄（事务内返回事务作用域 client）。
func (s *FlowStore) clientOf(ctx context.Context) *ent.Client {
	if tx, ok := ctx.Value(flowTxKey{}).(*ent.Tx); ok && tx != nil {
		return tx.Client()
	}
	return s.entClient.Client()
}

// ===================== ID / 值换算 =====================

// FlowIDOf uint32 主键 → 引擎字符串 ID（十进制）。
func FlowIDOf(id uint32) string { return strconv.FormatUint(uint64(id), 10) }

// FlowIDPtrOf 可空 uint32 → 引擎字符串 ID；nil 返回空串。
func FlowIDPtrOf(id *uint32) string {
	if id == nil {
		return ""
	}
	return FlowIDOf(*id)
}

// FromFlowID 引擎字符串 ID → uint32 主键。非法/空 ID 按 ErrNotFound 处理。
func FromFlowID(id string) (uint32, error) {
	if id == "" {
		return 0, fmt.Errorf("%w: empty id", flow.ErrNotFound)
	}
	v, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("%w: invalid numeric id %q", flow.ErrNotFound, id)
	}
	return uint32(v), nil
}

// FromFlowIDOrZero 非严格换算：非法/空返回 0（用于可空外键列）。
func FromFlowIDOrZero(id string) uint32 {
	v, _ := strconv.ParseUint(id, 10, 32)
	return uint32(v)
}

// nillableID 引擎字符串 ID → 可空 uint32；空串/非法返回 nil。
func nillableID(id string) *uint32 {
	if id == "" {
		return nil
	}
	v := FromFlowIDOrZero(id)
	if v == 0 {
		return nil
	}
	return &v
}

// marshalVars 流程变量 → JSON 文本。
func marshalVars(vars flow.Vars) string {
	if len(vars) == 0 {
		return ""
	}
	b, err := json.Marshal(map[string]any(vars))
	if err != nil {
		return ""
	}
	return string(b)
}

// unmarshalVars JSON 文本 → 流程变量。
func unmarshalVars(s string) flow.Vars {
	if s == "" {
		return nil
	}
	m := map[string]any{}
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return nil
	}
	return flow.Vars(m)
}

// marshalStrings 字符串数组 → JSON 文本。
func marshalStrings(list []string) string {
	if len(list) == 0 {
		return ""
	}
	b, err := json.Marshal(list)
	if err != nil {
		return ""
	}
	return string(b)
}

// unmarshalStrings JSON 文本 → 字符串数组。
func unmarshalStrings(s string) []string {
	if s == "" {
		return nil
	}
	var list []string
	if err := json.Unmarshal([]byte(s), &list); err != nil {
		return nil
	}
	return list
}

// ===================== DefinitionStore =====================

type flowDefinitionStore struct{ s *FlowStore }

func (s *FlowStore) Definitions() flow.DefinitionStore { return flowDefinitionStore{s} }

// definitionToFlow 行 → flow.Definition。node_config 经 windjson 解析；
// 并修正 POSITION 审批人的取值字段差异（见 patchPositionApproverValues）。
func (s *FlowStore) definitionToFlow(row *ent.WorkflowDefinition) (*flow.Definition, error) {
	if row.NodeConfig == nil {
		return nil, fmt.Errorf("%w: definition node_config missing", flow.ErrInvalidDefinition)
	}
	def, err := windjson.Parse([]byte(*row.NodeConfig))
	if err != nil {
		return nil, err
	}
	def.Key = ""
	if row.Code != nil {
		def.Key = *row.Code
	}
	if row.Version != nil {
		def.Version = *row.Version
	}
	def.TenantID = FlowIDPtrOf(row.TenantID)
	def.Format = windjson.FormatName
	def.Source = []byte(*row.NodeConfig)
	if row.CreatedAt != nil {
		def.CreatedAt = *row.CreatedAt
	}
	patchPositionApproverValues(def, *row.NodeConfig)
	return def, nil
}

// srcApprover node_config 里的审批人 JSON 形状（OA 设计器契约）。
type srcApprover struct {
	Type string `json:"type"`
	ID   uint32 `json:"id,omitempty"`
}

type srcNode struct {
	ID        string        `json:"id"`
	Approvers []srcApprover `json:"approvers,omitempty"`
}

// patchPositionApproverValues POSITION 审批人取值修正。
// go-wind-oa 设计器的 POSITION 审批人把职位 ID 写在 approvers[].id，而
// windjson 导入器按 code 字段读取（对齐 go-wind-im 的格式约定）。解析后
// 对 Value 为空的 POSITION 规则回填 id 值，其余类型不动。
// 兼容 v2 图格式与旧线性数组（旧格式节点 ID 由 windjson 合成为 __legacy_node_N）。
func patchPositionApproverValues(def *flow.Definition, nodeConfig string) {
	// v2 图格式：{"version":2,"nodes":[...]}。
	var graph struct {
		Version int       `json:"version"`
		Nodes   []srcNode `json:"nodes"`
	}
	if err := json.Unmarshal([]byte(nodeConfig), &graph); err == nil && graph.Version == 2 {
		for _, nj := range graph.Nodes {
			patchNodePositions(def.Nodes[nj.ID], nj.Approvers)
		}
		return
	}

	// 旧线性数组：[...]（windjson 合成的节点 ID 与顺序一一对应）。
	var legacy []srcNode
	if err := json.Unmarshal([]byte(nodeConfig), &legacy); err != nil {
		return
	}
	for i, nj := range legacy {
		node := def.Nodes[fmt.Sprintf("__legacy_node_%d", i)]
		patchNodePositions(node, nj.Approvers)
	}
}

func patchNodePositions(node *flow.Node, approvers []srcApprover) {
	if node == nil || node.Approver == nil {
		return
	}
	var positions []string
	for _, a := range approvers {
		if strings.EqualFold(a.Type, "POSITION") {
			positions = append(positions, strconv.FormatUint(uint64(a.ID), 10))
		}
	}
	if len(positions) == 0 {
		return
	}
	idx := 0
	for i, rule := range node.Approver.Rules {
		if rule.Kind == flow.ApproverPosition && rule.Value == "" && idx < len(positions) {
			node.Approver.Rules[i].Value = positions[idx]
			idx++
		}
	}
}

func (d flowDefinitionStore) Save(ctx context.Context, def *flow.Definition) error {
	// OA 的定义由 WorkflowDefinitionRepo 管理（DRAFT→ENABLED 状态机），
	// 引擎侧 Deploy 写入仅作完备性兜底：按 (tenant, code, version) upsert。
	tid := FromFlowIDOrZero(def.TenantID)
	source := string(def.Source)
	existing, err := d.s.clientOf(ctx).WorkflowDefinition.Query().
		Where(
			workflowdefinition.TenantIDEQ(tid),
			workflowdefinition.CodeEQ(def.Key),
			workflowdefinition.VersionEQ(def.Version),
		).
		Only(ctx)
	if err != nil {
		if !ent.IsNotFound(err) {
			return err
		}
		_, err = d.s.clientOf(ctx).WorkflowDefinition.Create().
			SetTenantID(tid).
			SetCode(def.Key).
			SetVersion(def.Version).
			SetNodeConfig(source).
			SetCreatedAt(def.CreatedAt).
			SetUpdatedAt(def.CreatedAt).
			Save(ctx)
		return err
	}
	return d.s.clientOf(ctx).WorkflowDefinition.UpdateOneID(existing.ID).
		SetNodeConfig(source).
		SetUpdatedAt(def.CreatedAt).
		Exec(ctx)
}

func (d flowDefinitionStore) Get(ctx context.Context, tenantID, key string, version int) (*flow.Definition, error) {
	tid := FromFlowIDOrZero(tenantID)
	q := d.s.clientOf(ctx).WorkflowDefinition.Query().
		Where(
			workflowdefinition.TenantIDEQ(tid),
			workflowdefinition.CodeEQ(key),
		)
	if version > 0 {
		q = q.Where(workflowdefinition.VersionEQ(version))
	}
	row, err := q.Order(ent.Desc(workflowdefinition.FieldVersion)).First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("%w: definition %s v%d", flow.ErrNotFound, key, version)
		}
		return nil, err
	}
	return d.s.definitionToFlow(row)
}

func (d flowDefinitionStore) MaxVersion(ctx context.Context, tenantID, key string) (int, error) {
	tid := FromFlowIDOrZero(tenantID)
	row, err := d.s.clientOf(ctx).WorkflowDefinition.Query().
		Where(
			workflowdefinition.TenantIDEQ(tid),
			workflowdefinition.CodeEQ(key),
		).
		Order(ent.Desc(workflowdefinition.FieldVersion)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return 0, nil
		}
		return 0, err
	}
	if row.Version == nil {
		return 0, nil
	}
	return *row.Version, nil
}

func (d flowDefinitionStore) List(ctx context.Context, tenantID, key string) ([]*flow.Definition, error) {
	tid := FromFlowIDOrZero(tenantID)
	q := d.s.clientOf(ctx).WorkflowDefinition.Query().Where(workflowdefinition.TenantIDEQ(tid))
	if key != "" {
		q = q.Where(workflowdefinition.CodeEQ(key))
	}
	rows, err := q.
		Order(ent.Asc(workflowdefinition.FieldCode), ent.Asc(workflowdefinition.FieldVersion)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*flow.Definition, 0, len(rows))
	for i := range rows {
		def, err := d.s.definitionToFlow(rows[i])
		if err != nil {
			return nil, err
		}
		out = append(out, def)
	}
	return out, nil
}

// ===================== InstanceStore =====================

// instanceStatusToEnt flow 实例状态 → ent 枚举。
// RUNNING 对应 PENDING；ABORTED 按 REJECTED 落库（fail-closed，proto 枚举无 ABORTED）。
func instanceStatusToEnt(s flow.InstanceStatus) workflowinstance.InstanceStatus {
	switch s {
	case flow.InstanceApproved:
		return workflowinstance.InstanceStatusApproved
	case flow.InstanceRejected, flow.InstanceAborted:
		return workflowinstance.InstanceStatusRejected
	case flow.InstanceWithdrawn:
		return workflowinstance.InstanceStatusWithdrawn
	default: // flow.InstanceRunning
		return workflowinstance.InstanceStatusPending
	}
}

// instanceStatusToFlow ent 枚举 → flow 实例状态。SUSPENDED 兼容为 RUNNING
// （旧引擎遗留行；flow 中父实例保持 RUNNING，令牌停在 callActivity）。
func instanceStatusToFlow(s workflowinstance.InstanceStatus) flow.InstanceStatus {
	switch s {
	case workflowinstance.InstanceStatusApproved:
		return flow.InstanceApproved
	case workflowinstance.InstanceStatusRejected:
		return flow.InstanceRejected
	case workflowinstance.InstanceStatusWithdrawn:
		return flow.InstanceWithdrawn
	default: // PENDING / SUSPENDED（旧引擎遗留）
		return flow.InstanceRunning
	}
}

func (s *FlowStore) Instances() flow.InstanceStore { return flowInstanceStore{s} }

type flowInstanceStore struct{ s *FlowStore }

func instanceToFlow(row *ent.WorkflowInstance) *flow.Instance {
	inst := &flow.Instance{
		ID:          FlowIDOf(row.ID),
		TenantID:    FlowIDPtrOf(row.TenantID),
		BusinessKey: FlowIDPtrOf(row.BusinessID),
		Status:      flow.InstanceRunning,
		CreatedAt:   time.Time{},
	}
	if row.DefinitionKey != nil {
		inst.DefinitionKey = *row.DefinitionKey
	}
	if row.DefinitionVersion != nil {
		inst.DefinitionVersion = *row.DefinitionVersion
	}
	if row.BusinessType != nil {
		inst.BusinessType = *row.BusinessType
	}
	if row.InstanceStatus != nil {
		inst.Status = instanceStatusToFlow(*row.InstanceStatus)
	}
	if row.FormData != nil {
		inst.Variables = unmarshalVars(*row.FormData)
	}
	if row.ParentInstanceID != nil {
		inst.ParentInstanceID = FlowIDOf(*row.ParentInstanceID)
	}
	if row.ParentTokenID != nil {
		inst.ParentTokenID = FlowIDOf(*row.ParentTokenID)
	}
	if row.CreatedAt != nil {
		inst.CreatedAt = *row.CreatedAt
	}
	inst.EndedAt = row.EndedAt
	// StarterID 取创建人（撤回权限校验用）。
	if row.CreatedBy != nil && *row.CreatedBy != 0 {
		inst.StarterID = FlowIDOf(*row.CreatedBy)
	}
	return inst
}

func (i flowInstanceStore) Save(ctx context.Context, inst *flow.Instance) error {
	tid := FromFlowIDOrZero(inst.TenantID)
	// definition_id 反查（保持既有外键列语义：删除定义级联实例）。
	defRow, err := i.s.clientOf(ctx).WorkflowDefinition.Query().
		Where(
			workflowdefinition.TenantIDEQ(tid),
			workflowdefinition.CodeEQ(inst.DefinitionKey),
			workflowdefinition.VersionEQ(inst.DefinitionVersion),
		).
		Only(ctx)
	if err != nil {
		return fmt.Errorf("flow store: definition %s v%d not found: %w",
			inst.DefinitionKey, inst.DefinitionVersion, err)
	}

	builder := i.s.clientOf(ctx).WorkflowInstance.Create().
		SetDefinitionID(defRow.ID).
		SetInstanceStatus(instanceStatusToEnt(inst.Status)).
		SetDefinitionKey(inst.DefinitionKey).
		SetDefinitionVersion(inst.DefinitionVersion).
		SetNillableFormData(nillableString(marshalVars(inst.Variables))).
		SetTenantID(tid).
		SetCreatedAt(inst.CreatedAt).
		SetUpdatedAt(inst.CreatedAt)
	if inst.BusinessType != "" {
		builder.SetBusinessType(inst.BusinessType)
	}
	if bid := FromFlowIDOrZero(inst.BusinessKey); bid != 0 {
		builder.SetBusinessID(bid)
	}
	if pid := nillableID(inst.ParentInstanceID); pid != nil {
		builder.SetParentInstanceID(*pid)
	}
	if ptid := nillableID(inst.ParentTokenID); ptid != nil {
		builder.SetParentTokenID(*ptid)
	}
	if inst.EndedAt != nil {
		builder.SetEndedAt(*inst.EndedAt)
	}
	if sid := nillableID(inst.StarterID); sid != nil {
		builder.SetCreatedBy(*sid)
	}

	row, err := builder.Save(ctx)
	if err != nil {
		return err
	}
	// 引擎在 Save 之后才引用实例 ID（指针语义），回写数据库分配的真实 ID。
	inst.ID = FlowIDOf(row.ID)
	return nil
}

// nillableString 空串归 nil（可空字符串列）。
func nillableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func (i flowInstanceStore) Update(ctx context.Context, inst *flow.Instance) error {
	id, err := FromFlowID(inst.ID)
	if err != nil {
		return fmt.Errorf("instance update: %w", err)
	}
	builder := i.s.clientOf(ctx).WorkflowInstance.Update().
		Where(workflowinstance.IDEQ(id)).
		SetInstanceStatus(instanceStatusToEnt(inst.Status)).
		SetNillableFormData(nillableString(marshalVars(inst.Variables))).
		SetUpdatedAt(time.Now())
	if inst.EndedAt != nil {
		builder.SetEndedAt(*inst.EndedAt)
	} else {
		builder.ClearEndedAt()
	}
	_, err = builder.Save(ctx)
	return err
}

func (i flowInstanceStore) Get(ctx context.Context, id string) (*flow.Instance, error) {
	uid, err := FromFlowID(id)
	if err != nil {
		return nil, fmt.Errorf("instance get: %w", err)
	}
	row, err := i.s.clientOf(ctx).WorkflowInstance.Query().
		Where(workflowinstance.IDEQ(uid)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("%w: instance %s", flow.ErrNotFound, id)
		}
		return nil, err
	}
	return instanceToFlow(row), nil
}

func (i flowInstanceStore) ListByStatus(ctx context.Context, status flow.InstanceStatus) ([]*flow.Instance, error) {
	predicates := []predicate.WorkflowInstance{}
	if status == flow.InstanceRunning {
		// RUNNING 兼容旧引擎遗留的 SUSPENDED 行。
		predicates = append(predicates, workflowinstance.InstanceStatusIn(
			workflowinstance.InstanceStatusPending, workflowinstance.InstanceStatusSuspended))
	} else {
		predicates = append(predicates, workflowinstance.InstanceStatusEQ(instanceStatusToEnt(status)))
	}
	rows, err := i.s.clientOf(ctx).WorkflowInstance.Query().
		Where(predicates...).
		Order(ent.Asc(workflowinstance.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*flow.Instance, 0, len(rows))
	for idx := range rows {
		out = append(out, instanceToFlow(rows[idx]))
	}
	return out, nil
}

func (i flowInstanceStore) List(ctx context.Context, tenantID string) ([]*flow.Instance, error) {
	rows, err := i.s.clientOf(ctx).WorkflowInstance.Query().
		Where(workflowinstance.TenantIDEQ(FromFlowIDOrZero(tenantID))).
		Order(ent.Asc(workflowinstance.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*flow.Instance, 0, len(rows))
	for idx := range rows {
		out = append(out, instanceToFlow(rows[idx]))
	}
	return out, nil
}

// ===================== TokenStore =====================

// tokenStatusToEnt / tokenStatusToFlow 令牌状态映射。
func tokenStatusToEnt(s flow.TokenStatus) workflowtoken.TokenStatus {
	switch s {
	case flow.TokenWaiting:
		return workflowtoken.TokenStatusWaiting
	case flow.TokenEnded:
		return workflowtoken.TokenStatusEnded
	default:
		return workflowtoken.TokenStatusRunning
	}
}

func tokenStatusToFlow(s workflowtoken.TokenStatus) flow.TokenStatus {
	switch s {
	case workflowtoken.TokenStatusWaiting:
		return flow.TokenWaiting
	case workflowtoken.TokenStatusEnded:
		return flow.TokenEnded
	default:
		return flow.TokenRunning
	}
}

func (s *FlowStore) Tokens() flow.TokenStore { return flowTokenStore{s} }

type flowTokenStore struct{ s *FlowStore }

func tokenToFlow(row *ent.WorkflowToken) *flow.Token {
	tk := &flow.Token{
		ID:          FlowIDOf(row.ID),
		ParentID:    FlowIDPtrOf(row.ParentID),
		ChildInstID: FlowIDPtrOf(row.ChildInstanceID),
		DueAt:       row.DueAt,
		Status:      flow.TokenRunning,
		CreatedAt:   time.Time{},
	}
	if row.NodeID != nil {
		tk.NodeID = *row.NodeID
	}
	if row.TokenStatus != nil {
		tk.Status = tokenStatusToFlow(*row.TokenStatus)
	}
	if row.Locals != nil {
		tk.Locals = unmarshalVars(*row.Locals)
	}
	if row.CreatedAt != nil {
		tk.CreatedAt = *row.CreatedAt
	}
	if row.Edges.Instance != nil {
		tk.InstanceID = FlowIDOf(row.Edges.Instance.ID)
	}
	return tk
}

func (t flowTokenStore) Save(ctx context.Context, tk *flow.Token) error {
	builder := t.s.clientOf(ctx).WorkflowToken.Create().
		SetInstanceID(FromFlowIDOrZero(tk.InstanceID)).
		SetNodeID(tk.NodeID).
		SetTokenStatus(tokenStatusToEnt(tk.Status)).
		SetCreatedAt(tk.CreatedAt).
		SetUpdatedAt(tk.CreatedAt)
	if pid := nillableID(tk.ParentID); pid != nil {
		builder.SetParentID(*pid)
	}
	if cid := nillableID(tk.ChildInstID); cid != nil {
		builder.SetChildInstanceID(*cid)
	}
	if tk.DueAt != nil {
		builder.SetDueAt(*tk.DueAt)
	}
	if locals := nillableString(marshalVars(tk.Locals)); locals != nil {
		builder.SetLocals(*locals)
	}
	row, err := builder.Save(ctx)
	if err != nil {
		return err
	}
	tk.ID = FlowIDOf(row.ID)
	return nil
}

func (t flowTokenStore) Update(ctx context.Context, tk *flow.Token) error {
	id, err := FromFlowID(tk.ID)
	if err != nil {
		return fmt.Errorf("token update: %w", err)
	}
	builder := t.s.clientOf(ctx).WorkflowToken.Update().
		Where(workflowtoken.IDEQ(id)).
		SetNodeID(tk.NodeID).
		SetTokenStatus(tokenStatusToEnt(tk.Status)).
		SetUpdatedAt(time.Now())
	if pid := nillableID(tk.ParentID); pid != nil {
		builder.SetParentID(*pid)
	} else {
		builder.ClearParentID()
	}
	if cid := nillableID(tk.ChildInstID); cid != nil {
		builder.SetChildInstanceID(*cid)
	} else {
		builder.ClearChildInstanceID()
	}
	if tk.DueAt != nil {
		builder.SetDueAt(*tk.DueAt)
	} else {
		builder.ClearDueAt()
	}
	if locals := nillableString(marshalVars(tk.Locals)); locals != nil {
		builder.SetLocals(*locals)
	} else {
		builder.ClearLocals()
	}
	_, err = builder.Save(ctx)
	return err
}

func (t flowTokenStore) Delete(ctx context.Context, id string) error {
	uid, err := FromFlowID(id)
	if err != nil {
		return err
	}
	_, err = t.s.clientOf(ctx).WorkflowToken.Delete().Where(workflowtoken.IDEQ(uid)).Exec(ctx)
	return err
}

func (t flowTokenStore) Get(ctx context.Context, id string) (*flow.Token, error) {
	uid, err := FromFlowID(id)
	if err != nil {
		return nil, fmt.Errorf("token get: %w", err)
	}
	row, err := t.s.clientOf(ctx).WorkflowToken.Query().
		Where(workflowtoken.IDEQ(uid)).
		WithInstance().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("%w: token %s", flow.ErrNotFound, id)
		}
		return nil, err
	}
	return tokenToFlow(row), nil
}

func (t flowTokenStore) ListByInstance(ctx context.Context, instanceID string) ([]*flow.Token, error) {
	iid := FromFlowIDOrZero(instanceID)
	rows, err := t.s.clientOf(ctx).WorkflowToken.Query().
		Where(workflowtoken.HasInstanceWith(workflowinstance.IDEQ(iid))).
		WithInstance().
		Order(ent.Asc(workflowtoken.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*flow.Token, 0, len(rows))
	for idx := range rows {
		out = append(out, tokenToFlow(rows[idx]))
	}
	return out, nil
}

func (t flowTokenStore) ListDueTimers(ctx context.Context, now time.Time) ([]*flow.Token, error) {
	rows, err := t.s.clientOf(ctx).WorkflowToken.Query().
		Where(
			workflowtoken.TokenStatusEQ(workflowtoken.TokenStatusWaiting),
			workflowtoken.DueAtNotNil(),
			workflowtoken.DueAtLTE(now),
		).
		WithInstance().
		Order(ent.Asc(workflowtoken.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*flow.Token, 0, len(rows))
	for idx := range rows {
		out = append(out, tokenToFlow(rows[idx]))
	}
	return out, nil
}

// ===================== TaskStore =====================

// taskStatusToEnt / taskStatusToFlow 任务状态映射。
// flow 的 FORWARDED 仅是预留值（引擎转办只改 Assignee，不切状态），不会落库。
func taskStatusToEnt(s flow.TaskStatus) workflowtask.TaskStatus {
	switch s {
	case flow.TaskApproved:
		return workflowtask.TaskStatusApproved
	case flow.TaskRejected:
		return workflowtask.TaskStatusRejected
	case flow.TaskCancelled:
		return workflowtask.TaskStatusCancelled
	default: // PENDING / FORWARDED（预留值）
		return workflowtask.TaskStatusPending
	}
}

func taskStatusToFlow(s workflowtask.TaskStatus) flow.TaskStatus {
	switch s {
	case workflowtask.TaskStatusApproved:
		return flow.TaskApproved
	case workflowtask.TaskStatusRejected:
		return flow.TaskRejected
	case workflowtask.TaskStatusCancelled:
		return flow.TaskCancelled
	default:
		return flow.TaskPending
	}
}

func (s *FlowStore) Tasks() flow.TaskStore { return flowTaskStore{s} }

type flowTaskStore struct{ s *FlowStore }

func taskToFlow(row *ent.WorkflowTask) *flow.Task {
	t := &flow.Task{
		ID:        FlowIDOf(row.ID),
		Assignee:  FlowIDPtrOf(row.AssigneeUserID),
		TenantID:  FlowIDPtrOf(row.TenantID),
		TokenID:   FlowIDPtrOf(row.TokenID),
		Status:    flow.TaskPending,
		CreatedAt: time.Time{},
	}
	if row.NodeID != nil {
		t.NodeID = *row.NodeID
	}
	if row.TaskName != nil {
		t.Name = *row.TaskName
	}
	if row.TaskStatus != nil {
		t.Status = taskStatusToFlow(*row.TaskStatus)
	}
	if row.Outcome != nil {
		t.Outcome = *row.Outcome
	}
	if row.Comment != nil {
		t.Comment = *row.Comment
	}
	if row.CandidateGroups != nil {
		t.CandidateGroups = unmarshalStrings(*row.CandidateGroups)
	}
	t.DueAt = row.DueAt
	if row.CreatedAt != nil {
		t.CreatedAt = *row.CreatedAt
	}
	t.EndedAt = row.EndedAt
	if row.Edges.Instance != nil {
		t.InstanceID = FlowIDOf(row.Edges.Instance.ID)
	}
	return t
}

func (t flowTaskStore) Save(ctx context.Context, task *flow.Task) error {
	builder := t.s.clientOf(ctx).WorkflowTask.Create().
		SetInstanceID(FromFlowIDOrZero(task.InstanceID)).
		SetNodeID(task.NodeID).
		SetTokenID(FromFlowIDOrZero(task.TokenID)).
		SetTaskName(task.Name).
		SetTaskStatus(taskStatusToEnt(task.Status)).
		SetTenantID(FromFlowIDOrZero(task.TenantID)).
		SetCreatedAt(task.CreatedAt).
		SetUpdatedAt(task.CreatedAt)
	if aid := nillableID(task.Assignee); aid != nil {
		builder.SetAssigneeUserID(*aid)
	}
	if groups := nillableString(marshalStrings(task.CandidateGroups)); groups != nil {
		builder.SetCandidateGroups(*groups)
	}
	if task.DueAt != nil {
		builder.SetDueAt(*task.DueAt)
	}
	row, err := builder.Save(ctx)
	if err != nil {
		return err
	}
	task.ID = FlowIDOf(row.ID)
	return nil
}

func (t flowTaskStore) Update(ctx context.Context, task *flow.Task) error {
	id, err := FromFlowID(task.ID)
	if err != nil {
		return fmt.Errorf("task update: %w", err)
	}
	builder := t.s.clientOf(ctx).WorkflowTask.Update().
		Where(workflowtask.IDEQ(id)).
		SetTaskStatus(taskStatusToEnt(task.Status)).
		SetUpdatedAt(time.Now())
	if aid := nillableID(task.Assignee); aid != nil {
		builder.SetAssigneeUserID(*aid)
	} else {
		builder.ClearAssigneeUserID()
	}
	if outcome := nillableString(task.Outcome); outcome != nil {
		builder.SetOutcome(*outcome)
	} else {
		builder.ClearOutcome()
	}
	if comment := nillableString(task.Comment); comment != nil {
		builder.SetComment(*comment)
	} else {
		builder.ClearComment()
	}
	if groups := nillableString(marshalStrings(task.CandidateGroups)); groups != nil {
		builder.SetCandidateGroups(*groups)
	} else {
		builder.ClearCandidateGroups()
	}
	if task.DueAt != nil {
		builder.SetDueAt(*task.DueAt)
	} else {
		builder.ClearDueAt()
	}
	if task.EndedAt != nil {
		builder.SetEndedAt(*task.EndedAt)
	} else {
		builder.ClearEndedAt()
	}
	_, err = builder.Save(ctx)
	return err
}

func (t flowTaskStore) Get(ctx context.Context, id string) (*flow.Task, error) {
	uid, err := FromFlowID(id)
	if err != nil {
		return nil, fmt.Errorf("task get: %w", err)
	}
	row, err := t.s.clientOf(ctx).WorkflowTask.Query().
		Where(workflowtask.IDEQ(uid)).
		WithInstance().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("%w: task %s", flow.ErrNotFound, id)
		}
		return nil, err
	}
	return taskToFlow(row), nil
}

func (t flowTaskStore) ListByInstance(ctx context.Context, instanceID string) ([]*flow.Task, error) {
	iid := FromFlowIDOrZero(instanceID)
	rows, err := t.s.clientOf(ctx).WorkflowTask.Query().
		Where(workflowtask.HasInstanceWith(workflowinstance.IDEQ(iid))).
		Order(ent.Asc(workflowtask.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*flow.Task, 0, len(rows))
	for idx := range rows {
		out = append(out, taskToFlow(rows[idx]))
	}
	return out, nil
}

func (t flowTaskStore) ListByToken(ctx context.Context, tokenID string) ([]*flow.Task, error) {
	tid := FromFlowIDOrZero(tokenID)
	rows, err := t.s.clientOf(ctx).WorkflowTask.Query().
		Where(workflowtask.TokenIDEQ(tid)).
		Order(ent.Asc(workflowtask.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*flow.Task, 0, len(rows))
	for idx := range rows {
		out = append(out, taskToFlow(rows[idx]))
	}
	return out, nil
}

func (t flowTaskStore) ListPendingByAssignee(ctx context.Context, tenantID, assignee string) ([]*flow.Task, error) {
	rows, err := t.s.clientOf(ctx).WorkflowTask.Query().
		Where(
			workflowtask.TaskStatusEQ(workflowtask.TaskStatusPending),
			workflowtask.TenantIDEQ(FromFlowIDOrZero(tenantID)),
			workflowtask.AssigneeUserIDEQ(FromFlowIDOrZero(assignee)),
		).
		Order(ent.Asc(workflowtask.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*flow.Task, 0, len(rows))
	for idx := range rows {
		out = append(out, taskToFlow(rows[idx]))
	}
	return out, nil
}

func (t flowTaskStore) ListUnclaimedByGroups(ctx context.Context, tenantID string, groups []string) ([]*flow.Task, error) {
	rows, err := t.s.clientOf(ctx).WorkflowTask.Query().
		Where(
			workflowtask.TaskStatusEQ(workflowtask.TaskStatusPending),
			workflowtask.TenantIDEQ(FromFlowIDOrZero(tenantID)),
			workflowtask.AssigneeUserIDIsNil(),
		).
		Order(ent.Asc(workflowtask.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	want := map[string]bool{}
	for _, g := range groups {
		want[g] = true
	}
	out := make([]*flow.Task, 0, len(rows))
	for idx := range rows {
		var groupsOf []string
		if rows[idx].CandidateGroups != nil {
			groupsOf = unmarshalStrings(*rows[idx].CandidateGroups)
		}
		for _, g := range groupsOf {
			if want[g] {
				out = append(out, taskToFlow(rows[idx]))
				break
			}
		}
	}
	return out, nil
}

func (t flowTaskStore) ListDue(ctx context.Context, now time.Time) ([]*flow.Task, error) {
	rows, err := t.s.clientOf(ctx).WorkflowTask.Query().
		Where(
			workflowtask.TaskStatusEQ(workflowtask.TaskStatusPending),
			workflowtask.DueAtNotNil(),
			workflowtask.DueAtLT(now),
		).
		Order(ent.Asc(workflowtask.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*flow.Task, 0, len(rows))
	for idx := range rows {
		out = append(out, taskToFlow(rows[idx]))
	}
	return out, nil
}

// ===================== HistoryStore（→ oa_workflow_log）=====================
//
// 引擎的全部流转事件中，只有五种能映射到既有日志动作枚举，其余（令牌移动、
// 网关判定、变量变更、加签、到期广播等）不落日志——与旧引擎的日志粒度一致。
// 转办（FORWARD）日志由服务层自行写入以携带审批语境，此处跳过避免重复。

type flowHistoryStore struct{ s *FlowStore }

func (s *FlowStore) History() flow.HistoryStore { return flowHistoryStore{s} }

// appendAuditLog 落一条审批日志。actorID 无法解析时不署名（如系统自动动作）。
func (h flowHistoryStore) appendAuditLog(
	ctx context.Context, instanceID uint32, tenantID uint32,
	nodeID string, action workflowlog.LogAction, comment string, actorID string, at time.Time,
) error {
	builder := h.s.clientOf(ctx).WorkflowLog.Create().
		SetInstanceID(instanceID).
		SetNodeID(nodeID).
		SetLogAction(action).
		SetTenantID(tenantID).
		SetCreatedAt(at).
		SetUpdatedAt(at)
	if comment != "" {
		builder.SetComment(comment)
	}
	if aid := nillableID(actorID); aid != nil {
		builder.SetCreatedBy(*aid)
	}
	_, err := builder.Save(ctx)
	return err
}

func (h flowHistoryStore) Append(ctx context.Context, ev *flow.HistoryEvent) error {
	iid := FromFlowIDOrZero(ev.InstanceID)
	// 日志行需要租户列：经实例行反查（flow.HistoryEvent 不携带租户）。
	instRow, err := h.s.clientOf(ctx).WorkflowInstance.Query().
		Where(workflowinstance.IDEQ(iid)).
		Only(ctx)
	if err != nil {
		return fmt.Errorf("flow store: append history for instance %s: %w", ev.InstanceID, err)
	}
	tenantID := uint32(0)
	if instRow.TenantID != nil {
		tenantID = *instRow.TenantID
	}

	switch ev.Type {
	case flow.EventProcessStarted:
		return h.appendAuditLog(ctx, iid, tenantID, "", workflowlog.LogActionSubmit, "", ev.ActorID, ev.CreatedAt)

	case flow.EventTaskCompleted:
		outcome, _ := ev.Payload["outcome"].(string)
		comment, _ := ev.Payload["comment"].(string)
		switch outcome {
		case flow.OutcomeApprove:
			return h.appendAuditLog(ctx, iid, tenantID, ev.NodeID, workflowlog.LogActionApprove, comment, ev.ActorID, ev.CreatedAt)
		case flow.OutcomeReject:
			return h.appendAuditLog(ctx, iid, tenantID, ev.NodeID, workflowlog.LogActionReject, comment, ev.ActorID, ev.CreatedAt)
		}
		return nil

	case flow.EventProcessEnded:
		// 实例终结不落日志：REJECTED 已随 task.completed 落 REJECT 行；
		// APPROVED 无对应日志动作；WITHDRAWN（HistoryEvent 不携带状态，
		// 无法在此区分）由服务层在撤回事务内显式写 WITHDRAW 行。
		return nil

	default:
		return nil
	}
}

func (h flowHistoryStore) ListByInstance(ctx context.Context, instanceID string) ([]*flow.HistoryEvent, error) {
	iid := FromFlowIDOrZero(instanceID)
	rows, err := h.s.clientOf(ctx).WorkflowLog.Query().
		Where(workflowlog.HasInstanceWith(workflowinstance.IDEQ(iid))).
		Order(ent.Asc(workflowlog.FieldID)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*flow.HistoryEvent, 0, len(rows))
	for idx := range rows {
		row := rows[idx]
		ev := &flow.HistoryEvent{
			ID:         FlowIDOf(row.ID),
			InstanceID: instanceID,
			ActorID:    FlowIDPtrOf(row.CreatedBy),
		}
		if row.NodeID != nil {
			ev.NodeID = *row.NodeID
		}
		if row.CreatedAt != nil {
			ev.CreatedAt = *row.CreatedAt
		}
		action := workflowlog.LogActionSubmit
		if row.LogAction != nil {
			action = *row.LogAction
		}
		switch action {
		case workflowlog.LogActionSubmit:
			ev.Type = flow.EventProcessStarted
		case workflowlog.LogActionApprove:
			ev.Type = flow.EventTaskCompleted
			ev.Payload = flow.Vars{"outcome": flow.OutcomeApprove}
			if row.Comment != nil {
				ev.Payload["comment"] = *row.Comment
			}
		case workflowlog.LogActionReject:
			ev.Type = flow.EventTaskCompleted
			ev.Payload = flow.Vars{"outcome": flow.OutcomeReject}
			if row.Comment != nil {
				ev.Payload["comment"] = *row.Comment
			}
		case workflowlog.LogActionForward:
			ev.Type = flow.EventTaskForwarded
		case workflowlog.LogActionWithdraw:
			ev.Type = flow.EventProcessEnded
		}
		out = append(out, ev)
	}
	return out, nil
}

// ===================== 服务层查询辅助 =====================

// AppendAuditLog 写一条服务层补记的审批日志（转办/加签的 FORWARD、撤回的
// WITHDRAW——这三类语境在引擎流转事件中不可还原或 History 适配层刻意跳过）。
// tx-aware：ctx 携带事务时写入事务。
func (s *FlowStore) AppendAuditLog(
	ctx context.Context, tenantID, actorUserID, instanceID uint32,
	nodeID string, action workflowlog.LogAction, comment string,
) error {
	now := time.Now()
	builder := s.clientOf(ctx).WorkflowLog.Create().
		SetInstanceID(instanceID).
		SetNodeID(nodeID).
		SetLogAction(action).
		SetTenantID(tenantID).
		SetCreatedAt(now).
		SetUpdatedAt(now)
	if comment != "" {
		builder.SetComment(comment)
	}
	if actorUserID != 0 {
		builder.SetCreatedBy(actorUserID)
	}
	_, err := builder.Save(ctx)
	return err
}

// SuspendedInstanceIDs 返回挂有子流程（存在 child_instance_id 非空令牌）的实例
// ID 集合——go-wind-flow 中父实例保持 RUNNING，展示层据此推断旧语义的
// SUSPENDED 状态。
func (s *FlowStore) SuspendedInstanceIDs(ctx context.Context, instanceIDs []uint32) (map[uint32]bool, error) {
	out := make(map[uint32]bool, len(instanceIDs))
	if len(instanceIDs) == 0 {
		return out, nil
	}
	ids, err := s.entClient.Client().WorkflowToken.Query().
		Where(
			workflowtoken.ChildInstanceIDNotNil(),
			workflowtoken.HasInstanceWith(workflowinstance.IDIn(instanceIDs...)),
		).
		QueryInstance().
		IDs(ctx)
	if err != nil {
		return nil, err
	}
	for _, id := range ids {
		out[id] = true
	}
	return out, nil
}
