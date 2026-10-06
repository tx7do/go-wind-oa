/**
 * 临时存放登录请求需要携带的验证码 Header。
 *
 * 后端登录通过 HTTP Header（X-Captcha-Id / X-Captcha-Value）校验验证码，
 * 而生成的 ApiClient 无法携带 per-request header。此处用一个模块级 FIFO
 * 队列作为旁路：由 use-auth 在调用 authLogin 前设置，transport.unary 消费队首——
 * 并发登录时 set/consume 按先进先出配对，不会互相覆盖。
 */
const pendingCaptchaHeaders: Record<string, string>[] = [];

export function setCaptchaHeaders(id: string, value: string) {
  pendingCaptchaHeaders.push({
    "X-Captcha-Id": id,
    "X-Captcha-Value": value,
  });
}

export function consumeCaptchaHeaders(): Record<string, string> | null {
  return pendingCaptchaHeaders.shift() ?? null;
}
