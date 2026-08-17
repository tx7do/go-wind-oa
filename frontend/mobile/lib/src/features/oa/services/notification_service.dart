/// 通知服务（骨架）。
///
/// 后端待办（详见 docs/oa-mobile-design.md §"后端待办"）：
///  - 对接 go-wind-cms internal_message 的 SSE 推送通道（admin 竑关的
///    InternalMessagePublisher），mobile 端需新增 SSE 客户端订阅
///    /admin/v1/.../events，并将收到的 notification 事件投递到此 Stream。
///  - FCM / JPush 集成（pubspec 预留依赖，未启用）：需要服务端 push token
///    注册接口、设备令牌与用户绑定关系。
///
/// 当前实现：[notificationStream] 返回一个永不产出事件的空 Stream，
/// UI 据此展示"推送未配置"占位。
class NotificationService {
  NotificationService._();

  /// 通知事件流。骨架阶段为空 Stream。
  Stream<String> get notificationStream => const Stream.empty();
}
