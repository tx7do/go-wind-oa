/**
 * 高德地图 Web JS API 加载器。
 *
 * 围栏圈选页按需运行时注入高德脚本（不写死在 index.html，不新增 npm 依赖）。
 * 单例：同会话只加载一次，重复调用复用已 resolve 的 Promise。
 * key 与 securityJsCode 从环境变量读取，缺失即拒绝（围栏页会显示"未配置"提示）。
 */

// 单例缓存：高德脚本重复注入会报错，必须全局只加载一次。
let amapPromise: Promise<any> | null = null;

/**
 * 加载高德 JS API，resolve 出 AMap 命名空间。
 * 重复调用返回同一个 Promise。
 */
export function loadAmap(): Promise<any> {
  if (amapPromise) {
    return amapPromise;
  }

  const key = import.meta.env.VITE_AMAP_KEY;
  const securityCode = import.meta.env.VITE_AMAP_SECURITY_CODE;
  if (!key || !securityCode) {
    amapPromise = Promise.reject(new Error("AMAP_NOT_CONFIGURED"));
    return amapPromise;
  }

  // v2.0 强制：脚本加载前须设置安全密钥，否则运行时报 INVALID_USER_SCODE。
  window._AMapSecurityConfig = { securityJsCode: securityCode };

  const src = `https://webapi.amap.com/maps?v=2.0&key=${encodeURIComponent(key)}&plugin=AMap.CircleEditor`;

  amapPromise = new Promise<any>((resolve, reject) => {
    // useScriptTag 由 @vueuse/core 自动导入；manual:true 表示不自动加载，由 load() 触发。
    const { load, unload } = useScriptTag(
      src,
      () => {
        // 脚本已加载并执行，window.AMap 现已可用。
        if (window.AMap) {
          resolve(window.AMap);
        } else {
          reject(new Error("AMAP_LOAD_FAILED"));
        }
      },
      { manual: true }
    );
    load().catch((e: unknown) => reject(e instanceof Error ? e : new Error("AMAP_LOAD_FAILED")));
    // 注意：不在此处 unload。脚本标签的生命周期与会话绑定；
    // 若卸载会移除 <script> 但 window.AMap 仍在，且重复注入会报错，
    // 故采用"加载一次、常驻"策略。
  });

  return amapPromise;
}
