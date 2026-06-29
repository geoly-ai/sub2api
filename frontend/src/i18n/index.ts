import { createI18n } from 'vue-i18n'
import zhMessages from './locales/zh'

type LocaleCode = 'zh'

const DEFAULT_LOCALE: LocaleCode = 'zh'
export const INTL_LOCALE = 'zh-CN'

function isLocaleCode(value: string): value is LocaleCode {
  return value === 'zh'
}

export const i18n = createI18n({
  legacy: false,
  locale: DEFAULT_LOCALE,
  fallbackLocale: DEFAULT_LOCALE,
  messages: {
    zh: zhMessages
  },
  // 禁用 HTML 消息警告 - 引导步骤使用富文本内容（driver.js 支持 HTML）
  // 这些内容是内部定义的，不存在 XSS 风险
  warnHtmlMessage: false
})

export async function initI18n(): Promise<void> {
  i18n.global.locale.value = DEFAULT_LOCALE
  document.documentElement.setAttribute('lang', INTL_LOCALE)
}

export function getLocale(): LocaleCode {
  const current = i18n.global.locale.value
  return isLocaleCode(current) ? current : DEFAULT_LOCALE
}

export default i18n
